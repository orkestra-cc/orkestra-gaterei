package repository

import (
	"context"
	"errors"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/orkestra/backend/internal/company/models"
)

// Repository errors
var (
	ErrLookupNotFound = errors.New("company lookup not found")
)

// CompanyRepository defines the interface for company lookup data access
type CompanyRepository interface {
	Upsert(ctx context.Context, lookup *models.CompanyLookup) error
	GetByTaxCode(ctx context.Context, taxCode string) (*models.CompanyLookup, error)
	GetByID(ctx context.Context, uuid string) (*models.CompanyLookup, error)
	List(ctx context.Context, page, pageSize int) ([]models.CompanyLookup, int64, error)
	Search(ctx context.Context, query string, page, pageSize int) ([]models.CompanyLookup, int64, error)
}

type companyRepository struct {
	collection *mongo.Collection
}

// NewCompanyRepository creates a new CompanyRepository
func NewCompanyRepository(db *mongo.Database) CompanyRepository {
	repo := &companyRepository{
		collection: db.Collection("company_lookups"),
	}
	repo.createIndexes(context.Background())
	return repo
}

func (r *companyRepository) createIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "uuid", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "taxCode", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "vatCode", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "companyName", Value: "text"}},
		},
		{
			Keys: bson.D{{Key: "createdAt", Value: -1}},
		},
	}

	_, _ = r.collection.Indexes().CreateMany(ctx, indexes)
}

// Upsert creates or updates a company lookup by tax code
func (r *companyRepository) Upsert(ctx context.Context, lookup *models.CompanyLookup) error {
	now := time.Now()
	lookup.UpdatedAt = now

	filter := bson.M{"taxCode": lookup.TaxCode}

	// Check if document exists
	var existing models.CompanyLookup
	err := r.collection.FindOne(ctx, filter).Decode(&existing)
	if err == nil {
		// Update existing
		lookup.ID = existing.ID
		lookup.UUID = existing.UUID
		lookup.CreatedAt = existing.CreatedAt

		_, err := r.collection.ReplaceOne(ctx, filter, lookup)
		return err
	}

	if !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}

	// Insert new
	lookup.CreatedAt = now
	result, err := r.collection.InsertOne(ctx, lookup)
	if err != nil {
		return err
	}

	lookup.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetByTaxCode retrieves a company lookup by tax code
func (r *companyRepository) GetByTaxCode(ctx context.Context, taxCode string) (*models.CompanyLookup, error) {
	var lookup models.CompanyLookup
	err := r.collection.FindOne(ctx, bson.M{"taxCode": taxCode}).Decode(&lookup)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrLookupNotFound
		}
		return nil, err
	}
	return &lookup, nil
}

// GetByID retrieves a company lookup by UUID
func (r *companyRepository) GetByID(ctx context.Context, uuid string) (*models.CompanyLookup, error) {
	var lookup models.CompanyLookup
	err := r.collection.FindOne(ctx, bson.M{"uuid": uuid}).Decode(&lookup)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrLookupNotFound
		}
		return nil, err
	}
	return &lookup, nil
}

// List returns a paginated list of company lookups
func (r *companyRepository) List(ctx context.Context, page, pageSize int) ([]models.CompanyLookup, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	total, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * pageSize)
	limit := int64(pageSize)

	opts := options.Find().
		SetSort(bson.D{{Key: "updatedAt", Value: -1}}).
		SetSkip(skip).
		SetLimit(limit)

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var lookups []models.CompanyLookup
	if err := cursor.All(ctx, &lookups); err != nil {
		return nil, 0, err
	}

	return lookups, total, nil
}

// Search searches company lookups by name, tax code, or VAT code
func (r *companyRepository) Search(ctx context.Context, query string, page, pageSize int) ([]models.CompanyLookup, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Escape special regex characters to prevent ReDoS attacks
	escapedQuery := regexp.QuoteMeta(query)
	filter := bson.M{
		"$or": []bson.M{
			{"companyName": bson.M{"$regex": escapedQuery, "$options": "i"}},
			{"taxCode": bson.M{"$regex": escapedQuery, "$options": "i"}},
			{"vatCode": bson.M{"$regex": escapedQuery, "$options": "i"}},
		},
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * pageSize)
	limit := int64(pageSize)

	opts := options.Find().
		SetSort(bson.D{{Key: "companyName", Value: 1}}).
		SetSkip(skip).
		SetLimit(limit)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var lookups []models.CompanyLookup
	if err := cursor.All(ctx, &lookups); err != nil {
		return nil, 0, err
	}

	return lookups, total, nil
}
