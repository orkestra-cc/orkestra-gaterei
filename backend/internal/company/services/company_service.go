package services

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/orkestra/backend/internal/company/models"
	"github.com/orkestra/backend/internal/company/repository"
)

// Service errors
var (
	ErrLookupNotFound = errors.New("company lookup not found")
)

// CompanyService defines the interface for company lookup business logic
type CompanyService interface {
	LookupCompany(ctx context.Context, taxCode string) (*models.CompanyLookup, error)
	GetLookup(ctx context.Context, uuid string) (*models.CompanyLookup, error)
	ListLookups(ctx context.Context, page, pageSize int) ([]models.CompanyLookup, int64, error)
	SearchLookups(ctx context.Context, query string, page, pageSize int) ([]models.CompanyLookup, int64, error)
}

type companyService struct {
	repo      repository.CompanyRepository
	apiClient CompanyAPIClient
	logger    *slog.Logger
}

// NewCompanyService creates a new CompanyService
func NewCompanyService(repo repository.CompanyRepository, apiClient CompanyAPIClient, logger *slog.Logger) CompanyService {
	return &companyService{
		repo:      repo,
		apiClient: apiClient,
		logger:    logger,
	}
}

// LookupCompany looks up a company by tax code from the external API,
// persists the result in MongoDB, and returns it
func (s *companyService) LookupCompany(ctx context.Context, taxCode string) (*models.CompanyLookup, error) {
	if taxCode == "" {
		return nil, ErrInvalidTaxCode
	}

	// Call external API (client handles Redis caching internally)
	response, err := s.apiClient.LookupByTaxCode(ctx, taxCode)
	if err != nil {
		if errors.Is(err, ErrCompanyNotFound) {
			return nil, ErrCompanyNotFound
		}
		s.logger.Error("company API lookup failed",
			"taxCode", taxCode,
			"error", err,
		)
		return nil, err
	}

	if response.Data == nil {
		return nil, ErrCompanyNotFound
	}

	// Map API response to domain model
	lookup := mapResponseToLookup(response.Data)

	// Check if we already have this lookup
	existing, err := s.repo.GetByTaxCode(ctx, lookup.TaxCode)
	if err == nil {
		lookup.UUID = existing.UUID
	} else {
		lookup.UUID = uuid.New().String()
	}

	// Upsert into MongoDB
	if err := s.repo.Upsert(ctx, lookup); err != nil {
		s.logger.Error("failed to upsert company lookup",
			"taxCode", taxCode,
			"error", err,
		)
		return nil, err
	}

	s.logger.Info("company lookup completed",
		"taxCode", taxCode,
		"companyName", lookup.CompanyName,
		"activityStatus", lookup.ActivityStatus,
	)

	return lookup, nil
}

// GetLookup retrieves a previously stored company lookup by UUID
func (s *companyService) GetLookup(ctx context.Context, id string) (*models.CompanyLookup, error) {
	lookup, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrLookupNotFound) {
			return nil, ErrLookupNotFound
		}
		return nil, err
	}
	return lookup, nil
}

// ListLookups returns a paginated list of stored company lookups
func (s *companyService) ListLookups(ctx context.Context, page, pageSize int) ([]models.CompanyLookup, int64, error) {
	return s.repo.List(ctx, page, pageSize)
}

// SearchLookups searches stored company lookups by name, tax code, or VAT code
func (s *companyService) SearchLookups(ctx context.Context, query string, page, pageSize int) ([]models.CompanyLookup, int64, error) {
	return s.repo.Search(ctx, query, page, pageSize)
}

// mapResponseToLookup converts an API response to a domain model
func mapResponseToLookup(data *models.OpenAPICompanyData) *models.CompanyLookup {
	lookup := &models.CompanyLookup{
		TaxCode:          data.TaxCode,
		CompanyName:      data.CompanyName,
		VATCode:          data.VATCode,
		ActivityStatus:   data.ActivityStatus,
		SDICode:          data.SDICode,
		RegistrationDate: data.RegistrationDate,
		SourceID:         data.ID,
	}

	if data.Address.RegisteredOffice != nil {
		office := data.Address.RegisteredOffice
		lookup.Address = models.Address{
			Street:       office.StreetName,
			StreetNumber: office.StreetNumber,
			Town:         office.Town,
			Province:     office.Province,
			ZipCode:      office.ZipCode,
			Region:       office.Region,
		}
	}

	return lookup
}
