package handlers

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"

	"github.com/orkestra/backend/internal/company/models"
	"github.com/orkestra/backend/internal/company/services"
)

// CompanyHandler handles company lookup HTTP requests
type CompanyHandler struct {
	service services.CompanyService
}

// NewCompanyHandler creates a new CompanyHandler
func NewCompanyHandler(service services.CompanyService) *CompanyHandler {
	return &CompanyHandler{
		service: service,
	}
}

// ========================================
// Request/Response Types
// ========================================

// LookupCompanyRequest represents the request to look up a company
type LookupCompanyRequest struct {
	TaxCode string `path:"taxCode" doc:"Italian tax code (Codice Fiscale) or VAT number (Partita IVA)" minLength:"11" maxLength:"16"`
}

// LookupCompanyResponse represents the response with company lookup data
type LookupCompanyResponse struct {
	Body models.CompanyLookup `json:"lookup" doc:"Company lookup result"`
}

// GetCompanyLookupRequest represents the request to get a stored lookup
type GetCompanyLookupRequest struct {
	ID string `path:"id" doc:"Company lookup UUID"`
}

// GetCompanyLookupResponse represents the response with a stored lookup
type GetCompanyLookupResponse struct {
	Body models.CompanyLookup `json:"lookup" doc:"Company lookup details"`
}

// ListCompanyLookupsRequest represents the request to list stored lookups
type ListCompanyLookupsRequest struct {
	Page     int `query:"page" default:"1" minimum:"1" doc:"Page number"`
	PageSize int `query:"pageSize" default:"20" minimum:"1" maximum:"100" doc:"Items per page"`
}

// ListCompanyLookupsResponse represents the paginated list of lookups
type ListCompanyLookupsResponse struct {
	Body models.CompanyLookupListResponse `json:"result" doc:"Paginated list of company lookups"`
}

// SearchCompanyLookupsRequest represents the request to search stored lookups
type SearchCompanyLookupsRequest struct {
	Query    string `query:"q" doc:"Search query (company name, tax code, or VAT code)"`
	Page     int    `query:"page" default:"1" minimum:"1" doc:"Page number"`
	PageSize int    `query:"pageSize" default:"20" minimum:"1" maximum:"100" doc:"Items per page"`
}

// SearchCompanyLookupsResponse represents the search results
type SearchCompanyLookupsResponse struct {
	Body models.CompanyLookupListResponse `json:"result" doc:"Search results"`
}

// ========================================
// Handler Methods
// ========================================

// LookupCompany looks up a company by tax code via the external API
func (h *CompanyHandler) LookupCompany(ctx context.Context, req *LookupCompanyRequest) (*LookupCompanyResponse, error) {
	lookup, err := h.service.LookupCompany(ctx, req.TaxCode)
	if err != nil {
		if errors.Is(err, services.ErrCompanyNotFound) {
			return nil, huma.Error404NotFound("Company not found for the given tax code", err)
		}
		if errors.Is(err, services.ErrInvalidTaxCode) {
			return nil, huma.Error400BadRequest("Invalid tax code", err)
		}
		if errors.Is(err, services.ErrCircuitBreakerOpen) {
			return nil, huma.Error503ServiceUnavailable("Company lookup service temporarily unavailable", err)
		}
		return nil, huma.Error500InternalServerError("Failed to look up company", err)
	}

	return &LookupCompanyResponse{Body: *lookup}, nil
}

// GetCompanyLookup retrieves a previously stored company lookup by ID
func (h *CompanyHandler) GetCompanyLookup(ctx context.Context, req *GetCompanyLookupRequest) (*GetCompanyLookupResponse, error) {
	lookup, err := h.service.GetLookup(ctx, req.ID)
	if err != nil {
		if errors.Is(err, services.ErrLookupNotFound) {
			return nil, huma.Error404NotFound("Company lookup not found", err)
		}
		return nil, huma.Error500InternalServerError("Failed to get company lookup", err)
	}

	return &GetCompanyLookupResponse{Body: *lookup}, nil
}

// ListCompanyLookups lists previously looked-up companies with pagination
func (h *CompanyHandler) ListCompanyLookups(ctx context.Context, req *ListCompanyLookupsRequest) (*ListCompanyLookupsResponse, error) {
	lookups, total, err := h.service.ListLookups(ctx, req.Page, req.PageSize)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to list company lookups", err)
	}

	if lookups == nil {
		lookups = []models.CompanyLookup{}
	}

	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	return &ListCompanyLookupsResponse{
		Body: models.CompanyLookupListResponse{
			Lookups:    lookups,
			Total:      total,
			Page:       req.Page,
			PageSize:   req.PageSize,
			TotalPages: totalPages,
		},
	}, nil
}

// SearchCompanyLookups searches stored lookups by name, tax code, or VAT code
func (h *CompanyHandler) SearchCompanyLookups(ctx context.Context, req *SearchCompanyLookupsRequest) (*SearchCompanyLookupsResponse, error) {
	if req.Query == "" {
		return nil, huma.Error400BadRequest("Search query is required", nil)
	}

	lookups, total, err := h.service.SearchLookups(ctx, req.Query, req.Page, req.PageSize)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to search company lookups", err)
	}

	if lookups == nil {
		lookups = []models.CompanyLookup{}
	}

	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	return &SearchCompanyLookupsResponse{
		Body: models.CompanyLookupListResponse{
			Lookups:    lookups,
			Total:      total,
			Page:       req.Page,
			PageSize:   req.PageSize,
			TotalPages: totalPages,
		},
	}, nil
}
