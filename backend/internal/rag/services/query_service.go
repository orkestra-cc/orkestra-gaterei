package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	graphRepo "github.com/orkestra/backend/internal/graph/repository"
	"github.com/orkestra/backend/internal/rag/models"
	"github.com/orkestra/backend/internal/rag/providers"
)

// QueryService handles RAG query execution
type QueryService interface {
	Query(ctx context.Context, question string, topK int, minScore float64, isoStandard string, llmOverrideUUID string) (*models.RAGQueryResponse, error)
}

type queryService struct {
	graphRepo    graphRepo.GraphRepository
	modelService ModelService
	defaultTopK  int
	logger       *slog.Logger
}

// NewQueryService creates a new QueryService
func NewQueryService(gr graphRepo.GraphRepository, modelSvc ModelService, defaultTopK int, logger *slog.Logger) QueryService {
	return &queryService{
		graphRepo:    gr,
		modelService: modelSvc,
		defaultTopK:  defaultTopK,
		logger:       logger.With(slog.String("module", "rag-query")),
	}
}

func (s *queryService) Query(ctx context.Context, question string, topK int, minScore float64, isoStandard string, llmOverrideUUID string) (*models.RAGQueryResponse, error) {
	totalStart := time.Now()

	if topK <= 0 {
		topK = s.defaultTopK
	}
	if minScore <= 0 {
		minScore = 0.3
	}

	// Step 1: Embed the question
	embStart := time.Now()
	embProvider, err := s.modelService.GetDefaultEmbeddingProvider(ctx)
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
		if isoStandard != "" {
			docInfo := s.getDocumentInfo(ctx, src.DocumentUUID)
			if docInfo != nil {
				src.DocumentTitle = docInfo.title
				src.ISOStandard = docInfo.isoStandard
				if src.ISOStandard != isoStandard {
					continue
				}
			}
		} else {
			docInfo := s.getDocumentInfo(ctx, src.DocumentUUID)
			if docInfo != nil {
				src.DocumentTitle = docInfo.title
				src.ISOStandard = docInfo.isoStandard
			}
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

	if len(sources) == 0 {
		resp := &models.RAGQueryResponse{}
		resp.Body.Answer = "No relevant information found in the knowledge base for your question."
		resp.Body.Sources = []models.SourceRef{}
		resp.Body.Metadata = models.QueryMeta{
			EmbeddingTimeMs: embTimeMs,
			SearchTimeMs:    searchTimeMs,
			TotalTimeMs:     time.Since(totalStart).Milliseconds(),
		}
		return resp, nil
	}

	// Step 3: Build prompt and generate answer
	llmStart := time.Now()
	var llmProvider providers.LLMProvider
	if llmOverrideUUID != "" {
		model, err := s.modelService.GetModel(ctx, llmOverrideUUID)
		if err == nil {
			llmProvider, _ = providers.NewLLMProvider(providers.ModelConfig{
				Provider:  model.Provider,
				ModelName: model.ModelName,
				BaseURL:   model.BaseURL,
				APIKey:    model.APIKey,
			})
		}
	}
	if llmProvider == nil {
		llmProvider, err = s.modelService.GetDefaultLLMProvider(ctx)
		if err != nil {
			return nil, fmt.Errorf("no LLM model: %w", err)
		}
	}

	contextStr := strings.Join(contextParts, "\n---\n")
	prompt := fmt.Sprintf("CONTEXT:\n%s\n\nQUESTION: %s\n\nANSWER:", contextStr, question)

	systemPrompt := `You are an ISO compliance expert assistant. Answer the question based ONLY on the provided context.
If the context does not contain enough information to answer, say so clearly.
Always cite the source document and section when referencing specific requirements.
Be precise and actionable in your responses.
/no_think`

	// Use a dedicated context with longer timeout for LLM generation
	llmCtx, llmCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer llmCancel()

	answer, err := llmProvider.Complete(llmCtx, prompt, providers.CompletionOptions{
		SystemPrompt: systemPrompt,
		Temperature:  0.1,
		MaxTokens:    2048,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}
	llmTimeMs := time.Since(llmStart).Milliseconds()

	resp := &models.RAGQueryResponse{}
	resp.Body.Answer = answer
	resp.Body.Sources = sources
	resp.Body.Metadata = models.QueryMeta{
		EmbeddingTimeMs: embTimeMs,
		SearchTimeMs:    searchTimeMs,
		LLMTimeMs:       llmTimeMs,
		TotalTimeMs:     time.Since(totalStart).Milliseconds(),
		ChunksRetrieved: len(sources),
		ModelUsed:       llmProvider.ModelName(),
	}

	s.logger.Info("RAG query completed",
		slog.Int64("totalMs", resp.Body.Metadata.TotalTimeMs),
		slog.Int("chunks", len(sources)),
	)

	return resp, nil
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
