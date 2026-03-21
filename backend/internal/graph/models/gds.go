package models

// GDSProjection represents a graph projection in the GDS library
type GDSProjection struct {
	Name              string `json:"name" doc:"Projection name"`
	NodeCount         int64  `json:"nodeCount" doc:"Number of nodes in projection"`
	RelationshipCount int64  `json:"relationshipCount" doc:"Number of relationships in projection"`
	NodeProjection    string `json:"nodeProjection" doc:"Node projection specification"`
	RelProjection     string `json:"relationshipProjection" doc:"Relationship projection specification"`
	MemoryUsage       string `json:"memoryUsage" doc:"Memory usage of the projection"`
}

// AlgorithmRequest represents a request to run a GDS algorithm
type AlgorithmRequest struct {
	Algorithm      string                 `json:"algorithm" doc:"Algorithm name (e.g., pageRank, louvain, nodeSimilarity)"`
	Mode           string                 `json:"mode" doc:"Execution mode: stream, stats, mutate, write" enum:"stream,stats,mutate,write"`
	ProjectionName string                 `json:"projectionName" doc:"Name of the graph projection to use"`
	Config         map[string]interface{} `json:"config,omitempty" doc:"Algorithm-specific configuration"`
}

// AlgorithmInfo describes a GDS algorithm
type AlgorithmInfo struct {
	Name        string   `json:"name" doc:"Algorithm name"`
	Category    string   `json:"category" doc:"Category: centrality, community, similarity, pathfinding, embedding"`
	Modes       []string `json:"modes" doc:"Supported execution modes"`
	Description string   `json:"description" doc:"Brief description"`
}

// CreateProjectionRequest contains parameters for creating a graph projection
type CreateProjectionRequest struct {
	Name               string      `json:"name" doc:"Projection name"`
	NodeProjection     interface{} `json:"nodeProjection" doc:"Node projection (string or object)"`
	RelProjection      interface{} `json:"relationshipProjection" doc:"Relationship projection (string or object)"`
	NodeProperties     interface{} `json:"nodeProperties,omitempty" doc:"Node properties to include"`
	RelProperties      interface{} `json:"relationshipProperties,omitempty" doc:"Relationship properties to include"`
}

// VectorIndex represents a vector index in Neo4j
type VectorIndex struct {
	Name       string `json:"name" doc:"Index name"`
	Label      string `json:"label" doc:"Node label"`
	Property   string `json:"property" doc:"Indexed property"`
	Dimensions int    `json:"dimensions" doc:"Vector dimensions"`
	Similarity string `json:"similarity" doc:"Similarity function (cosine, euclidean)"`
	State      string `json:"state" doc:"Index state"`
}

// VectorSearchRequest contains parameters for vector similarity search
type VectorSearchRequest struct {
	IndexName   string    `json:"indexName" doc:"Name of the vector index"`
	QueryVector []float64 `json:"queryVector" doc:"Query vector for similarity search"`
	TopK        int       `json:"topK,omitempty" doc:"Number of results to return" default:"10"`
	MinScore    float64   `json:"minScore,omitempty" doc:"Minimum similarity score threshold"`
}

// CreateVectorIndexRequest contains parameters for creating a vector index
type CreateVectorIndexRequest struct {
	Name       string `json:"name" doc:"Index name"`
	Label      string `json:"label" doc:"Node label to index"`
	Property   string `json:"property" doc:"Property containing vectors"`
	Dimensions int    `json:"dimensions" doc:"Vector dimensions"`
	Similarity string `json:"similarity" doc:"Similarity function: cosine or euclidean" enum:"cosine,euclidean" default:"cosine"`
}
