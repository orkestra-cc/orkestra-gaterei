package repository

import (
	"context"
	"errors"
	"time"

	"github.com/orkestra/backend/internal/core/tenant/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	CollOrgs        = "tenant_orgs"
	CollMemberships = "tenant_memberships"
	CollInvites     = "tenant_org_invites"
)

var ErrNotFound = errors.New("tenant: not found")

type Repository struct {
	db *mongo.Database
}

func New(db *mongo.Database) *Repository {
	return &Repository{db: db}
}

// --- Orgs ---

func (r *Repository) CreateOrg(ctx context.Context, o *models.Org) error {
	o.CreatedAt = time.Now()
	o.UpdatedAt = o.CreatedAt
	_, err := r.db.Collection(CollOrgs).InsertOne(ctx, o)
	return err
}

func (r *Repository) GetOrgByUUID(ctx context.Context, uuid string) (*models.Org, error) {
	var o models.Org
	err := r.db.Collection(CollOrgs).FindOne(ctx, bson.M{"uuid": uuid, "deletedAt": nil}).Decode(&o)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	return &o, err
}

func (r *Repository) GetOrgBySlug(ctx context.Context, slug string) (*models.Org, error) {
	var o models.Org
	err := r.db.Collection(CollOrgs).FindOne(ctx, bson.M{"slug": slug, "deletedAt": nil}).Decode(&o)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	return &o, err
}

func (r *Repository) UpdateOrg(ctx context.Context, uuid string, update bson.M) error {
	update["updatedAt"] = time.Now()
	res, err := r.db.Collection(CollOrgs).UpdateOne(ctx, bson.M{"uuid": uuid}, bson.M{"$set": update})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) SoftDeleteOrg(ctx context.Context, uuid string) error {
	now := time.Now()
	res, err := r.db.Collection(CollOrgs).UpdateOne(ctx, bson.M{"uuid": uuid}, bson.M{"$set": bson.M{"deletedAt": now, "updatedAt": now}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Memberships ---

func (r *Repository) CreateMembership(ctx context.Context, m *models.Membership) error {
	m.JoinedAt = time.Now()
	_, err := r.db.Collection(CollMemberships).InsertOne(ctx, m)
	return err
}

func (r *Repository) DeleteMembership(ctx context.Context, userUUID, orgUUID string) error {
	_, err := r.db.Collection(CollMemberships).DeleteOne(ctx, bson.M{"userUUID": userUUID, "orgId": orgUUID})
	return err
}

func (r *Repository) ListMembershipsByUser(ctx context.Context, userUUID string) ([]models.Membership, error) {
	cur, err := r.db.Collection(CollMemberships).Find(ctx, bson.M{
		"userUUID": userUUID,
		"$or":      []bson.M{{"expiresAt": nil}, {"expiresAt": bson.M{"$gt": time.Now()}}},
	})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []models.Membership
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) ListMembershipsByOrg(ctx context.Context, orgUUID string) ([]models.Membership, error) {
	cur, err := r.db.Collection(CollMemberships).Find(ctx, bson.M{"orgId": orgUUID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []models.Membership
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) GetMembership(ctx context.Context, userUUID, orgUUID string) (*models.Membership, error) {
	var m models.Membership
	err := r.db.Collection(CollMemberships).FindOne(ctx, bson.M{"userUUID": userUUID, "orgId": orgUUID}).Decode(&m)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	return &m, err
}

func (r *Repository) UpdateMembershipRoles(ctx context.Context, userUUID, orgUUID string, roles []string) error {
	_, err := r.db.Collection(CollMemberships).UpdateOne(ctx,
		bson.M{"userUUID": userUUID, "orgId": orgUUID},
		bson.M{"$set": bson.M{"roles": roles}})
	return err
}

// --- Invites ---

func (r *Repository) CreateInvite(ctx context.Context, inv *models.Invite) error {
	inv.CreatedAt = time.Now()
	_, err := r.db.Collection(CollInvites).InsertOne(ctx, inv)
	return err
}

func (r *Repository) GetInviteByToken(ctx context.Context, token string) (*models.Invite, error) {
	var inv models.Invite
	err := r.db.Collection(CollInvites).FindOne(ctx, bson.M{"token": token}).Decode(&inv)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	return &inv, err
}

func (r *Repository) MarkInviteAccepted(ctx context.Context, uuid string) error {
	now := time.Now()
	_, err := r.db.Collection(CollInvites).UpdateOne(ctx,
		bson.M{"uuid": uuid},
		bson.M{"$set": bson.M{"acceptedAt": now}})
	return err
}
