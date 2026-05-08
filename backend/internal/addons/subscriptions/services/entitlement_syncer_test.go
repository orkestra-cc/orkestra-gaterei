package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/orkestra/backend/internal/addons/subscriptions/models"
	"github.com/orkestra/backend/internal/addons/subscriptions/repository"
	"github.com/orkestra/backend/internal/shared/iface"
)

// stubAccessProvider records Grant/Revoke calls for assertion. Implements
// iface.AccessProvider — the polymorphic capability surface the syncer
// consumes.
type stubAccessProvider struct {
	grants   []iface.GrantCapabilityInput
	revokes  []stubRevoke
	grantErr error
}

type stubRevoke struct {
	owner        iface.Owner
	capabilityID string
}

func (s *stubAccessProvider) HasCapability(context.Context, iface.Owner, string) (bool, error) {
	return false, nil
}
func (s *stubAccessProvider) GrantCapability(_ context.Context, in iface.GrantCapabilityInput) error {
	s.grants = append(s.grants, in)
	return s.grantErr
}
func (s *stubAccessProvider) RevokeCapability(_ context.Context, owner iface.Owner, capabilityID string) error {
	s.revokes = append(s.revokes, stubRevoke{owner: owner, capabilityID: capabilityID})
	return nil
}
func (s *stubAccessProvider) ListCapabilityIDs(context.Context, iface.Owner) ([]string, error) {
	return nil, nil
}

// stubServiceRepo satisfies repository.ServiceRepository with an in-memory map.
type stubServiceRepo struct {
	byUUID map[string]*models.Service
}

func (s *stubServiceRepo) Create(context.Context, *models.Service) error { return nil }
func (s *stubServiceRepo) GetByUUID(_ context.Context, uuid string) (*models.Service, error) {
	if svc, ok := s.byUUID[uuid]; ok {
		return svc, nil
	}
	return nil, errors.New("not found")
}
func (s *stubServiceRepo) GetByCode(context.Context, string) (*models.Service, error) { return nil, nil }
func (s *stubServiceRepo) List(context.Context, repository.ServiceFilters) ([]models.Service, error) {
	return nil, nil
}
func (s *stubServiceRepo) Update(context.Context, *models.Service) error { return nil }
func (s *stubServiceRepo) Delete(context.Context, string) error          { return nil }

// nopLogger discards output so tests stay quiet.
func nopLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// serviceWithTier builds a Service with a single tier carrying the given
// capability IDs. The tier code is fixed as "std" so tests reference it
// without boilerplate.
func serviceWithTier(uuid string, caps ...string) *models.Service {
	return &models.Service{
		UUID: uuid,
		Code: "svc-" + uuid,
		PricingTiers: []models.PricingTier{
			{Code: "std", Capabilities: caps},
		},
	}
}

func tenantSub(uuid, tenantUUID, serviceUUID string) *models.Subscription {
	return &models.Subscription{
		UUID:        uuid,
		OwnerKind:   iface.OwnerKindTenant,
		OwnerUUID:   tenantUUID,
		ServiceUUID: serviceUUID,
		TierCode:    "std",
	}
}

func TestSyncer_OnActivateGrantsEveryCapability(t *testing.T) {
	repo := &stubServiceRepo{byUUID: map[string]*models.Service{
		"svc-1": serviceWithTier("svc-1", "rag.query", "agents.run"),
	}}
	ap := &stubAccessProvider{}
	syncer := NewEntitlementSyncer(repo, ap, nil, false, nopLogger())

	periodEnd := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	sub := tenantSub("sub-1", "tenant-1", "svc-1")
	sub.CurrentPeriodEnd = periodEnd
	syncer.OnActivate(context.Background(), sub)

	if len(ap.grants) != 2 {
		t.Fatalf("want 2 grants, got %d: %+v", len(ap.grants), ap.grants)
	}
	for _, g := range ap.grants {
		if g.Owner.Kind != iface.OwnerKindTenant || g.Owner.UUID != "tenant-1" {
			t.Errorf("grant owner: %+v", g.Owner)
		}
		if g.Source != CapabilitySourceSubscription {
			t.Errorf("grant source: %s", g.Source)
		}
		if g.SourceRef != "sub-1" {
			t.Errorf("grant source ref: %s", g.SourceRef)
		}
		if g.ExpiresAt == nil || !g.ExpiresAt.Equal(periodEnd) {
			t.Errorf("grant expiry: %v want %v", g.ExpiresAt, periodEnd)
		}
	}
}

func TestSyncer_OnActivateGrantsForUserOwner(t *testing.T) {
	// Self-registered client subscriptions carry OwnerKind="user" — the
	// grant must land on Owner{Kind:"user", UUID:userUUID} not on a tenant.
	repo := &stubServiceRepo{byUUID: map[string]*models.Service{
		"svc-1": serviceWithTier("svc-1", "rag.query"),
	}}
	ap := &stubAccessProvider{}
	syncer := NewEntitlementSyncer(repo, ap, nil, false, nopLogger())

	sub := &models.Subscription{
		UUID:        "sub-1",
		OwnerKind:   iface.OwnerKindUser,
		OwnerUUID:   "user-9",
		ServiceUUID: "svc-1",
		TierCode:    "std",
	}
	syncer.OnActivate(context.Background(), sub)

	if len(ap.grants) != 1 {
		t.Fatalf("want 1 grant, got %d", len(ap.grants))
	}
	if ap.grants[0].Owner.Kind != iface.OwnerKindUser || ap.grants[0].Owner.UUID != "user-9" {
		t.Errorf("user-owned grant: %+v", ap.grants[0].Owner)
	}
}

func TestSyncer_OnActivateSkipsWhenOwnerMissing(t *testing.T) {
	// Subscription row missing owner — must not panic, must not grant.
	repo := &stubServiceRepo{byUUID: map[string]*models.Service{
		"svc-1": serviceWithTier("svc-1", "rag.query"),
	}}
	ap := &stubAccessProvider{}
	syncer := NewEntitlementSyncer(repo, ap, nil, false, nopLogger())

	sub := &models.Subscription{
		UUID:        "sub-1",
		ServiceUUID: "svc-1",
		TierCode:    "std",
	}
	syncer.OnActivate(context.Background(), sub)

	if len(ap.grants) != 0 {
		t.Fatalf("expected zero grants when owner is empty, got %d", len(ap.grants))
	}
}

func TestSyncer_OnActivateNoopWhenTierHasNoCapabilities(t *testing.T) {
	repo := &stubServiceRepo{byUUID: map[string]*models.Service{
		"svc-1": serviceWithTier("svc-1"), // empty caps
	}}
	ap := &stubAccessProvider{}
	syncer := NewEntitlementSyncer(repo, ap, nil, false, nopLogger())
	syncer.OnActivate(context.Background(), tenantSub("sub-1", "tenant-1", "svc-1"))
	if len(ap.grants) != 0 {
		t.Fatalf("expected zero grants, got %d", len(ap.grants))
	}
}

func TestSyncer_OnDeactivateRevokesEveryCapability(t *testing.T) {
	repo := &stubServiceRepo{byUUID: map[string]*models.Service{
		"svc-1": serviceWithTier("svc-1", "rag.query", "agents.run"),
	}}
	ap := &stubAccessProvider{}
	syncer := NewEntitlementSyncer(repo, ap, nil, false, nopLogger())

	syncer.OnDeactivate(context.Background(), tenantSub("sub-1", "tenant-1", "svc-1"))

	if len(ap.revokes) != 2 {
		t.Fatalf("want 2 revokes, got %d: %+v", len(ap.revokes), ap.revokes)
	}
	if ap.revokes[0].owner.Kind != iface.OwnerKindTenant || ap.revokes[0].owner.UUID != "tenant-1" {
		t.Errorf("revoke owner: %+v", ap.revokes[0].owner)
	}
}

func TestSyncer_NilAccessProviderIsNoop(t *testing.T) {
	// Degraded mode: no tenant module registered. Syncer must not panic;
	// subscription lifecycle should continue producing its other side
	// effects unaffected.
	syncer := NewEntitlementSyncer(&stubServiceRepo{}, nil, nil, false, nopLogger())
	syncer.OnActivate(context.Background(), tenantSub("sub-1", "tenant-1", "svc-1"))
	syncer.OnDeactivate(context.Background(), tenantSub("sub-1", "tenant-1", "svc-1"))
	// No assertions needed — the test passes if we didn't panic.
}

func TestSyncer_OnActivateContinuesOnIndividualGrantError(t *testing.T) {
	// A transient failure granting one capability must not block the other
	// capabilities in the same tier from landing. The logger warn path is
	// exercised but we only assert call count here.
	repo := &stubServiceRepo{byUUID: map[string]*models.Service{
		"svc-1": serviceWithTier("svc-1", "rag.query", "agents.run"),
	}}
	ap := &stubAccessProvider{grantErr: errors.New("transient")}
	syncer := NewEntitlementSyncer(repo, ap, nil, false, nopLogger())

	syncer.OnActivate(context.Background(), tenantSub("sub-1", "tenant-1", "svc-1"))
	if len(ap.grants) != 2 {
		t.Fatalf("syncer must attempt every grant even when earlier ones fail, got %d", len(ap.grants))
	}
}

// stubSyncerTenantProvider exposes EnsureTenantForUser to the syncer's
// Phase 2 path. Other TenantProvider methods are stubs — only the lazy
// path exercises this provider through the syncer.
type stubSyncerTenantProvider struct {
	personalByUser map[string]*iface.Tenant
	ensureCalls    []string
	ensureErr      error
}

func (s *stubSyncerTenantProvider) GetTenant(context.Context, string) (*iface.Tenant, error) {
	return nil, nil
}
func (s *stubSyncerTenantProvider) ListUserMemberships(context.Context, string) ([]iface.TenantMembership, error) {
	return nil, nil
}
func (s *stubSyncerTenantProvider) IsMember(context.Context, string, string) (bool, error) {
	return false, nil
}
func (s *stubSyncerTenantProvider) ActivateTenant(context.Context, string) error { return nil }
func (s *stubSyncerTenantProvider) SetTenantStripeCustomerID(context.Context, string, string) error {
	return nil
}
func (s *stubSyncerTenantProvider) EnsureTenantForUser(_ context.Context, userUUID string) (*iface.Tenant, error) {
	s.ensureCalls = append(s.ensureCalls, userUUID)
	if s.ensureErr != nil {
		return nil, s.ensureErr
	}
	return s.personalByUser[userUUID], nil
}

// TestSyncer_LazyTenantFlagOffKeepsUserOwner asserts the default behavior:
// user-owned subscriptions still project onto the user owner. The flag is
// off until Phase 3 flips it.
func TestSyncer_LazyTenantFlagOffKeepsUserOwner(t *testing.T) {
	repo := &stubServiceRepo{byUUID: map[string]*models.Service{
		"svc-1": serviceWithTier("svc-1", "rag.query"),
	}}
	ap := &stubAccessProvider{}
	tp := &stubSyncerTenantProvider{
		personalByUser: map[string]*iface.Tenant{"user-9": {UUID: "tenant-personal-9"}},
	}
	syncer := NewEntitlementSyncer(repo, ap, tp, false, nopLogger())

	syncer.OnActivate(context.Background(), &models.Subscription{
		UUID: "sub-1", OwnerKind: iface.OwnerKindUser, OwnerUUID: "user-9",
		ServiceUUID: "svc-1", TierCode: "std",
	})

	if len(tp.ensureCalls) != 0 {
		t.Fatalf("EnsureTenantForUser must not be called when flag is off, got %v", tp.ensureCalls)
	}
	if len(ap.grants) != 1 || ap.grants[0].Owner.Kind != iface.OwnerKindUser || ap.grants[0].Owner.UUID != "user-9" {
		t.Fatalf("flag-off grant must stay on user owner, got %+v", ap.grants)
	}
}

// TestSyncer_LazyTenantFlagOnRedirectsToPersonalTenant asserts that when
// the flag is on the activation projects onto the user's personal tenant
// (resolved via EnsureTenantForUser) rather than the user owner — this is
// the projection-rewriting half of the Phase 2 contract.
func TestSyncer_LazyTenantFlagOnRedirectsToPersonalTenant(t *testing.T) {
	repo := &stubServiceRepo{byUUID: map[string]*models.Service{
		"svc-1": serviceWithTier("svc-1", "rag.query"),
	}}
	ap := &stubAccessProvider{}
	tp := &stubSyncerTenantProvider{
		personalByUser: map[string]*iface.Tenant{"user-9": {UUID: "tenant-personal-9"}},
	}
	syncer := NewEntitlementSyncer(repo, ap, tp, true, nopLogger())

	syncer.OnActivate(context.Background(), &models.Subscription{
		UUID: "sub-1", OwnerKind: iface.OwnerKindUser, OwnerUUID: "user-9",
		ServiceUUID: "svc-1", TierCode: "std",
	})

	if len(tp.ensureCalls) != 1 || tp.ensureCalls[0] != "user-9" {
		t.Fatalf("EnsureTenantForUser calls: %v", tp.ensureCalls)
	}
	if len(ap.grants) != 1 {
		t.Fatalf("expected 1 grant, got %d", len(ap.grants))
	}
	got := ap.grants[0].Owner
	if got.Kind != iface.OwnerKindTenant || got.UUID != "tenant-personal-9" {
		t.Fatalf("flag-on grant must target the personal tenant, got %+v", got)
	}
}

// TestSyncer_LazyTenantFlagOnLeavesTenantSubsAlone asserts that tenant-owned
// subscriptions are pass-through under the flag — no spurious EnsureTenantForUser
// call, owner stays unchanged.
func TestSyncer_LazyTenantFlagOnLeavesTenantSubsAlone(t *testing.T) {
	repo := &stubServiceRepo{byUUID: map[string]*models.Service{
		"svc-1": serviceWithTier("svc-1", "rag.query"),
	}}
	ap := &stubAccessProvider{}
	tp := &stubSyncerTenantProvider{}
	syncer := NewEntitlementSyncer(repo, ap, tp, true, nopLogger())

	syncer.OnActivate(context.Background(), tenantSub("sub-1", "tenant-X", "svc-1"))

	if len(tp.ensureCalls) != 0 {
		t.Fatalf("tenant-owned subs must not trigger EnsureTenantForUser, got %v", tp.ensureCalls)
	}
	if len(ap.grants) != 1 || ap.grants[0].Owner.UUID != "tenant-X" {
		t.Fatalf("tenant-owned grant: %+v", ap.grants)
	}
}

// TestSyncer_LazyTenantFlagOnEnsureErrorSkips asserts that when EnsureTenantForUser
// fails the syncer logs and skips rather than misattributing the grant to the
// raw user owner — better to drop the projection update than to write to the
// wrong aggregate.
func TestSyncer_LazyTenantFlagOnEnsureErrorSkips(t *testing.T) {
	repo := &stubServiceRepo{byUUID: map[string]*models.Service{
		"svc-1": serviceWithTier("svc-1", "rag.query"),
	}}
	ap := &stubAccessProvider{}
	tp := &stubSyncerTenantProvider{ensureErr: errors.New("mongo down")}
	syncer := NewEntitlementSyncer(repo, ap, tp, true, nopLogger())

	syncer.OnActivate(context.Background(), &models.Subscription{
		UUID: "sub-1", OwnerKind: iface.OwnerKindUser, OwnerUUID: "user-9",
		ServiceUUID: "svc-1", TierCode: "std",
	})

	if len(ap.grants) != 0 {
		t.Fatalf("expected zero grants when EnsureTenantForUser errors, got %d", len(ap.grants))
	}
}
