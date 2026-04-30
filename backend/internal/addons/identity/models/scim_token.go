package models

import "time"

// ScimTokensCollection holds the hashed bearer tokens IdPs use to
// authenticate against `/scim/v2/*`. One active token per tenant — the
// rotate endpoint revokes the previous row and inserts a new one in the
// same transaction semantics (upsert with CAS on revokedAt).
const ScimTokensCollection = "identity_scim_tokens"

// ScimToken is the persisted row.
//
// TokenHash is a SHA-256 hex digest of the raw token. The raw token is
// returned to the admin exactly once on rotation and never persisted.
// A collision-resistant 32-byte random seed is the source material so
// brute-forcing the hash offline is computationally infeasible.
type ScimToken struct {
	UUID      string    `bson:"uuid" json:"uuid"`
	TenantID  string    `bson:"tenantId" json:"tenantId"`
	TokenHash string    `bson:"tokenHash" json:"-"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	// RevokedAt is nil while the token is live; rotation stamps it before
	// inserting the replacement so old hashes stop authenticating.
	RevokedAt *time.Time `bson:"revokedAt,omitempty" json:"revokedAt,omitempty"`
}
