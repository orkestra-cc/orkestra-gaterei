package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/orkestra/backend/internal/graph/models"
	"github.com/orkestra/backend/internal/graph/repository"
)

// GDSService defines the interface for Graph Data Science operations
type GDSService interface {
	ListProjections(ctx context.Context, database string) ([]models.GDSProjection, error)
	CreateProjection(ctx context.Context, database string, req models.CreateProjectionRequest) (*models.GDSProjection, error)
	DropProjection(ctx context.Context, database, name string) error
	RunAlgorithm(ctx context.Context, database string, req models.AlgorithmRequest) (*models.QueryResult, error)
	ListAlgorithms(ctx context.Context) []models.AlgorithmInfo
}

type gdsService struct {
	repo   repository.GraphRepository
	logger *slog.Logger
}

// NewGDSService creates a new GDSService
func NewGDSService(repo repository.GraphRepository, logger *slog.Logger) GDSService {
	return &gdsService{
		repo:   repo,
		logger: logger.With(slog.String("module", "graph-gds")),
	}
}

func (s *gdsService) ListProjections(ctx context.Context, database string) ([]models.GDSProjection, error) {
	cypher := "CALL gds.graph.list() YIELD graphName, nodeCount, relationshipCount, nodeProjection, relationshipProjection, memoryUsage RETURN graphName, nodeCount, relationshipCount, nodeProjection, relationshipProjection, memoryUsage"
	result, err := s.repo.ExecuteRead(ctx, database, cypher, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list projections (is GDS installed?): %w", err)
	}

	var projections []models.GDSProjection
	for _, row := range result.Rows {
		p := models.GDSProjection{}
		if v, ok := row["graphName"].(string); ok {
			p.Name = v
		}
		if v, ok := row["nodeCount"].(int64); ok {
			p.NodeCount = v
		}
		if v, ok := row["relationshipCount"].(int64); ok {
			p.RelationshipCount = v
		}
		if v, ok := row["nodeProjection"].(string); ok {
			p.NodeProjection = v
		}
		if v, ok := row["relationshipProjection"].(string); ok {
			p.RelProjection = v
		}
		if v, ok := row["memoryUsage"].(string); ok {
			p.MemoryUsage = v
		}
		projections = append(projections, p)
	}

	return projections, nil
}

func (s *gdsService) CreateProjection(ctx context.Context, database string, req models.CreateProjectionRequest) (*models.GDSProjection, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("projection name is required")
	}

	cypher := "CALL gds.graph.project($name, $nodeProjection, $relProjection)"
	params := map[string]interface{}{
		"name":           req.Name,
		"nodeProjection": req.NodeProjection,
		"relProjection":  req.RelProjection,
	}

	s.logger.Info("creating graph projection",
		slog.String("name", req.Name),
		slog.String("database", database),
	)

	result, err := s.repo.ExecuteWrite(ctx, database, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create projection: %w", err)
	}

	p := &models.GDSProjection{Name: req.Name}
	if len(result.Rows) > 0 {
		row := result.Rows[0]
		if v, ok := row["nodeCount"].(int64); ok {
			p.NodeCount = v
		}
		if v, ok := row["relationshipCount"].(int64); ok {
			p.RelationshipCount = v
		}
	}
	return p, nil
}

func (s *gdsService) DropProjection(ctx context.Context, database, name string) error {
	if name == "" {
		return fmt.Errorf("projection name is required")
	}

	cypher := "CALL gds.graph.drop($name)"
	params := map[string]interface{}{"name": name}

	s.logger.Info("dropping graph projection",
		slog.String("name", name),
		slog.String("database", database),
	)

	_, err := s.repo.ExecuteWrite(ctx, database, cypher, params)
	if err != nil {
		return fmt.Errorf("failed to drop projection: %w", err)
	}
	return nil
}

func (s *gdsService) RunAlgorithm(ctx context.Context, database string, req models.AlgorithmRequest) (*models.QueryResult, error) {
	if req.Algorithm == "" {
		return nil, fmt.Errorf("algorithm name is required")
	}
	if req.ProjectionName == "" {
		return nil, fmt.Errorf("projection name is required")
	}

	// Default mode is stream
	mode := req.Mode
	if mode == "" {
		mode = "stream"
	}

	// Validate mode
	switch mode {
	case "stream", "stats", "mutate", "write":
		// valid
	default:
		return nil, fmt.Errorf("invalid mode: %s (must be stream, stats, mutate, or write)", mode)
	}

	// Build the algorithm call
	cypher := fmt.Sprintf("CALL gds.%s.%s($projection, $config)", req.Algorithm, mode)
	config := req.Config
	if config == nil {
		config = map[string]interface{}{}
	}
	params := map[string]interface{}{
		"projection": req.ProjectionName,
		"config":     config,
	}

	s.logger.Info("running GDS algorithm",
		slog.String("algorithm", req.Algorithm),
		slog.String("mode", mode),
		slog.String("projection", req.ProjectionName),
	)

	// Stream and stats are read operations, mutate and write are write operations
	if mode == "stream" || mode == "stats" {
		return s.repo.ExecuteRead(ctx, database, cypher, params)
	}
	return s.repo.ExecuteWrite(ctx, database, cypher, params)
}

func (s *gdsService) ListAlgorithms(ctx context.Context) []models.AlgorithmInfo {
	return []models.AlgorithmInfo{
		// Centrality
		{Name: "pageRank", Category: "centrality", Modes: []string{"stream", "stats", "mutate", "write"}, Description: "Measures the importance of nodes based on incoming relationships"},
		{Name: "betweennessCentrality", Category: "centrality", Modes: []string{"stream", "stats", "mutate", "write"}, Description: "Measures how often a node lies on shortest paths between other nodes"},
		{Name: "degreeCentrality", Category: "centrality", Modes: []string{"stream", "stats", "mutate", "write"}, Description: "Measures the number of relationships a node has"},
		{Name: "closeness", Category: "centrality", Modes: []string{"stream", "stats", "mutate", "write"}, Description: "Measures how close a node is to all other nodes"},
		{Name: "articleRank", Category: "centrality", Modes: []string{"stream", "stats", "mutate", "write"}, Description: "Variant of PageRank that reduces influence of low-degree nodes"},
		{Name: "eigenvector", Category: "centrality", Modes: []string{"stream", "stats", "mutate", "write"}, Description: "Measures the influence of a node based on the influence of its neighbors"},
		// Community Detection
		{Name: "louvain", Category: "community", Modes: []string{"stream", "stats", "mutate", "write"}, Description: "Detects communities using modularity optimization"},
		{Name: "labelPropagation", Category: "community", Modes: []string{"stream", "stats", "mutate", "write"}, Description: "Detects communities by propagating labels through the network"},
		{Name: "wcc", Category: "community", Modes: []string{"stream", "stats", "mutate", "write"}, Description: "Finds weakly connected components"},
		{Name: "scc", Category: "community", Modes: []string{"stream", "stats", "mutate", "write"}, Description: "Finds strongly connected components"},
		{Name: "triangleCount", Category: "community", Modes: []string{"stream", "stats", "mutate", "write"}, Description: "Counts the number of triangles each node is part of"},
		{Name: "localClusteringCoefficient", Category: "community", Modes: []string{"stream", "stats", "mutate", "write"}, Description: "Measures how connected a node's neighbors are"},
		// Similarity
		{Name: "nodeSimilarity", Category: "similarity", Modes: []string{"stream", "stats", "mutate", "write"}, Description: "Computes similarity between nodes based on shared neighbors"},
		{Name: "knn", Category: "similarity", Modes: []string{"stream", "stats", "mutate", "write"}, Description: "Computes k-nearest neighbors based on node properties"},
		// Path Finding
		{Name: "shortestPath.dijkstra", Category: "pathfinding", Modes: []string{"stream"}, Description: "Finds the shortest weighted path between two nodes"},
		{Name: "shortestPath.astar", Category: "pathfinding", Modes: []string{"stream"}, Description: "Finds the shortest path using A* algorithm"},
		{Name: "allShortestPaths.dijkstra", Category: "pathfinding", Modes: []string{"stream"}, Description: "Finds all shortest paths between two nodes"},
		// Embeddings
		{Name: "fastRP", Category: "embedding", Modes: []string{"stream", "stats", "mutate", "write"}, Description: "Generates node embeddings using Fast Random Projection"},
		{Name: "node2vec", Category: "embedding", Modes: []string{"stream", "mutate", "write"}, Description: "Generates node embeddings using Node2Vec random walks"},
		{Name: "hashgnn", Category: "embedding", Modes: []string{"stream", "mutate"}, Description: "Generates node embeddings using HashGNN"},
	}
}
