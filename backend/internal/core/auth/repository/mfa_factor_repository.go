package repository

import (
	"context"
	"errors"
	"time"

	"github.com/orkestra/backend/internal/core/auth/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ErrMFAFactorNotFound signals that no matching factor exists for the user.
var ErrMFAFactorNotFound = errors.New("mfa factor not found")

// MFAFactorRepository persists the MFA factor rows owned by the auth module.
// All methods are context-aware and return ErrMFAFactorNotFound when the
// caller should treat the absence as "not enrolled" rather than an error.
type MFAFactorRepository interface {
	Insert(ctx context.Context, doc *models.MFAFactorDoc) error
	FindByUserAndType(ctx context.Context, userUUID string, factorType models.MFAFactorType) (*models.MFAFactorDoc, error)
	UpdateLastUsed(ctx context.Context, uuid string, when time.Time) error
	ConsumeBackupCode(ctx context.Context, userUUID, hashedCode string) (bool, error)
	Delete(ctx context.Context, uuid string) error
}

type mfaFactorRepository struct {
	coll *mongo.Collection
}

// NewMFAFactorRepository wires a repository against the canonical collection.
func NewMFAFactorRepository(db *mongo.Database) MFAFactorRepository {
	return &mfaFactorRepository{
		coll: db.Collection(models.MFAFactorsCollection),
	}
}

func (r *mfaFactorRepository) Insert(ctx context.Context, doc *models.MFAFactorDoc) error {
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}
	_, err := r.coll.InsertOne(ctx, doc)
	return err
}

func (r *mfaFactorRepository) FindByUserAndType(ctx context.Context, userUUID string, factorType models.MFAFactorType) (*models.MFAFactorDoc, error) {
	var doc models.MFAFactorDoc
	err := r.coll.FindOne(ctx, bson.M{"userUuid": userUUID, "type": factorType}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrMFAFactorNotFound
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *mfaFactorRepository) UpdateLastUsed(ctx context.Context, uuid string, when time.Time) error {
	_, err := r.coll.UpdateOne(ctx,
		bson.M{"uuid": uuid},
		bson.M{"$set": bson.M{"lastUsedAt": when}},
	)
	return err
}

// ConsumeBackupCode removes a single backup-code hash from the user's factor.
// Returns (true, nil) when a code was removed, (false, nil) when the hash was
// not present (already used or never issued). A non-nil error is only returned
// for driver/io failures.
func (r *mfaFactorRepository) ConsumeBackupCode(ctx context.Context, userUUID, hashedCode string) (bool, error) {
	res, err := r.coll.UpdateOne(ctx,
		bson.M{
			"userUuid":          userUUID,
			"backupCodesHashed": hashedCode,
		},
		bson.M{"$pull": bson.M{"backupCodesHashed": hashedCode}},
	)
	if err != nil {
		return false, err
	}
	return res.ModifiedCount > 0, nil
}

func (r *mfaFactorRepository) Delete(ctx context.Context, uuid string) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"uuid": uuid})
	return err
}
