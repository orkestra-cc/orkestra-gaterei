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
	syncer := NewEntitlementSyncer(repo, ap, nil, nopLogger())

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
	syncer := NewEntitlementSyncer(repo, ap, nil, nopLogger())

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
	syncer := NewEntitlementSyncer(repo, ap, nil, nopLogger())

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
	syncer := NewEntitlementSyncer(repo, ap, nil, nopLogger())
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
	syncer := NewEntitlementSyncer(repo, ap, nil, nopLogger())

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
	syncer := NewEntitlementSyncer(&stubServiceRepo{}, nil, nil, nopLogger())
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
	syncer := NewEntitlementSyncer(repo, ap, nil, nopLogger())

	syncer.OnActivate(context.Background(), tenantSub("sub-1", "tenant-1", "svc-1"))
	if len(ap.grants) != 2 {
		t.Fatalf("syncer must attempt every grant even when earlier ones fail, got %d", len(ap.grants))
	}
}
