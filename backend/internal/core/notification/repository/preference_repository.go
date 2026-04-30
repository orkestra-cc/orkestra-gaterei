package repository

import (
	"context"
	"errors"
	"time"

	"github.com/orkestra/backend/internal/core/notification/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PreferenceRepository manages user notification preferences and
// the address-level suppression list.
type PreferenceRepository interface {
	GetPreference(ctx context.Context, userUUID, category, channel string) (*models.PreferenceDoc, error)
	ListByUser(ctx context.Context, userUUID string) ([]*models.PreferenceDoc, error)
	UpsertPreference(ctx context.Context, doc *models.PreferenceDoc) error

	IsSuppressed(ctx context.Context, address string) (bool, error)
	AddSuppression(ctx context.Context, doc *models.SuppressionDoc) error
	RemoveSuppression(ctx context.Context, address string) error
}

type preferenceRepository struct {
	prefColl       *mongo.Collection
	suppressedColl *mongo.Collection
}

func NewPreferenceRepository(db *mongo.Database) PreferenceRepository {
	return &preferenceRepository{
		prefColl:       db.Collection(models.NotificationPreferencesCollection),
		suppressedColl: db.Collection(models.NotificationSuppressionsCollection),
	}
}

func (r *preferenceRepository) GetPreference(ctx context.Context, userUUID, category, channel string) (*models.PreferenceDoc, error) {
	var doc models.PreferenceDoc
	err := r.prefColl.FindOne(ctx, bson.M{
		"userUuid": userUUID,
		"category": category,
		"channel":  channel,
	}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *preferenceRepository) ListByUser(ctx context.Context, userUUID string) ([]*models.PreferenceDoc, error) {
	cursor, err := r.prefColl.Find(ctx, bson.M{"userUuid": userUUID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var out []*models.PreferenceDoc
	for cursor.Next(ctx) {
		var d models.PreferenceDoc
		if err := cursor.Decode(&d); err != nil {
			return nil, err
		}
		out = append(out, &d)
	}
	return out, cursor.Err()
}

func (r *preferenceRepository) UpsertPreference(ctx context.Context, doc *models.PreferenceDoc) error {
	doc.UpdatedAt = time.Now()
	_, err := r.prefColl.UpdateOne(ctx,
		bson.M{
			"userUuid": doc.UserUUID,
			"category": doc.Category,
			"channel":  doc.Channel,
		},
		bson.M{
			"$set": bson.M{
				"optedIn":   doc.OptedIn,
				"updatedAt": doc.UpdatedAt,
			},
			"$setOnInsert": bson.M{
				"userUuid": doc.UserUUID,
				"category": doc.Category,
				"channel":  doc.Channel,
			},
		},
		options.Update().SetUpsert(true),
	)
	return err
}

func (r *preferenceRepository) IsSuppressed(ctx context.Context, address string) (bool, error) {
	count, err := r.suppressedColl.CountDocuments(ctx, bson.M{"address": address})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *preferenceRepository) AddSuppression(ctx context.Context, doc *models.SuppressionDoc) error {
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}
	_, err := r.suppressedColl.UpdateOne(ctx,
		bson.M{"address": doc.Address},
		bson.M{
			"$set": bson.M{
				"reason":    doc.Reason,
				"createdAt": doc.CreatedAt,
			},
			"$setOnInsert": bson.M{
				"address": doc.Address,
			},
		},
		options.Update().SetUpsert(true),
	)
	return err
}

func (r *preferenceRepository) RemoveSuppression(ctx context.Context, address string) error {
	_, err := r.suppressedColl.DeleteOne(ctx, bson.M{"address": address})
	return err
}
