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
	// AdvanceLastUsedStep atomically updates LastUsedStep only if the
	// supplied step is strictly greater than the stored value. Returns
	// (true, nil) when the write landed, (false, nil) when another
	// concurrent verify beat this one to the same step. The concurrency
	// guarantee is what prevents replay within the 30-second TOTP window.
	AdvanceLastUsedStep(ctx context.Context, uuid string, step int64, when time.Time) (bool, error)
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

// AdvanceLastUsedStep updates the step only if strictly newer than the
// stored value. The filter's `$lt` predicate turns the CAS into a single
// atomic op; a concurrent Verify that races a second call with the same
// step sees ModifiedCount==0 and knows its code was already consumed.
func (r *mfaFactorRepository) AdvanceLastUsedStep(ctx context.Context, uuid string, step int64, when time.Time) (bool, error) {
	res, err := r.coll.UpdateOne(ctx,
		bson.M{
			"uuid": uuid,
			"$or": []bson.M{
				{"lastUsedStep": bson.M{"$lt": step}},
				{"lastUsedStep": bson.M{"$exists": false}},
			},
		},
		bson.M{"$set": bson.M{"lastUsedStep": step, "lastUsedAt": when}},
	)
	if err != nil {
		return false, err
	}
	return res.ModifiedCount > 0, nil
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
