// Package ownerrepo provides scoping helpers for collections keyed by a
// polymorphic owner — the post-onboarding generalization of tenantrepo.
// Where tenantrepo enforces "every query carries tenantId", ownerrepo
// enforces "every query carries (ownerKind, ownerUUID)" — the same
// fail-closed shape, applied to subscriptions, transactions, payment
// methods, billing customers, capability entitlements.
//
// Usage:
//
//	filter := ownerrepo.Scope(owner, bson.M{"status": "succeeded"})
//	cur, err := coll.Find(ctx, filter)
//
// The helper rejects a zero owner via panic in dev (so missing-scope bugs
// surface during development) and a bson filter that will match nothing
// in production (the safest fallback when callers somehow reach the
// repository without an owner).
package ownerrepo

import (
	"os"

	"github.com/orkestra/backend/internal/shared/iface"
	"go.mongodb.org/mongo-driver/bson"
)

// Field names persisted on every owner-scoped document. Centralized so
// renames flow through one constant rather than hopping through every
// repository.
const (
	OwnerKindField = "ownerKind"
	OwnerUUIDField = "ownerUUID"
)

// Scope returns the filter with ownerKind+ownerUUID added from the given
// owner. Panics in dev when the owner is zero so missing-scope bugs surface
// loudly during development; in production, returns a filter that cannot
// match any document (defense in depth — better an empty result set than
// a silent cross-owner read).
func Scope(owner iface.Owner, filter bson.M) bson.M {
	if owner.IsZero() {
		if isDev() {
			panic("ownerrepo.Scope: owner is zero — caller forgot to scope this query")
		}
		// Production fallback: a filter that cannot match any document.
		return bson.M{"_id": bson.M{"$exists": false}, "_": bson.M{"$exists": false}}
	}
	if filter == nil {
		filter = bson.M{}
	}
	filter[OwnerKindField] = string(owner.Kind)
	filter[OwnerUUIDField] = owner.UUID
	return filter
}

// MustScope is Scope for call sites statically known to have a non-zero
// owner. Panics on a zero owner unconditionally.
func MustScope(owner iface.Owner, filter bson.M) bson.M {
	if owner.IsZero() {
		panic("ownerrepo.MustScope: owner is zero")
	}
	return Scope(owner, filter)
}

// StampInsert returns the (ownerKind, ownerUUID) pair the caller should
// stamp on a new document before InsertOne. Helpful when the document
// shape is a struct rather than bson.M and the caller manages field
// assignment directly.
func StampInsert(owner iface.Owner) (kind iface.OwnerKind, uuid string) {
	if owner.IsZero() {
		if isDev() {
			panic("ownerrepo.StampInsert: owner is zero")
		}
	}
	return owner.Kind, owner.UUID
}

// StampInsertM mutates and returns doc with the polymorphic owner stamped.
// Mirrors tenantrepo.StampInsertM for bson-map inserts.
func StampInsertM(owner iface.Owner, doc bson.M) bson.M {
	if owner.IsZero() {
		if isDev() {
			panic("ownerrepo.StampInsertM: owner is zero")
		}
		return doc
	}
	if doc == nil {
		doc = bson.M{}
	}
	doc[OwnerKindField] = string(owner.Kind)
	doc[OwnerUUIDField] = owner.UUID
	return doc
}

func isDev() bool {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = os.Getenv("ENV")
	}
	switch env {
	case "development", "dev", "local", "":
		return true
	}
	return false
}
