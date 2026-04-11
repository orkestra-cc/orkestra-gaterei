package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Plan names used by the tenant module. Not an exhaustive enum — custom
// plans can be created at runtime — just the defaults seeded on first boot.
const (
	PlanFree       = "free"
	PlanPro        = "pro"
	PlanEnterprise = "enterprise"
)

// Feature keys are the entitlements a plan can grant. Modules advertise the
// feature they require on their tenant-scoped routes via RequireEntitlement.
// A tenant holds an explicit list of enabled features, and the wildcard "*"
// grants all features (used by the enterprise plan by default).
const FeatureWildcard = "*"

// Org is a tenant. One user can belong to many orgs with different roles in
// each via the Membership model.
type Org struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	UUID          string             `bson:"uuid" json:"id" validate:"required"`
	Name          string             `bson:"name" json:"name" validate:"required,min=1,max=120"`
	Slug          string             `bson:"slug" json:"slug" validate:"required,min=1,max=80"`
	OwnerUserUUID string             `bson:"ownerUserUUID" json:"ownerUserUUID"`
	Plan          string             `bson:"plan" json:"plan"`
	Features      []string           `bson:"features" json:"features"`
	Settings      map[string]string  `bson:"settings,omitempty" json:"settings,omitempty"`
	CreatedAt     time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt" json:"updatedAt"`
	DeletedAt     *time.Time         `bson:"deletedAt,omitempty" json:"-"`
}

// HasFeature reports whether the org's plan includes the given feature.
func (o *Org) HasFeature(feature string) bool {
	for _, f := range o.Features {
		if f == FeatureWildcard || f == feature {
			return true
		}
	}
	return false
}

// Membership links a user to an org with a set of role names (defined in
// the authz module). A user with no membership cannot access that org.
type Membership struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	UUID      string             `bson:"uuid" json:"id"`
	UserUUID  string             `bson:"userUUID" json:"userUUID" validate:"required"`
	OrgUUID   string             `bson:"orgId" json:"orgId" validate:"required"`
	Roles     []string           `bson:"roles" json:"roles"`
	IsOwner   bool               `bson:"isOwner" json:"isOwner"`
	InvitedBy string             `bson:"invitedBy,omitempty" json:"invitedBy,omitempty"`
	JoinedAt  time.Time          `bson:"joinedAt" json:"joinedAt"`
	ExpiresAt *time.Time         `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
}

// Invite is a pending invitation for a user to join an org. Tokens are
// single-use and expire automatically via a TTL index on expiresAt.
//
// The raw Token is serialized with `omitempty` so it appears on the create
// response (where the service populates it) but is scrubbed from list
// responses by the service zeroing the field before return.
type Invite struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	UUID       string             `bson:"uuid" json:"id"`
	OrgUUID    string             `bson:"orgId" json:"orgId"`
	Email      string             `bson:"email" json:"email"`
	Roles      []string           `bson:"roles" json:"roles"`
	Token      string             `bson:"token" json:"token,omitempty"`
	InvitedBy  string             `bson:"invitedBy" json:"invitedBy"`
	CreatedAt  time.Time          `bson:"createdAt" json:"createdAt"`
	ExpiresAt  time.Time          `bson:"expiresAt" json:"expiresAt"`
	AcceptedAt *time.Time         `bson:"acceptedAt,omitempty" json:"acceptedAt,omitempty"`
}

// --- API DTOs ---

type CreateOrgInput struct {
	Name string `json:"name" validate:"required,min=1,max=120"`
	Slug string `json:"slug" validate:"required,min=1,max=80"`
	Plan string `json:"plan,omitempty"`
}

type UpdateOrgInput struct {
	Name     *string           `json:"name,omitempty"`
	Slug     *string           `json:"slug,omitempty"`
	Settings map[string]string `json:"settings,omitempty"`
}

type UpdatePlanInput struct {
	Plan     string   `json:"plan" validate:"required"`
	Features []string `json:"features"`
}

type InviteInput struct {
	Email string   `json:"email" validate:"required,email"`
	Roles []string `json:"roles" validate:"required,min=1"`
}

type AcceptInviteInput struct {
	Token string `json:"token" validate:"required"`
}

type OrgListResponse struct {
	Orgs []Org `json:"orgs"`
}

type MembershipListResponse struct {
	Memberships []Membership `json:"memberships"`
}
