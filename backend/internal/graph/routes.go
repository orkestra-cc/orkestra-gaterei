package graph

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/orkestra/backend/internal/graph/handlers"
)

// RegisterRoutes registers all graph database module routes
func RegisterRoutes(api huma.API, handler *handlers.GraphHandler) {
	// ========================================
	// Health & Database Management
	// ========================================

	huma.Register(api, huma.Operation{
		OperationID: "graph-health",
		Method:      http.MethodGet,
		Path:        "/v1/graph/health",
		Summary:     "Check Neo4j connectivity",
		Description: "Verifies that the Neo4j database is reachable and responsive.",
		Tags:        []string{"Graph"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, handler.HealthCheck)

	huma.Register(api, huma.Operation{
		OperationID: "list-graph-databases",
		Method:      http.MethodGet,
		Path:        "/v1/graph/databases",
		Summary:     "List available databases",
		Description: "Lists all Neo4j databases with their status information.",
		Tags:        []string{"Graph"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, handler.ListDatabases)

	huma.Register(api, huma.Operation{
		OperationID: "get-graph-schema",
		Method:      http.MethodGet,
		Path:        "/v1/graph/schema",
		Summary:     "Get database schema",
		Description: "Returns the schema of a Neo4j database including labels, relationship types, indexes, and constraints.",
		Tags:        []string{"Graph"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, handler.GetSchema)

	// ========================================
	// Query Execution
	// ========================================

	huma.Register(api, huma.Operation{
		OperationID: "execute-graph-query",
		Method:      http.MethodPost,
		Path:        "/v1/graph/query",
		Summary:     "Execute Cypher query",
		Description: "Executes a Cypher query against the specified database. Returns tabular results and extracted graph data for visualization.",
		Tags:        []string{"Graph"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, handler.ExecuteQuery)

	// ========================================
	// Node & Relationship Browsing
	// ========================================

	huma.Register(api, huma.Operation{
		OperationID: "browse-graph-nodes",
		Method:      http.MethodGet,
		Path:        "/v1/graph/nodes",
		Summary:     "Browse nodes",
		Description: "Browse nodes with optional label filtering and pagination.",
		Tags:        []string{"Graph"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, handler.BrowseNodes)

	huma.Register(api, huma.Operation{
		OperationID: "browse-graph-relationships",
		Method:      http.MethodGet,
		Path:        "/v1/graph/relationships",
		Summary:     "Browse relationships",
		Description: "Browse relationships with optional type filtering and pagination.",
		Tags:        []string{"Graph"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, handler.BrowseRelationships)

	huma.Register(api, huma.Operation{
		OperationID: "get-node-neighbors",
		Method:      http.MethodGet,
		Path:        "/v1/graph/nodes/{nodeId}/neighbors",
		Summary:     "Get node neighbors",
		Description: "Returns the neighborhood subgraph around a node for visualization. Supports configurable depth and limit.",
		Tags:        []string{"Graph"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, handler.GetNodeNeighbors)

	// ========================================
	// GDS (Graph Data Science)
	// ========================================

	huma.Register(api, huma.Operation{
		OperationID: "list-gds-projections",
		Method:      http.MethodGet,
		Path:        "/v1/graph/gds/projections",
		Summary:     "List graph projections",
		Description: "Lists all in-memory graph projections created by GDS.",
		Tags:        []string{"Graph GDS"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, handler.ListProjections)

	huma.Register(api, huma.Operation{
		OperationID: "create-gds-projection",
		Method:      http.MethodPost,
		Path:        "/v1/graph/gds/projections",
		Summary:     "Create graph projection",
		Description: "Creates a new in-memory graph projection for GDS algorithm execution.",
		Tags:        []string{"Graph GDS"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, handler.CreateProjection)

	huma.Register(api, huma.Operation{
		OperationID: "drop-gds-projection",
		Method:      http.MethodDelete,
		Path:        "/v1/graph/gds/projections/{name}",
		Summary:     "Drop graph projection",
		Description: "Drops an in-memory graph projection, freeing its resources.",
		Tags:        []string{"Graph GDS"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, handler.DropProjection)

	huma.Register(api, huma.Operation{
		OperationID: "run-gds-algorithm",
		Method:      http.MethodPost,
		Path:        "/v1/graph/gds/algorithms",
		Summary:     "Run GDS algorithm",
		Description: "Runs a GDS algorithm (e.g., PageRank, Louvain, Node Similarity) on a graph projection.",
		Tags:        []string{"Graph GDS"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, handler.RunAlgorithm)

	huma.Register(api, huma.Operation{
		OperationID: "list-gds-algorithms",
		Method:      http.MethodGet,
		Path:        "/v1/graph/gds/algorithms",
		Summary:     "List available algorithms",
		Description: "Returns a curated list of supported GDS algorithms with their categories and modes.",
		Tags:        []string{"Graph GDS"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, handler.ListAlgorithms)

	// ========================================
	// Vector Search
	// ========================================

	huma.Register(api, huma.Operation{
		OperationID: "vector-search",
		Method:      http.MethodPost,
		Path:        "/v1/graph/vector/search",
		Summary:     "Vector similarity search",
		Description: "Performs vector similarity search using a Neo4j vector index.",
		Tags:        []string{"Graph Vector"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, handler.VectorSearch)

	huma.Register(api, huma.Operation{
		OperationID: "list-vector-indexes",
		Method:      http.MethodGet,
		Path:        "/v1/graph/vector/indexes",
		Summary:     "List vector indexes",
		Description: "Lists all vector indexes in the database.",
		Tags:        []string{"Graph Vector"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, handler.ListVectorIndexes)

	huma.Register(api, huma.Operation{
		OperationID: "create-vector-index",
		Method:      http.MethodPost,
		Path:        "/v1/graph/vector/indexes",
		Summary:     "Create vector index",
		Description: "Creates a new vector index for similarity search.",
		Tags:        []string{"Graph Vector"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, handler.CreateVectorIndex)

	huma.Register(api, huma.Operation{
		OperationID: "drop-vector-index",
		Method:      http.MethodDelete,
		Path:        "/v1/graph/vector/indexes/{name}",
		Summary:     "Drop vector index",
		Description: "Drops a vector index from the database.",
		Tags:        []string{"Graph Vector"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, handler.DropVectorIndex)
}
