package repository

import (
	"context"
	"errors"
	"time"

	"github.com/orkestra/backend/internal/core/notification/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// UnsubscribeRepository stores hashed unsubscribe tokens.
// Tokens are single-use: UsedAt is set on consume and the document
// TTLs after 30 days via a MongoDB index on ExpiresAt.
type UnsubscribeRepository interface {
	Create(ctx context.Context, doc *models.UnsubscribeTokenDoc) error
	GetByHash(ctx context.Context, hash string) (*models.UnsubscribeTokenDoc, error)
	MarkUsed(ctx context.Context, hash string) error
}

type unsubscribeRepository struct {
	coll *mongo.Collection
}

func NewUnsubscribeRepository(db *mongo.Database) UnsubscribeRepository {
	return &unsubscribeRepository{
		coll: db.Collection(models.NotificationUnsubscribeTokensCollect),
	}
}

func (r *unsubscribeRepository) Create(ctx context.Context, doc *models.UnsubscribeTokenDoc) error {
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}
	_, err := r.coll.InsertOne(ctx, doc)
	return err
}

func (r *unsubscribeRepository) GetByHash(ctx context.Context, hash string) (*models.UnsubscribeTokenDoc, error) {
	var doc models.UnsubscribeTokenDoc
	err := r.coll.FindOne(ctx, bson.M{"tokenHash": hash}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *unsubscribeRepository) MarkUsed(ctx context.Context, hash string) error {
	now := time.Now()
	_, err := r.coll.UpdateOne(ctx,
		bson.M{"tokenHash": hash},
		bson.M{"$set": bson.M{"usedAt": now}},
	)
	return err
}
