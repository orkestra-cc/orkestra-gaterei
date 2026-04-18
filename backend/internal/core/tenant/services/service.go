package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/orkestra/backend/internal/core/tenant/models"
	"github.com/orkestra/backend/internal/core/tenant/repository"
	"github.com/orkestra/backend/internal/shared/iface"
	"go.mongodb.org/mongo-driver/bson"
)

// Service owns tenant lifecycle and implements iface.TenantProvider.
type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// --- Provider interface ---

func (s *Service) GetOrg(ctx context.Context, orgUUID string) (*iface.Org, error) {
	o, err := s.repo.GetOrgByUUID(ctx, orgUUID)
	if err != nil {
		return nil, err
	}
	return &iface.Org{
		UUID:     o.UUID,
		Name:     o.Name,
		Slug:     o.Slug,
		Plan:     o.Plan,
		Features: o.Features,
	}, nil
}

func (s *Service) ListUserMemberships(ctx context.Context, userUUID string) ([]iface.Membership, error) {
	mbrs, err := s.repo.ListMembershipsByUser(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	out := make([]iface.Membership, 0, len(mbrs))
	for _, m := range mbrs {
		o, err := s.repo.GetOrgByUUID(ctx, m.OrgUUID)
		if err != nil {
			continue // org may be soft-deleted, skip
		}
		out = append(out, iface.Membership{
			OrgUUID: o.UUID,
			OrgName: o.Name,
			OrgSlug: o.Slug,
			Roles:   m.Roles,
			IsOwner: m.IsOwner,
		})
	}
	return out, nil
}

func (s *Service) IsMember(ctx context.Context, userUUID, orgUUID string) (bool, error) {
	_, err := s.repo.GetMembership(ctx, userUUID, orgUUID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Service) HasEntitlement(ctx context.Context, orgUUID, feature string) (bool, error) {
	o, err := s.repo.GetOrgByUUID(ctx, orgUUID)
	if err != nil {
		return false, err
	}
	return o.HasFeature(feature), nil
}

// --- Org lifecycle ---

func (s *Service) CreateOrg(ctx context.Context, ownerUUID string, input models.CreateOrgInput) (*models.Org, error) {
	slug := slugify(input.Slug)
	if slug == "" {
		slug = slugify(input.Name)
	}
	if existing, _ := s.repo.GetOrgBySlug(ctx, slug); existing != nil {
		return nil, fmt.Errorf("slug already in use: %s", slug)
	}

	plan := input.Plan
	if plan == "" {
		plan = models.PlanFree
	}
	features := defaultFeaturesForPlan(plan)

	org := &models.Org{
		UUID:          uuid.Must(uuid.NewV7()).String(),
		Name:          strings.TrimSpace(input.Name),
		Slug:          slug,
		OwnerUserUUID: ownerUUID,
		Plan:          plan,
		Features:      features,
	}

	if err := s.repo.CreateOrg(ctx, org); err != nil {
		return nil, err
	}

	// Owner is auto-enrolled as a member with the "administrator" role.
	membership := &models.Membership{
		UUID:     uuid.Must(uuid.NewV7()).String(),
		UserUUID: ownerUUID,
		OrgUUID:  org.UUID,
		Roles:    []string{"administrator"},
		IsOwner:  true,
	}
	if err := s.repo.CreateMembership(ctx, membership); err != nil {
		return nil, err
	}
	return org, nil
}

func (s *Service) UpdateOrg(ctx context.Context, orgUUID string, input models.UpdateOrgInput) error {
	update := bson.M{}
	if input.Name != nil {
		update["name"] = strings.TrimSpace(*input.Name)
	}
	if input.Slug != nil {
		slug := slugify(*input.Slug)
		if existing, _ := s.repo.GetOrgBySlug(ctx, slug); existing != nil && existing.UUID != orgUUID {
			return fmt.Errorf("slug already in use: %s", slug)
		}
		update["slug"] = slug
	}
	if input.Settings != nil {
		update["settings"] = input.Settings
	}
	if len(update) == 0 {
		return nil
	}
	return s.repo.UpdateOrg(ctx, orgUUID, update)
}

func (s *Service) UpdatePlan(ctx context.Context, orgUUID string, input models.UpdatePlanInput) error {
	features := input.Features
	if features == nil {
		features = defaultFeaturesForPlan(input.Plan)
	}
	return s.repo.UpdateOrg(ctx, orgUUID, bson.M{"plan": input.Plan, "features": features})
}

func (s *Service) DeleteOrg(ctx context.Context, orgUUID string) error {
	return s.repo.SoftDeleteOrg(ctx, orgUUID)
}

// OrgAdminView is an org plus its current member count, used by the
// platform-admin list endpoint to avoid an N+1.
type OrgAdminView struct {
	Org         *models.Org
	MemberCount int
}

// ListAllOrgs returns every org in the system with live member counts.
// Used by the platform admin tenant management page — bypasses per-org
// membership gates and is only callable via system.tenants.admin.
func (s *Service) ListAllOrgs(ctx context.Context, includeDeleted bool) ([]OrgAdminView, error) {
	orgs, err := s.repo.ListAllOrgs(ctx, includeDeleted)
	if err != nil {
		return nil, err
	}
	if len(orgs) == 0 {
		return []OrgAdminView{}, nil
	}
	uuids := make([]string, len(orgs))
	for i := range orgs {
		uuids[i] = orgs[i].UUID
	}
	counts, err := s.repo.CountMembersByOrgs(ctx, uuids)
	if err != nil {
		return nil, err
	}
	out := make([]OrgAdminView, len(orgs))
	for i := range orgs {
		o := orgs[i]
		out[i] = OrgAdminView{Org: &o, MemberCount: counts[o.UUID]}
	}
	return out, nil
}

// --- Memberships ---

func (s *Service) ListMembers(ctx context.Context, orgUUID string) ([]models.Membership, error) {
	return s.repo.ListMembershipsByOrg(ctx, orgUUID)
}

func (s *Service) RemoveMember(ctx context.Context, orgUUID, userUUID string) error {
	return s.repo.DeleteMembership(ctx, userUUID, orgUUID)
}

func (s *Service) SetMemberRoles(ctx context.Context, orgUUID, userUUID string, roles []string) error {
	return s.repo.UpdateMembershipRoles(ctx, userUUID, orgUUID, roles)
}

// --- Invites ---

// CreateInvite generates a single-use invite token, persists only its hash,
// and returns the raw token exactly once on the struct's transient Token
// field. Callers must relay the raw token to the invitee (over email or a
// copy-paste UI) immediately; after this function returns there is no way to
// recover it — the database only has the hash.
func (s *Service) CreateInvite(ctx context.Context, orgUUID, invitedBy string, input models.InviteInput) (*models.Invite, error) {
	raw, hash, err := generateInviteToken()
	if err != nil {
		return nil, fmt.Errorf("tenant: generate invite token: %w", err)
	}
	inv := &models.Invite{
		UUID:      uuid.Must(uuid.NewV7()).String(),
		OrgUUID:   orgUUID,
		Email:     strings.ToLower(strings.TrimSpace(input.Email)),
		Roles:     input.Roles,
		Token:     raw, // transient: returned to caller, not persisted
		TokenHash: hash,
		InvitedBy: invitedBy,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := s.repo.CreateInvite(ctx, inv); err != nil {
		return nil, err
	}
	return inv, nil
}

// ListInvites returns invites for an org. Caller scopes visibility: pending-only
// by default, all invites when onlyPending is false. Raw tokens are zeroed out
// before returning — they are only retrievable once at creation time.
func (s *Service) ListInvites(ctx context.Context, orgUUID string, onlyPending bool) ([]models.Invite, error) {
	invs, err := s.repo.ListInvitesByOrg(ctx, orgUUID, onlyPending)
	if err != nil {
		return nil, err
	}
	for i := range invs {
		invs[i].Token = ""
	}
	return invs, nil
}

// RevokeInvite deletes a pending invite by UUID. The orgUUID is required to
// prevent cross-org spoofing via a guessed invite UUID.
func (s *Service) RevokeInvite(ctx context.Context, orgUUID, inviteUUID string) error {
	return s.repo.DeleteInvite(ctx, orgUUID, inviteUUID)
}

func (s *Service) AcceptInvite(ctx context.Context, userUUID, token string) (*models.Org, error) {
	// Look up by hash, not plaintext — the plaintext only exists in the
	// invitee's email/UI, never in the database.
	inv, err := s.repo.GetInviteByTokenHash(ctx, hashInviteToken(token))
	if err != nil {
		return nil, err
	}
	if inv.AcceptedAt != nil {
		return nil, errors.New("invite already accepted")
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil, errors.New("invite expired")
	}
	membership := &models.Membership{
		UUID:      uuid.Must(uuid.NewV7()).String(),
		UserUUID:  userUUID,
		OrgUUID:   inv.OrgUUID,
		Roles:     inv.Roles,
		InvitedBy: inv.InvitedBy,
	}
	if err := s.repo.CreateMembership(ctx, membership); err != nil {
		return nil, err
	}
	if err := s.repo.MarkInviteAccepted(ctx, inv.UUID); err != nil {
		return nil, err
	}
	return s.repo.GetOrgByUUID(ctx, inv.OrgUUID)
}

func (s *Service) GetOrgModel(ctx context.Context, orgUUID string) (*models.Org, error) {
	return s.repo.GetOrgByUUID(ctx, orgUUID)
}

// --- Helpers ---

func defaultFeaturesForPlan(plan string) []string {
	switch plan {
	case models.PlanEnterprise:
		return []string{models.FeatureWildcard}
	case models.PlanPro:
		return []string{"billing", "documents", "company", "sales", "agents"}
	default:
		return []string{"billing", "documents"}
	}
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// generateInviteToken mirrors auth/services/password_auth_service.go's
// generateEmailToken: 32 random bytes → base64url → SHA-256 hex digest.
// The raw token is returned to the caller once; only the hash is stored.
func generateInviteToken() (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	hash = hashInviteToken(raw)
	return raw, hash, nil
}

func hashInviteToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
