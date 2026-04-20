package repository

import (
	"context"
	"errors"
	"time"

	"github.com/orkestra/backend/internal/addons/payments/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrTransactionNotFound = errors.New("payments: transaction not found")

type TransactionFilters struct {
	SubscriptionUUID string
	InvoiceUUID      string
	ClientUUID       string
	TenantUUID       string
	Status           models.TransactionStatus
	Since            *time.Time
	Until            *time.Time
}

type TransactionRepository interface {
	Create(ctx context.Context, t *models.Transaction) error
	GetByUUID(ctx context.Context, uuid string) (*models.Transaction, error)
	GetByProviderTxID(ctx context.Context, provider models.ProviderName, providerTxID string) (*models.Transaction, error)
	List(ctx context.Context, f TransactionFilters) ([]models.Transaction, error)
	// FindByTenantUUID lists transactions bound to the tenant via the
	// Phase 1 forward-looking TenantUUID. Backs the Phase 2 aggregator
	// GET /v1/admin/tenants/{id}/payments.
	FindByTenantUUID(ctx context.Context, tenantUUID string) ([]models.Transaction, error)
	// FindWithoutTenantUUID returns transactions still missing TenantUUID —
	// the cold-tail set the Phase 1 backfill migration walks. `limit` caps the
	// page size; 0 = caller-defined default.
	FindWithoutTenantUUID(ctx context.Context, limit int64) ([]models.Transaction, error)
	// SetTenantUUID back-stamps the forward-looking tenant binding on an
	// existing transaction row. Used by the backfill migration.
	SetTenantUUID(ctx context.Context, transactionUUID, tenantUUID string) error
	Update(ctx context.Context, t *models.Transaction) error
}

type transactionRepository struct {
	coll *mongo.Collection
}

func NewTransactionRepository(db *mongo.Database) TransactionRepository {
	return &transactionRepository{coll: db.Collection(models.TransactionsCollection)}
}

func (r *transactionRepository) Create(ctx context.Context, t *models.Transaction) error {
	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, t)
	return err
}

func (r *transactionRepository) GetByUUID(ctx context.Context, uuid string) (*models.Transaction, error) {
	var t models.Transaction
	err := r.coll.FindOne(ctx, bson.M{"uuid": uuid}).Decode(&t)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrTransactionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *transactionRepository) GetByProviderTxID(ctx context.Context, provider models.ProviderName, providerTxID string) (*models.Transaction, error) {
	var t models.Transaction
	err := r.coll.FindOne(ctx, bson.M{"provider": provider, "providerTxID": providerTxID}).Decode(&t)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrTransactionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *transactionRepository) List(ctx context.Context, f TransactionFilters) ([]models.Transaction, error) {
	filter := bson.M{}
	if f.SubscriptionUUID != "" {
		filter["subscriptionUUID"] = f.SubscriptionUUID
	}
	if f.InvoiceUUID != "" {
		filter["invoiceUUID"] = f.InvoiceUUID
	}
	if f.ClientUUID != "" {
		filter["clientUUID"] = f.ClientUUID
	}
	if f.TenantUUID != "" {
		filter["tenantUUID"] = f.TenantUUID
	}
	if f.Status != "" {
		filter["status"] = f.Status
	}
	if f.Since != nil || f.Until != nil {
		rng := bson.M{}
		if f.Since != nil {
			rng["$gte"] = *f.Since
		}
		if f.Until != nil {
			rng["$lte"] = *f.Until
		}
		filter["createdAt"] = rng
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := make([]models.Transaction, 0)
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *transactionRepository) FindByTenantUUID(ctx context.Context, tenantUUID string) ([]models.Transaction, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := r.coll.Find(ctx, bson.M{"tenantUUID": tenantUUID}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := make([]models.Transaction, 0)
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *transactionRepository) FindWithoutTenantUUID(ctx context.Context, limit int64) ([]models.Transaction, error) {
	filter := bson.M{"$or": []bson.M{
		{"tenantUUID": bson.M{"$exists": false}},
		{"tenantUUID": ""},
	}}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}})
	if limit > 0 {
		opts = opts.SetLimit(limit)
	}
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := make([]models.Transaction, 0)
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *transactionRepository) SetTenantUUID(ctx context.Context, transactionUUID, tenantUUID string) error {
	res, err := r.coll.UpdateOne(ctx, bson.M{"uuid": transactionUUID}, bson.M{
		"$set": bson.M{"tenantUUID": tenantUUID, "updatedAt": time.Now().UTC()},
	})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrTransactionNotFound
	}
	return nil
}

func (r *transactionRepository) Update(ctx context.Context, t *models.Transaction) error {
	t.UpdatedAt = time.Now().UTC()
	res, err := r.coll.ReplaceOne(ctx, bson.M{"uuid": t.UUID}, t)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrTransactionNotFound
	}
	return nil
}
