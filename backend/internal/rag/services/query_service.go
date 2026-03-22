package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	aimodelsProviders "github.com/orkestra/backend/internal/aimodels/providers"
	graphRepo "github.com/orkestra/backend/internal/graph/repository"
	"github.com/orkestra/backend/internal/rag/models"
)

const ragSystemPrompt = `You are an ISO compliance expert assistant. Answer the question based ONLY on the provided context.
If the context does not contain enough information to answer, say so clearly.
Always cite the source document and section when referencing specific requirements.
Be precise and actionable in your responses.
/no_think`

// StreamResult holds the pre-LLM data and token channel for streaming queries
type StreamResult struct {
	Sources   []models.SourceRef
	PreMeta   models.QueryMeta
	TokenChan <-chan aimodelsProviders.StreamChunk
	ModelName string
	StartTime time.Time // total query start for final metadata
	LLMStart  time.Time // LLM start for llmTimeMs
}

// QueryService handles RAG query execution
type QueryService interface {
	Query(ctx context.Context, question string, topK int, minScore float64, isoStandard string, llmOverrideUUID string) (*models.RAGQueryResponse, error)
	QueryStream(ctx context.Context, question string, topK int, minScore float64, isoStandard string, llmOverrideUUID string) (*StreamResult, error)
}

type queryService struct {
	graphRepo     graphRepo.GraphRepository
	modelProvider AIModelProvider
	defaultTopK   int
	logger        *slog.Logger
}

// NewQueryService creates a new QueryService
func NewQueryService(gr graphRepo.GraphRepository, modelProvider AIModelProvider, defaultTopK int, logger *slog.Logger) QueryService {
	return &queryService{
		graphRepo:     gr,
		modelProvider: modelProvider,
		defaultTopK:   defaultTopK,
		logger:        logger.With(slog.String("module", "rag-query")),
	}
}

// preparedContext holds the results of embedding + vector search + context expansion
type preparedContext struct {
	sources      []models.SourceRef
	contextStr   string
	prompt       string
	embTimeMs    int64
	searchTimeMs int64
	llmProvider  aimodelsProviders.LLMProvider
}

// prepareContext performs embedding, vector search, context expansion, and LLM provider resolution.
// Shared by both Query and QueryStream.
func (s *queryService) prepareContext(ctx context.Context, question string, topK int, minScore float64, isoStandard string, llmOverrideUUID string) (*preparedContext, error) {
	if topK <= 0 {
		topK = s.defaultTopK
	}
	if minScore <= 0 {
		minScore = 0.3
	}

	// Step 1: Embed the question
	embStart := time.Now()
	embProvider, err := s.modelProvider.GetDefaultEmbeddingProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("no embedding model: %w", err)
	}

	queryVector, err := embProvider.Embed(ctx, question)
	if err != nil {
		return nil, fmt.Errorf("embedding failed: %w", err)
	}
	embTimeMs := time.Since(embStart).Milliseconds()

	// Step 2: Vector search
	searchStart := time.Now()
	searchResult, err := s.graphRepo.ExecuteRead(ctx, "", `
		CALL vector_search.search('rag_chunk_embedding', $topK, $queryVector)
		YIELD node, similarity
		WITH node, similarity
		WHERE similarity >= $minScore
		RETURN node.uuid AS chunkUuid, node.text AS text,
		       node.sectionTitle AS sectionTitle, node.position AS position,
		       node.documentUuid AS documentUuid, similarity
		ORDER BY similarity DESC
	`, map[string]interface{}{
		"topK":        topK,
		"queryVector": queryVector,
		"minScore":    minScore,
	})
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	searchTimeMs := time.Since(searchStart).Milliseconds()

	// Build source refs with document metadata
	var sources []models.SourceRef
	var contextParts []string

	for _, row := range searchResult.Rows {
		src := models.SourceRef{}
		if v, ok := row["chunkUuid"].(string); ok {
			src.ChunkUUID = v
		}
		if v, ok := row["text"].(string); ok {
			src.ChunkText = v
		}
		if v, ok := row["sectionTitle"].(string); ok {
			src.SectionTitle = v
		}
		if v, ok := row["position"].(int64); ok {
			src.Position = int(v)
		}
		if v, ok := row["documentUuid"].(string); ok {
			src.DocumentUUID = v
		}
		if v, ok := row["similarity"].(float64); ok {
			src.Score = v
		}

		// Filter by ISO standard if specified
		docInfo := s.getDocumentInfo(ctx, src.DocumentUUID)
		if docInfo != nil {
			src.DocumentTitle = docInfo.title
			src.ISOStandard = docInfo.isoStandard
		}
		if isoStandard != "" && src.ISOStandard != isoStandard {
			continue
		}

		sources = append(sources, src)

		// Build context with neighboring chunks
		expandedText := s.expandContext(ctx, src.ChunkUUID, src.ChunkText)
		section := src.SectionTitle
		if section == "" {
			section = "General"
		}
		contextParts = append(contextParts, fmt.Sprintf(
			"Source: %s (%s), Section: %s\n%s",
			src.DocumentTitle, src.ISOStandard, section, expandedText,
		))
	}

	// Resolve LLM provider
	var llmProvider aimodelsProviders.LLMProvider
	if llmOverrideUUID != "" {
		llmProvider, _ = s.modelProvider.GetLLMProvider(ctx, llmOverrideUUID)
	}
	if llmProvider == nil {
		llmProvider, err = s.modelProvider.GetDefaultLLMProvider(ctx)
		if err != nil {
			return nil, fmt.Errorf("no LLM model: %w", err)
		}
	}

	contextStr := strings.Join(contextParts, "\n---\n")
	prompt := fmt.Sprintf("CONTEXT:\n%s\n\nQUESTION: %s\n\nANSWER:", contextStr, question)

	return &preparedContext{
		sources:      sources,
		contextStr:   contextStr,
		prompt:       prompt,
		embTimeMs:    embTimeMs,
		searchTimeMs: searchTimeMs,
		llmProvider:  llmProvider,
	}, nil
}

func (s *queryService) Query(ctx context.Context, question string, topK int, minScore float64, isoStandard string, llmOverrideUUID string) (*models.RAGQueryResponse, error) {
	totalStart := time.Now()

	pc, err := s.prepareContext(ctx, question, topK, minScore, isoStandard, llmOverrideUUID)
	if err != nil {
		return nil, err
	}

	if len(pc.sources) == 0 {
		resp := &models.RAGQueryResponse{}
		resp.Body.Answer = "No relevant information found in the knowledge base for your question."
		resp.Body.Sources = []models.SourceRef{}
		resp.Body.Metadata = models.QueryMeta{
			EmbeddingTimeMs: pc.embTimeMs,
			SearchTimeMs:    pc.searchTimeMs,
			TotalTimeMs:     time.Since(totalStart).Milliseconds(),
		}
		return resp, nil
	}

	// Generate answer with LLM
	llmStart := time.Now()
	llmCtx, llmCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer llmCancel()

	answer, err := pc.llmProvider.Complete(llmCtx, pc.prompt, aimodelsProviders.CompletionOptions{
		SystemPrompt: ragSystemPrompt,
		Temperature:  0.1,
		MaxTokens:    2048,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}
	llmTimeMs := time.Since(llmStart).Milliseconds()

	resp := &models.RAGQueryResponse{}
	resp.Body.Answer = answer
	resp.Body.Sources = pc.sources
	resp.Body.Metadata = models.QueryMeta{
		EmbeddingTimeMs: pc.embTimeMs,
		SearchTimeMs:    pc.searchTimeMs,
		LLMTimeMs:       llmTimeMs,
		TotalTimeMs:     time.Since(totalStart).Milliseconds(),
		ChunksRetrieved: len(pc.sources),
		ModelUsed:       pc.llmProvider.ModelName(),
	}

	s.logger.Info("RAG query completed",
		slog.Int64("totalMs", resp.Body.Metadata.TotalTimeMs),
		slog.Int("chunks", len(pc.sources)),
	)

	return resp, nil
}

func (s *queryService) QueryStream(ctx context.Context, question string, topK int, minScore float64, isoStandard string, llmOverrideUUID string) (*StreamResult, error) {
	totalStart := time.Now()

	pc, err := s.prepareContext(ctx, question, topK, minScore, isoStandard, llmOverrideUUID)
	if err != nil {
		return nil, err
	}

	if len(pc.sources) == 0 {
		// Return a result with nil channel to signal "no results" to the handler
		return &StreamResult{
			Sources: []models.SourceRef{},
			PreMeta: models.QueryMeta{
				EmbeddingTimeMs: pc.embTimeMs,
				SearchTimeMs:    pc.searchTimeMs,
				TotalTimeMs:     time.Since(totalStart).Milliseconds(),
			},
			TokenChan: nil,
		}, nil
	}

	// Start streaming LLM generation
	llmCtx, _ := context.WithTimeout(context.Background(), 5*time.Minute)
	llmStart := time.Now()

	tokenChan, err := pc.llmProvider.StreamComplete(llmCtx, pc.prompt, aimodelsProviders.CompletionOptions{
		SystemPrompt: ragSystemPrompt,
		Temperature:  0.1,
		MaxTokens:    2048,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM stream failed: %w", err)
	}

	return &StreamResult{
		Sources: pc.sources,
		PreMeta: models.QueryMeta{
			EmbeddingTimeMs: pc.embTimeMs,
			SearchTimeMs:    pc.searchTimeMs,
			ChunksRetrieved: len(pc.sources),
			ModelUsed:       pc.llmProvider.ModelName(),
		},
		TokenChan: tokenChan,
		ModelName: pc.llmProvider.ModelName(),
		StartTime: totalStart,
		LLMStart:  llmStart,
	}, nil
}

type docInfo struct {
	title       string
	isoStandard string
}

func (s *queryService) getDocumentInfo(ctx context.Context, docUUID string) *docInfo {
	result, err := s.graphRepo.ExecuteRead(ctx, "", `
		MATCH (d:RagDocument {uuid: $uuid})
		RETURN d.title AS title, d.isoStandard AS isoStandard
	`, map[string]interface{}{"uuid": docUUID})
	if err != nil || len(result.Rows) == 0 {
		return nil
	}
	info := &docInfo{}
	if v, ok := result.Rows[0]["title"].(string); ok {
		info.title = v
	}
	if v, ok := result.Rows[0]["isoStandard"].(string); ok {
		info.isoStandard = v
	}
	return info
}

func (s *queryService) expandContext(ctx context.Context, chunkUUID, chunkText string) string {
	result, err := s.graphRepo.ExecuteRead(ctx, "", `
		MATCH (c:RagChunk {uuid: $uuid})
		OPTIONAL MATCH (prev:RagChunk)-[:NEXT]->(c)
		OPTIONAL MATCH (c)-[:NEXT]->(next:RagChunk)
		RETURN prev.text AS prevText, next.text AS nextText
	`, map[string]interface{}{"uuid": chunkUUID})

	if err != nil || len(result.Rows) == 0 {
		return chunkText
	}

	var parts []string
	if v, ok := result.Rows[0]["prevText"].(string); ok && v != "" {
		parts = append(parts, v)
	}
	parts = append(parts, chunkText)
	if v, ok := result.Rows[0]["nextText"].(string); ok && v != "" {
		parts = append(parts, v)
	}
	return strings.Join(parts, "\n\n")
}
