package handlers

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/orkestra/backend/internal/graph/models"
	"github.com/orkestra/backend/internal/graph/services"
)

// GraphHandler handles all graph-related HTTP requests
type GraphHandler struct {
	graphService  services.GraphService
	gdsService    services.GDSService
	vectorService services.VectorService
	neo4jURI      string
}

// NewGraphHandler creates a new GraphHandler
func NewGraphHandler(graphSvc services.GraphService, gdsSvc services.GDSService, vectorSvc services.VectorService, neo4jURI string) *GraphHandler {
	return &GraphHandler{
		graphService:  graphSvc,
		gdsService:    gdsSvc,
		vectorService: vectorSvc,
		neo4jURI:      neo4jURI,
	}
}

// --- Core Endpoints ---

func (h *GraphHandler) HealthCheck(ctx context.Context, req *struct{}) (*models.HealthCheckResponse, error) {
	if err := h.graphService.HealthCheck(ctx); err != nil {
		return nil, huma.Error503ServiceUnavailable("Neo4j is not reachable", err)
	}
	resp := &models.HealthCheckResponse{}
	resp.Body.Status = "connected"
	resp.Body.URI = h.neo4jURI
	return resp, nil
}

func (h *GraphHandler) ListDatabases(ctx context.Context, req *struct{}) (*models.ListDatabasesResponse, error) {
	databases, err := h.graphService.ListDatabases(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to list databases", err)
	}
	resp := &models.ListDatabasesResponse{}
	resp.Body.Databases = databases
	return resp, nil
}

func (h *GraphHandler) GetSchema(ctx context.Context, req *models.GetSchemaRequest) (*models.GetSchemaResponse, error) {
	schema, err := h.graphService.GetSchema(ctx, req.Database)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get schema", err)
	}
	return &models.GetSchemaResponse{Body: *schema}, nil
}

func (h *GraphHandler) ExecuteQuery(ctx context.Context, req *models.ExecuteQueryRequest) (*models.ExecuteQueryResponse, error) {
	result, err := h.graphService.ExecuteQuery(ctx, req.Body.Database, req.Body.Cypher, req.Body.Params, req.Body.ReadOnly)
	if err != nil {
		return nil, huma.Error400BadRequest("Query execution failed", err)
	}
	return &models.ExecuteQueryResponse{Body: *result}, nil
}

func (h *GraphHandler) BrowseNodes(ctx context.Context, req *models.BrowseNodesRequest) (*models.BrowseNodesResponse, error) {
	result, err := h.graphService.BrowseNodes(ctx, req.Database, req.Labels, req.Limit, req.Skip)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to browse nodes", err)
	}
	return &models.BrowseNodesResponse{Body: *result}, nil
}

func (h *GraphHandler) BrowseRelationships(ctx context.Context, req *models.BrowseRelationshipsRequest) (*models.BrowseRelationshipsResponse, error) {
	result, err := h.graphService.BrowseRelationships(ctx, req.Database, req.Types, req.Limit, req.Skip)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to browse relationships", err)
	}
	return &models.BrowseRelationshipsResponse{Body: *result}, nil
}

func (h *GraphHandler) GetNodeNeighbors(ctx context.Context, req *models.GetNodeNeighborsRequest) (*models.GetNodeNeighborsResponse, error) {
	graphData, err := h.graphService.GetNodeNeighbors(ctx, req.Database, req.NodeID, req.Depth, req.Limit)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get node neighbors", err)
	}
	return &models.GetNodeNeighborsResponse{Body: *graphData}, nil
}

// --- GDS Endpoints ---

func (h *GraphHandler) ListProjections(ctx context.Context, req *models.ListProjectionsRequest) (*models.ListProjectionsResponse, error) {
	projections, err := h.gdsService.ListProjections(ctx, req.Database)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to list projections", err)
	}
	resp := &models.ListProjectionsResponse{}
	resp.Body.Projections = projections
	return resp, nil
}

func (h *GraphHandler) CreateProjection(ctx context.Context, req *models.CreateProjectionRequestDTO) (*models.CreateProjectionResponse, error) {
	projection, err := h.gdsService.CreateProjection(ctx, req.Body.Database, req.Body.CreateProjectionRequest)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to create projection", err)
	}
	return &models.CreateProjectionResponse{Body: *projection}, nil
}

func (h *GraphHandler) DropProjection(ctx context.Context, req *models.DropProjectionRequest) (*models.DropProjectionResponse, error) {
	if err := h.gdsService.DropProjection(ctx, req.Database, req.Name); err != nil {
		return nil, huma.Error400BadRequest("Failed to drop projection", err)
	}
	resp := &models.DropProjectionResponse{}
	resp.Body.Message = "Projection dropped successfully"
	return resp, nil
}

func (h *GraphHandler) RunAlgorithm(ctx context.Context, req *models.RunAlgorithmRequestDTO) (*models.RunAlgorithmResponse, error) {
	result, err := h.gdsService.RunAlgorithm(ctx, req.Body.Database, req.Body.AlgorithmRequest)
	if err != nil {
		return nil, huma.Error400BadRequest("Algorithm execution failed", err)
	}
	return &models.RunAlgorithmResponse{Body: *result}, nil
}

func (h *GraphHandler) ListAlgorithms(ctx context.Context, req *struct{}) (*models.ListAlgorithmsResponse, error) {
	algorithms := h.gdsService.ListAlgorithms(ctx)
	resp := &models.ListAlgorithmsResponse{}
	resp.Body.Algorithms = algorithms
	return resp, nil
}

// --- Vector Endpoints ---

func (h *GraphHandler) VectorSearch(ctx context.Context, req *models.VectorSearchRequestDTO) (*models.VectorSearchResponse, error) {
	result, err := h.vectorService.VectorSearch(ctx, req.Body.Database, req.Body.VectorSearchRequest)
	if err != nil {
		return nil, huma.Error400BadRequest("Vector search failed", err)
	}
	return &models.VectorSearchResponse{Body: *result}, nil
}

func (h *GraphHandler) ListVectorIndexes(ctx context.Context, req *models.ListVectorIndexesRequest) (*models.ListVectorIndexesResponse, error) {
	indexes, err := h.vectorService.ListVectorIndexes(ctx, req.Database)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to list vector indexes", err)
	}
	resp := &models.ListVectorIndexesResponse{}
	resp.Body.Indexes = indexes
	return resp, nil
}

func (h *GraphHandler) CreateVectorIndex(ctx context.Context, req *models.CreateVectorIndexRequestDTO) (*models.CreateVectorIndexResponse, error) {
	if err := h.vectorService.CreateVectorIndex(ctx, req.Body.Database, req.Body.CreateVectorIndexRequest); err != nil {
		return nil, huma.Error400BadRequest("Failed to create vector index", err)
	}
	resp := &models.CreateVectorIndexResponse{}
	resp.Body.Message = "Vector index created successfully"
	return resp, nil
}

func (h *GraphHandler) DropVectorIndex(ctx context.Context, req *models.DropVectorIndexRequest) (*models.DropVectorIndexResponse, error) {
	if err := h.vectorService.DropVectorIndex(ctx, req.Database, req.Name); err != nil {
		return nil, huma.Error400BadRequest("Failed to drop vector index", err)
	}
	resp := &models.DropVectorIndexResponse{}
	resp.Body.Message = "Vector index dropped successfully"
	return resp, nil
}
