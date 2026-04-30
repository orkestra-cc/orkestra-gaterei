package repository

import (
	"context"
	"errors"
	"time"

	"github.com/orkestra/backend/internal/core/tenant/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	CollTenants     = "tenants"
	CollMemberships = "tenant_memberships"
	CollInvites     = "tenant_invites"
	// CollAncestors materializes the transitive-closure hierarchy for
	// external multi-tenant clients. See ADR-0001.
	CollAncestors = "tenant_ancestors"
)

var ErrNotFound = errors.New("tenant: not found")

type Repository struct {
	db *mongo.Database
}

func New(db *mongo.Database) *Repository {
	return &Repository{db: db}
}

// --- Tenants ---

func (r *Repository) CreateTenant(ctx context.Context, t *models.Tenant) error {
	t.CreatedAt = time.Now()
	t.UpdatedAt = t.CreatedAt
	if !t.Kind.Valid() {
		t.Kind = models.TenantKindInternal
	}
	if !t.Status.Valid() {
		t.Status = models.TenantStatusActive
	}
	if t.Region == "" {
		t.Region = "eu-west"
	}
	_, err := r.db.Collection(CollTenants).InsertOne(ctx, t)
	return err
}

func (r *Repository) GetTenantByUUID(ctx context.Context, uuid string) (*models.Tenant, error) {
	var t models.Tenant
	err := r.db.Collection(CollTenants).FindOne(ctx, bson.M{"uuid": uuid, "deletedAt": nil}).Decode(&t)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	return &t, err
}

func (r *Repository) GetTenantBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	var t models.Tenant
	err := r.db.Collection(CollTenants).FindOne(ctx, bson.M{"slug": slug, "deletedAt": nil}).Decode(&t)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	return &t, err
}

func (r *Repository) UpdateTenant(ctx context.Context, uuid string, update bson.M) error {
	update["updatedAt"] = time.Now()
	res, err := r.db.Collection(CollTenants).UpdateOne(ctx, bson.M{"uuid": uuid}, bson.M{"$set": update})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) SoftDeleteTenant(ctx context.Context, uuid string) error {
	now := time.Now()
	res, err := r.db.Collection(CollTenants).UpdateOne(ctx,
		bson.M{"uuid": uuid},
		bson.M{"$set": bson.M{
			"deletedAt":  now,
			"updatedAt":  now,
			"status":     string(models.TenantStatusArchived),
			"archivedAt": now,
		}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// ListAllTenants returns every tenant in the system for platform-admin views.
func (r *Repository) ListAllTenants(ctx context.Context, includeDeleted bool) ([]models.Tenant, error) {
	filter := bson.M{}
	if !includeDeleted {
		filter["deletedAt"] = nil
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := r.db.Collection(CollTenants).Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []models.Tenant
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListTenantsByKind returns every tenant of a specific tier. Used by platform
// operators to inspect the internal/external split.
func (r *Repository) ListTenantsByKind(ctx context.Context, kind models.TenantKind, includeDeleted bool) ([]models.Tenant, error) {
	filter := bson.M{"kind": string(kind)}
	if !includeDeleted {
		filter["deletedAt"] = nil
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := r.db.Collection(CollTenants).Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []models.Tenant
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// TenantListFilter narrows platform-admin tenant lookups. All fields are
// optional; zero values mean "no filter on that axis". Used by the admin list
// endpoint to back the split between Internal Operations and Client
// Management.
type TenantListFilter struct {
	// Kind, when non-empty, restricts to one tier. Must be a valid TenantKind.
	Kind models.TenantKind
	// ParentTenantUUID, when non-nil, restricts to direct children of that
	// tenant (exact match on the parentTenantUUID field). Pointer so callers
	// can distinguish "no filter" from "roots only" via RootsOnly.
	ParentTenantUUID *string
	// RootsOnly, when true, restricts to tenants that have no parent. Useful
	// for the Clients list page where divisions should not appear. Mutually
	// exclusive with ParentTenantUUID — if both are set, RootsOnly wins.
	RootsOnly bool
	// IncludeDeleted returns soft-deleted rows when true.
	IncludeDeleted bool
}

// ListTenants is the general-purpose admin list with optional filters. Sorts
// by createdAt descending.
func (r *Repository) ListTenants(ctx context.Context, f TenantListFilter) ([]models.Tenant, error) {
	filter := bson.M{}
	if f.Kind != "" {
		filter["kind"] = string(f.Kind)
	}
	switch {
	case f.RootsOnly:
		filter["$or"] = []bson.M{
			{"parentTenantUUID": bson.M{"$exists": false}},
			{"parentTenantUUID": nil},
			{"parentTenantUUID": ""},
		}
	case f.ParentTenantUUID != nil:
		filter["parentTenantUUID"] = *f.ParentTenantUUID
	}
	if !f.IncludeDeleted {
		filter["deletedAt"] = nil
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := r.db.Collection(CollTenants).Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []models.Tenant
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateTenantStatus transitions a tenant to a new lifecycle state.
func (r *Repository) UpdateTenantStatus(ctx context.Context, uuid string, status models.TenantStatus) error {
	update := bson.M{"status": string(status), "updatedAt": time.Now()}
	if status == models.TenantStatusArchived {
		update["archivedAt"] = time.Now()
	}
	if status == models.TenantStatusPurged {
		update["purgedAt"] = time.Now()
	}
	res, err := r.db.Collection(CollTenants).UpdateOne(ctx, bson.M{"uuid": uuid}, bson.M{"$set": update})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// CountMembersByTenants returns a map tenantUUID -> member count for the
// given tenants. Uses a single $group aggregation on tenant_memberships so
// list views can attach member counts without N round trips.
func (r *Repository) CountMembersByTenants(ctx context.Context, tenantUUIDs []string) (map[string]int, error) {
	out := make(map[string]int, len(tenantUUIDs))
	if len(tenantUUIDs) == 0 {
		return out, nil
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"tenantId": bson.M{"$in": tenantUUIDs}}}},
		{{Key: "$group", Value: bson.M{"_id": "$tenantId", "count": bson.M{"$sum": 1}}}},
	}
	cur, err := r.db.Collection(CollMemberships).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var row struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, err
		}
		out[row.ID] = row.Count
	}
	return out, cur.Err()
}

// --- Memberships ---

func (r *Repository) CreateMembership(ctx context.Context, m *models.TenantMembership) error {
	m.JoinedAt = time.Now()
	_, err := r.db.Collection(CollMemberships).InsertOne(ctx, m)
	return err
}

func (r *Repository) DeleteMembership(ctx context.Context, userUUID, tenantUUID string) error {
	_, err := r.db.Collection(CollMemberships).DeleteOne(ctx, bson.M{"userUUID": userUUID, "tenantId": tenantUUID})
	return err
}

func (r *Repository) ListMembershipsByUser(ctx context.Context, userUUID string) ([]models.TenantMembership, error) {
	cur, err := r.db.Collection(CollMemberships).Find(ctx, bson.M{
		"userUUID": userUUID,
		"$or":      []bson.M{{"expiresAt": nil}, {"expiresAt": bson.M{"$gt": time.Now()}}},
	})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []models.TenantMembership
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) ListMembershipsByTenant(ctx context.Context, tenantUUID string) ([]models.TenantMembership, error) {
	cur, err := r.db.Collection(CollMemberships).Find(ctx, bson.M{"tenantId": tenantUUID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []models.TenantMembership
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) GetMembership(ctx context.Context, userUUID, tenantUUID string) (*models.TenantMembership, error) {
	var m models.TenantMembership
	err := r.db.Collection(CollMemberships).FindOne(ctx, bson.M{"userUUID": userUUID, "tenantId": tenantUUID}).Decode(&m)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	return &m, err
}

func (r *Repository) UpdateMembershipRoles(ctx context.Context, userUUID, tenantUUID string, roles []string) error {
	_, err := r.db.Collection(CollMemberships).UpdateOne(ctx,
		bson.M{"userUUID": userUUID, "tenantId": tenantUUID},
		bson.M{"$set": bson.M{"roles": roles}})
	return err
}

// --- Invites ---

func (r *Repository) CreateInvite(ctx context.Context, inv *models.TenantInvite) error {
	inv.CreatedAt = time.Now()
	_, err := r.db.Collection(CollInvites).InsertOne(ctx, inv)
	return err
}

// GetInviteByTokenHash looks up an invite by the SHA-256 hash of the raw
// token. The caller computes the hash from the plaintext supplied by the
// user; we never store or query the plaintext.
func (r *Repository) GetInviteByTokenHash(ctx context.Context, tokenHash string) (*models.TenantInvite, error) {
	var inv models.TenantInvite
	err := r.db.Collection(CollInvites).FindOne(ctx, bson.M{"tokenHash": tokenHash}).Decode(&inv)
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

// ListInvitesByTenant returns invites for a tenant. When onlyPending is
// true, accepted and expired invites are filtered out.
func (r *Repository) ListInvitesByTenant(ctx context.Context, tenantUUID string, onlyPending bool) ([]models.TenantInvite, error) {
	filter := bson.M{"tenantId": tenantUUID}
	if onlyPending {
		filter["acceptedAt"] = nil
		filter["expiresAt"] = bson.M{"$gt": time.Now()}
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := r.db.Collection(CollInvites).Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []models.TenantInvite
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteInvite hard-deletes an invite scoped to the given tenant.
func (r *Repository) DeleteInvite(ctx context.Context, tenantUUID, inviteUUID string) error {
	res, err := r.db.Collection(CollInvites).DeleteOne(ctx, bson.M{"uuid": inviteUUID, "tenantId": tenantUUID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Tenant hierarchy (closure table) ---

// InsertSelfAncestor inserts the depth=0 self-row for a freshly created
// tenant. Called inside CreateTenant (via the service).
func (r *Repository) InsertSelfAncestor(ctx context.Context, tenantUUID string) error {
	_, err := r.db.Collection(CollAncestors).InsertOne(ctx, &models.TenantAncestor{
		DescendantUUID: tenantUUID,
		AncestorUUID:   tenantUUID,
		Depth:          0,
		CreatedAt:      time.Now(),
	})
	return err
}

// AttachToParent populates the closure table for a new descendant by copying
// every ancestor row of the parent and adding one for the direct parent.
// Idempotent-unsafe — call once per tenant creation.
func (r *Repository) AttachToParent(ctx context.Context, childUUID, parentUUID string) error {
	cur, err := r.db.Collection(CollAncestors).Find(ctx, bson.M{"descendantUUID": parentUUID})
	if err != nil {
		return err
	}
	defer cur.Close(ctx)
	now := time.Now()
	var rows []any
	for cur.Next(ctx) {
		var anc models.TenantAncestor
		if err := cur.Decode(&anc); err != nil {
			return err
		}
		rows = append(rows, &models.TenantAncestor{
			DescendantUUID: childUUID,
			AncestorUUID:   anc.AncestorUUID,
			Depth:          anc.Depth + 1,
			CreatedAt:      now,
		})
	}
	if err := cur.Err(); err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	_, err = r.db.Collection(CollAncestors).InsertMany(ctx, rows)
	return err
}

// ListAncestors returns every ancestor row of a tenant, sorted by depth
// ascending (self first, root last). Includes the self-row at depth=0.
func (r *Repository) ListAncestors(ctx context.Context, tenantUUID string) ([]models.TenantAncestor, error) {
	opts := options.Find().SetSort(bson.D{{Key: "depth", Value: 1}})
	cur, err := r.db.Collection(CollAncestors).Find(ctx, bson.M{"descendantUUID": tenantUUID}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []models.TenantAncestor
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListDescendantUUIDs returns the UUIDs of every descendant (including self).
func (r *Repository) ListDescendantUUIDs(ctx context.Context, ancestorUUID string) ([]string, error) {
	cur, err := r.db.Collection(CollAncestors).Find(ctx, bson.M{"ancestorUUID": ancestorUUID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []string
	for cur.Next(ctx) {
		var anc models.TenantAncestor
		if err := cur.Decode(&anc); err != nil {
			return nil, err
		}
		out = append(out, anc.DescendantUUID)
	}
	return out, cur.Err()
}

// IsAncestorOf reports whether ancestorUUID is an ancestor (at any depth,
// including equal) of descendantUUID. O(1) indexed lookup.
func (r *Repository) IsAncestorOf(ctx context.Context, ancestorUUID, descendantUUID string) (bool, error) {
	err := r.db.Collection(CollAncestors).FindOne(ctx, bson.M{
		"ancestorUUID":   ancestorUUID,
		"descendantUUID": descendantUUID,
	}).Err()
	if err == nil {
		return true, nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return false, err
}
