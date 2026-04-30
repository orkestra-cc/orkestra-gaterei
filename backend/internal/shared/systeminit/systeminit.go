// Package systeminit owns the platform's one-time bootstrap sentinel.
//
// The "first user becomes super_admin" rule has two callers — the setup
// wizard (`POST /v1/setup/admin`) and ordinary signup (`POST /v1/auth/
// register`) — and both historically used a `GetUserCount() == 0` check
// immediately followed by a user create. That's a TOCTOU race: two concurrent
// signups on a fresh install can both see count==0 and both be granted
// super_admin. The patch is a single document in `system_init` with a
// unique-indexed `key`; the first caller to upsert it wins.
//
// The package is deliberately outside any Module so main.go can construct it
// next to the ModuleConfigRepository and inject it into both the setup
// service and the password auth service. It carries no business logic — only
// the atomic CAS contract.
package systeminit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// collection is the single MongoDB collection this package owns. Plural
	// or prefixed names would signal "more to come"; this really is one doc
	// for one platform invariant, so a plain name is correct.
	collection = "system_init"

	// keyFirstAdmin is the fixed document key for the first-admin seat. If
	// we ever need another platform-wide seat (e.g. "first tenant"), use a
	// new key rather than reusing this one.
	keyFirstAdmin = "first_admin"
)

// Repo owns the system_init collection.
type Repo struct {
	coll *mongo.Collection
}

// NewRepo returns a Repo and ensures the `key` unique index exists. The
// index-creation call is idempotent; safe to call on every boot. If the
// index creation fails the caller gets an error rather than a silently
// broken claim path, because without the unique index the race we're
// defending against returns.
func NewRepo(ctx context.Context, db *mongo.Database) (*Repo, error) {
	coll := db.Collection(collection)
	_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "key", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("key_unique"),
	})
	if err != nil {
		return nil, fmt.Errorf("systeminit: ensure index: %w", err)
	}
	return &Repo{coll: coll}, nil
}

// ClaimFirstAdmin atomically tries to register userUUID as the platform's
// first super_admin. It returns claimed=true only if this call performed the
// insert; concurrent callers see claimed=false and should fall back to a
// non-super role.
//
// Rollback: if the subsequent user-creation step fails, the caller MUST call
// Release(userUUID). Leaving the sentinel behind while no user exists bricks
// future signups into "operator" forever — the setup wizard would silently
// stop minting super_admins even on a DB that otherwise looks fresh.
func (r *Repo) ClaimFirstAdmin(ctx context.Context, userUUID string) (claimed bool, err error) {
	if userUUID == "" {
		return false, errors.New("systeminit: userUUID is required")
	}
	// Upsert with $setOnInsert is the atomic CAS: either this call inserts
	// the document (UpsertedCount == 1) or it doesn't (the sentinel already
	// exists, somebody else won).
	res, err := r.coll.UpdateOne(ctx,
		bson.M{"key": keyFirstAdmin},
		bson.M{"$setOnInsert": bson.M{
			"key":       keyFirstAdmin,
			"userUUID":  userUUID,
			"claimedAt": time.Now().UTC(),
		}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return false, fmt.Errorf("systeminit: claim first admin: %w", err)
	}
	return res.UpsertedCount > 0, nil
}

// Release removes the first-admin sentinel if — and only if — it currently
// points at userUUID. Used for rollback when the user-creation step fails
// after a successful claim. The guarded delete ensures a slow rollback
// from one caller cannot clobber a fresh sentinel already claimed by another.
func (r *Repo) Release(ctx context.Context, userUUID string) error {
	if userUUID == "" {
		return errors.New("systeminit: userUUID is required")
	}
	_, err := r.coll.DeleteOne(ctx, bson.M{
		"key":      keyFirstAdmin,
		"userUUID": userUUID,
	})
	if err != nil {
		return fmt.Errorf("systeminit: release first admin: %w", err)
	}
	return nil
}

// IsFirstAdminClaimed reports whether the sentinel exists. Used by the setup
// wizard's Status endpoint to decide whether to show the first-install flow.
// The count-based "setup completed" check in the setup service is kept as
// well because it's a cheaper sanity read and stays meaningful even if the
// sentinel hasn't been populated on legacy deployments.
func (r *Repo) IsFirstAdminClaimed(ctx context.Context) (bool, error) {
	n, err := r.coll.CountDocuments(ctx, bson.M{"key": keyFirstAdmin}, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("systeminit: check claim: %w", err)
	}
	return n > 0, nil
}
