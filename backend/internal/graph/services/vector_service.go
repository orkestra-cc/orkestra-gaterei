package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/orkestra/backend/internal/graph/models"
	"github.com/orkestra/backend/internal/graph/repository"
)

// VectorService defines the interface for vector search operations
type VectorService interface {
	VectorSearch(ctx context.Context, database string, req models.VectorSearchRequest) (*models.QueryResult, error)
	ListVectorIndexes(ctx context.Context, database string) ([]models.VectorIndex, error)
	CreateVectorIndex(ctx context.Context, database string, req models.CreateVectorIndexRequest) error
	DropVectorIndex(ctx context.Context, database, name string) error
}

type vectorService struct {
	repo   repository.GraphRepository
	logger *slog.Logger
}

// NewVectorService creates a new VectorService
func NewVectorService(repo repository.GraphRepository, logger *slog.Logger) VectorService {
	return &vectorService{
		repo:   repo,
		logger: logger.With(slog.String("module", "graph-vector")),
	}
}

func (s *vectorService) VectorSearch(ctx context.Context, database string, req models.VectorSearchRequest) (*models.QueryResult, error) {
	if req.IndexName == "" {
		return nil, fmt.Errorf("index name is required")
	}
	if len(req.QueryVector) == 0 {
		return nil, fmt.Errorf("query vector is required")
	}

	topK := req.TopK
	if topK <= 0 {
		topK = 10
	}
	if topK > 1000 {
		topK = 1000
	}

	cypher := "CALL db.index.vector.queryNodes($indexName, $topK, $queryVector) YIELD node, score RETURN node, score"
	params := map[string]interface{}{
		"indexName":   req.IndexName,
		"topK":       topK,
		"queryVector": req.QueryVector,
	}

	s.logger.Info("executing vector search",
		slog.String("index", req.IndexName),
		slog.Int("topK", topK),
		slog.Int("vectorDimensions", len(req.QueryVector)),
	)

	result, err := s.repo.ExecuteRead(ctx, database, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Filter by minimum score if specified
	if req.MinScore > 0 && result.Rows != nil {
		filtered := make([]map[string]interface{}, 0)
		for _, row := range result.Rows {
			if score, ok := row["score"].(float64); ok && score >= req.MinScore {
				filtered = append(filtered, row)
			}
		}
		result.Rows = filtered
		result.Metadata.ResultCount = len(filtered)
	}

	return result, nil
}

func (s *vectorService) ListVectorIndexes(ctx context.Context, database string) ([]models.VectorIndex, error) {
	cypher := `SHOW INDEXES WHERE type = 'VECTOR'`
	result, err := s.repo.ExecuteRead(ctx, database, cypher, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list vector indexes: %w", err)
	}

	var indexes []models.VectorIndex
	for _, row := range result.Rows {
		idx := models.VectorIndex{}
		if v, ok := row["name"].(string); ok {
			idx.Name = v
		}
		if v, ok := row["state"].(string); ok {
			idx.State = v
		}
		if v, ok := row["labelsOrTypes"]; ok {
			if labels, ok := v.([]interface{}); ok && len(labels) > 0 {
				idx.Label, _ = labels[0].(string)
			}
		}
		if v, ok := row["properties"]; ok {
			if props, ok := v.([]interface{}); ok && len(props) > 0 {
				idx.Property, _ = props[0].(string)
			}
		}
		// Extract dimensions and similarity from indexConfig if available
		if v, ok := row["indexConfig"].(map[string]interface{}); ok {
			if dim, ok := v["vector.dimensions"].(int64); ok {
				idx.Dimensions = int(dim)
			}
			if sim, ok := v["vector.similarity_function"].(string); ok {
				idx.Similarity = sim
			}
		}
		indexes = append(indexes, idx)
	}

	return indexes, nil
}

func (s *vectorService) CreateVectorIndex(ctx context.Context, database string, req models.CreateVectorIndexRequest) error {
	if req.Name == "" || req.Label == "" || req.Property == "" || req.Dimensions <= 0 {
		return fmt.Errorf("name, label, property, and dimensions are required")
	}

	similarity := req.Similarity
	if similarity == "" {
		similarity = "cosine"
	}

	cypher := fmt.Sprintf(
		"CREATE VECTOR INDEX `%s` FOR (n:`%s`) ON (n.`%s`) OPTIONS {indexConfig: {`vector.dimensions`: %d, `vector.similarity_function`: '%s'}}",
		req.Name, req.Label, req.Property, req.Dimensions, similarity,
	)

	s.logger.Info("creating vector index",
		slog.String("name", req.Name),
		slog.String("label", req.Label),
		slog.String("property", req.Property),
		slog.Int("dimensions", req.Dimensions),
		slog.String("similarity", similarity),
	)

	_, err := s.repo.ExecuteWrite(ctx, database, cypher, nil)
	if err != nil {
		return fmt.Errorf("failed to create vector index: %w", err)
	}
	return nil
}

func (s *vectorService) DropVectorIndex(ctx context.Context, database, name string) error {
	if name == "" {
		return fmt.Errorf("index name is required")
	}

	cypher := fmt.Sprintf("DROP INDEX `%s`", name)

	s.logger.Info("dropping vector index", slog.String("name", name))

	_, err := s.repo.ExecuteWrite(ctx, database, cypher, nil)
	if err != nil {
		return fmt.Errorf("failed to drop vector index: %w", err)
	}
	return nil
}
