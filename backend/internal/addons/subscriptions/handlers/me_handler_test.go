package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/orkestra/backend/internal/addons/subscriptions/models"
	"github.com/orkestra/backend/internal/addons/subscriptions/repository"
	"github.com/orkestra/backend/internal/addons/subscriptions/services"
	"github.com/orkestra/backend/internal/shared/iface"
	"github.com/orkestra/backend/internal/testkit"
)

// stubSubRepo is a minimal SubscriptionRepository that records every List
// filter and replays a fixture set indexed by polymorphic owner. The other
// methods are intentionally not implemented — MeList only exercises List
// and a green test must not silently drift into hitting the unimplemented
// surface.
type stubSubRepo struct {
	byOwner map[iface.Owner][]models.Subscription
	calls   []repository.SubscriptionFilters
}

func newStubSubRepo(rows ...models.Subscription) *stubSubRepo {
	r := &stubSubRepo{byOwner: map[iface.Owner][]models.Subscription{}}
	for _, s := range rows {
		r.byOwner[s.Owner()] = append(r.byOwner[s.Owner()], s)
	}
	return r
}

func (r *stubSubRepo) Create(context.Context, *models.Subscription) error { return nil }
func (r *stubSubRepo) GetByUUID(context.Context, string) (*models.Subscription, error) {
	return nil, errors.New("not implemented")
}
func (r *stubSubRepo) List(_ context.Context, f repository.SubscriptionFilters) ([]models.Subscription, error) {
	r.calls = append(r.calls, f)
	if f.Owner.IsZero() {
		out := []models.Subscription{}
		for _, rows := range r.byOwner {
			out = append(out, rows...)
		}
		return out, nil
	}
	rows := r.byOwner[f.Owner]
	out := make([]models.Subscription, 0, len(rows))
	for _, s := range rows {
		if f.Status != "" && s.Status != f.Status {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}
func (r *stubSubRepo) FindDue(context.Context, time.Time) ([]models.Subscription, error) {
	return nil, nil
}
func (r *stubSubRepo) FindByOwner(context.Context, iface.Owner) ([]models.Subscription, error) {
	return nil, nil
}
func (r *stubSubRepo) UpdateStatus(context.Context, string, models.SubStatus) error { return nil }
func (r *stubSubRepo) Update(context.Context, *models.Subscription) error           { return nil }
func (r *stubSubRepo) Delete(context.Context, string) error                         { return nil }

// stubSvcRepoMin satisfies repository.ServiceRepository — MeList does not
// hit any of these methods, so an outright errors implementation suffices to
// fail loud if the handler ever drifts into reading the catalog.
type stubSvcRepoMin struct{}

func (stubSvcRepoMin) Create(context.Context, *models.Service) error                 { return nil }
func (stubSvcRepoMin) GetByUUID(context.Context, string) (*models.Service, error)    { return nil, nil }
func (stubSvcRepoMin) GetByCode(context.Context, string) (*models.Service, error)    { return nil, nil }
func (stubSvcRepoMin) Update(context.Context, *models.Service) error                 { return nil }
func (stubSvcRepoMin) Delete(context.Context, string) error                          { return nil }
func (stubSvcRepoMin) List(context.Context, repository.ServiceFilters) ([]models.Service, error) {
	return nil, nil
}

// stubTenantProvider returns a controlled membership set per userUUID and
// records EnsureTenantForUser invocations. All other TenantProvider methods
// are stubs — MeList only touches ListUserMemberships and (under the Phase 2
// flag) EnsureTenantForUser.
type stubTenantProvider struct {
	memberships  map[string][]iface.TenantMembership
	personalByUser map[string]*iface.Tenant
	ensureCalls  []string
}

func (s *stubTenantProvider) GetTenant(context.Context, string) (*iface.Tenant, error) {
	return nil, nil
}
func (s *stubTenantProvider) ListUserMemberships(_ context.Context, userUUID string) ([]iface.TenantMembership, error) {
	return s.memberships[userUUID], nil
}
func (s *stubTenantProvider) IsMember(context.Context, string, string) (bool, error) {
	return false, nil
}
func (s *stubTenantProvider) ActivateTenant(context.Context, string) error           { return nil }
func (s *stubTenantProvider) SetTenantStripeCustomerID(context.Context, string, string) error {
	return nil
}
func (s *stubTenantProvider) EnsureTenantForUser(_ context.Context, userUUID string) (*iface.Tenant, error) {
	s.ensureCalls = append(s.ensureCalls, userUUID)
	if s.personalByUser == nil {
		return nil, nil
	}
	return s.personalByUser[userUUID], nil
}

// nopLogger keeps the test output quiet. Reused across tests in this file.
func nopLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// newHandlerForList wires a SubscriptionHandler with just the deps MeList
// touches — no renewal/invoice/activity surface needed.
func newHandlerForList(repo repository.SubscriptionRepository, tp iface.TenantProvider) *SubscriptionHandler {
	subs := services.NewSubscriptionService(repo, stubSvcRepoMin{}, nil, nil, tp, nopLogger())
	return NewSubscriptionHandler(subs, nil, nil, nil, tp, false)
}

func userSub(uuid, userUUID string, status models.SubStatus) models.Subscription {
	return models.Subscription{
		UUID:      uuid,
		OwnerKind: iface.OwnerKindUser,
		OwnerUUID: userUUID,
		Status:    status,
	}
}

func tenantSub(uuid, tenantUUID string, status models.SubStatus) models.Subscription {
	return models.Subscription{
		UUID:      uuid,
		OwnerKind: iface.OwnerKindTenant,
		OwnerUUID: tenantUUID,
		Status:    status,
	}
}

func TestMeList_AnonymousReturns401(t *testing.T) {
	h := newHandlerForList(newStubSubRepo(), &stubTenantProvider{})
	_, err := h.MeList(context.Background(), &MeListSubscriptionsRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	se, ok := err.(huma.StatusError)
	if !ok {
		t.Fatalf("error type: %T (%v)", err, err)
	}
	if got := se.GetStatus(); got != 401 {
		t.Fatalf("status: %d, want 401", got)
	}
}

func TestMeList_UserOnlyReturnsUserOwnedRows(t *testing.T) {
	repo := newStubSubRepo(
		userSub("u-1", "user-A", models.SubActive),
		tenantSub("t-1", "tenant-X", models.SubActive),
	)
	tp := &stubTenantProvider{memberships: map[string][]iface.TenantMembership{
		"user-A": nil, // no memberships
	}}
	h := newHandlerForList(repo, tp)

	id := testkit.NewIdentity("user-A", "a@example.com", "user")
	ctx := id.ContextFor(context.Background(), "-")

	resp, err := h.MeList(ctx, &MeListSubscriptionsRequest{})
	if err != nil {
		t.Fatalf("MeList: %v", err)
	}
	if resp.Body.Total != 1 || resp.Body.Items[0].UUID != "u-1" {
		t.Fatalf("expected only user-owned u-1, got %+v", resp.Body.Items)
	}
}

func TestMeList_OwnedTenantFansIn(t *testing.T) {
	repo := newStubSubRepo(
		userSub("u-1", "user-A", models.SubActive),
		tenantSub("t-1", "tenant-X", models.SubActive),
		tenantSub("t-2", "tenant-Y", models.SubActive), // not owned
	)
	tp := &stubTenantProvider{memberships: map[string][]iface.TenantMembership{
		"user-A": {
			{TenantUUID: "tenant-X", IsOwner: true},
			{TenantUUID: "tenant-Y", IsOwner: false}, // member but not owner
		},
	}}
	h := newHandlerForList(repo, tp)

	ctx := testkit.NewIdentity("user-A", "a@example.com", "user").ContextFor(context.Background(), "-")
	resp, err := h.MeList(ctx, &MeListSubscriptionsRequest{})
	if err != nil {
		t.Fatalf("MeList: %v", err)
	}
	if resp.Body.Total != 2 {
		t.Fatalf("expected 2 rows (user + owned tenant), got %d: %+v", resp.Body.Total, resp.Body.Items)
	}
	got := map[string]bool{}
	for _, s := range resp.Body.Items {
		got[s.UUID] = true
	}
	if !got["u-1"] || !got["t-1"] || got["t-2"] {
		t.Fatalf("unexpected fan-out shape: %+v", got)
	}
}

func TestMeList_FilterUnownedPrincipalReturnsEmpty(t *testing.T) {
	// Caller asks for a tenant they do not own — the handler must reply
	// with zero rows and not leak existence by erroring out.
	repo := newStubSubRepo(tenantSub("t-1", "tenant-Z", models.SubActive))
	tp := &stubTenantProvider{memberships: map[string][]iface.TenantMembership{
		"user-A": {{TenantUUID: "tenant-X", IsOwner: true}},
	}}
	h := newHandlerForList(repo, tp)

	ctx := testkit.NewIdentity("user-A", "a@example.com", "user").ContextFor(context.Background(), "-")
	resp, err := h.MeList(ctx, &MeListSubscriptionsRequest{
		OwnerKind: string(iface.OwnerKindTenant),
		OwnerUUID: "tenant-Z",
	})
	if err != nil {
		t.Fatalf("MeList: %v", err)
	}
	if resp.Body.Total != 0 {
		t.Fatalf("expected 0 rows for unowned-tenant filter, got %d", resp.Body.Total)
	}
}

func TestMeList_LegacyTenantAliasResolves(t *testing.T) {
	repo := newStubSubRepo(
		userSub("u-1", "user-A", models.SubActive),
		tenantSub("t-1", "tenant-X", models.SubActive),
	)
	tp := &stubTenantProvider{memberships: map[string][]iface.TenantMembership{
		"user-A": {{TenantUUID: "tenant-X", IsOwner: true}},
	}}
	h := newHandlerForList(repo, tp)

	ctx := testkit.NewIdentity("user-A", "a@example.com", "user").ContextFor(context.Background(), "-")
	resp, err := h.MeList(ctx, &MeListSubscriptionsRequest{TenantUUID: "tenant-X"})
	if err != nil {
		t.Fatalf("MeList: %v", err)
	}
	if resp.Body.Total != 1 || resp.Body.Items[0].UUID != "t-1" {
		t.Fatalf("legacy alias mismatch: %+v", resp.Body.Items)
	}
}

func TestMeList_StatusFilterPropagates(t *testing.T) {
	repo := newStubSubRepo(
		userSub("u-active", "user-A", models.SubActive),
		userSub("u-cancelled", "user-A", models.SubCancelled),
	)
	tp := &stubTenantProvider{memberships: map[string][]iface.TenantMembership{"user-A": nil}}
	h := newHandlerForList(repo, tp)

	ctx := testkit.NewIdentity("user-A", "a@example.com", "user").ContextFor(context.Background(), "-")
	resp, err := h.MeList(ctx, &MeListSubscriptionsRequest{Status: string(models.SubActive)})
	if err != nil {
		t.Fatalf("MeList: %v", err)
	}
	if resp.Body.Total != 1 || resp.Body.Items[0].UUID != "u-active" {
		t.Fatalf("status filter not honored: %+v", resp.Body.Items)
	}
	for _, c := range repo.calls {
		if c.Status != models.SubActive {
			t.Fatalf("repo.List called without status filter: %+v", c)
		}
	}
}

func TestMeList_NilTenantProviderStillReturnsUserRows(t *testing.T) {
	// Subscriptions module must not hard-require the tenant module — the
	// caller's user identity alone is a valid scope.
	repo := newStubSubRepo(userSub("u-1", "user-A", models.SubActive))
	h := newHandlerForList(repo, nil)

	ctx := testkit.NewIdentity("user-A", "a@example.com", "user").ContextFor(context.Background(), "-")
	resp, err := h.MeList(ctx, &MeListSubscriptionsRequest{})
	if err != nil {
		t.Fatalf("MeList: %v", err)
	}
	if resp.Body.Total != 1 || resp.Body.Items[0].UUID != "u-1" {
		t.Fatalf("expected user-owned row, got %+v", resp.Body.Items)
	}
}

// TestMeList_InvalidOwnerKindRejected checks that the explicit
// (ownerKind, ownerUuid) pair validates kind even when the principal is
// owned — bogus enum values must 400, not silently fall through.
func TestMeList_InvalidOwnerKindRejected(t *testing.T) {
	repo := newStubSubRepo()
	tp := &stubTenantProvider{}
	h := newHandlerForList(repo, tp)

	ctx := testkit.NewIdentity("user-A", "a@example.com", "user").ContextFor(context.Background(), "-")
	_, err := h.MeList(ctx, &MeListSubscriptionsRequest{
		OwnerKind: "bogus",
		OwnerUUID: "user-A",
	})
	if err == nil {
		t.Fatal("expected 400 for invalid ownerKind, got nil")
	}
	se, ok := err.(huma.StatusError)
	if !ok || se.GetStatus() != 400 {
		t.Fatalf("status: %v %T", err, err)
	}
	if !strings.Contains(strings.ToLower(se.Error()), "ownerkind") {
		t.Fatalf("error message: %s", se.Error())
	}
}

// newHandlerForListWithFlag wires the same surface as newHandlerForList but
// flips the Unified Client Aggregate Phase 2 lazy-tenant feature flag.
func newHandlerForListWithFlag(repo repository.SubscriptionRepository, tp iface.TenantProvider, lazy bool) *SubscriptionHandler {
	subs := services.NewSubscriptionService(repo, stubSvcRepoMin{}, nil, nil, tp, nopLogger())
	return NewSubscriptionHandler(subs, nil, nil, nil, tp, lazy)
}

// TestMeList_LazyTenantFlagOffSkipsEnsure asserts that the default
// (flag-off) behavior does not call EnsureTenantForUser — the path is still
// reachable but inert until Phase 3 flips the flag.
func TestMeList_LazyTenantFlagOffSkipsEnsure(t *testing.T) {
	repo := newStubSubRepo(userSub("u-1", "user-A", models.SubActive))
	tp := &stubTenantProvider{
		memberships:    map[string][]iface.TenantMembership{"user-A": nil},
		personalByUser: map[string]*iface.Tenant{"user-A": {UUID: "tenant-personal-A"}},
	}
	h := newHandlerForListWithFlag(repo, tp, false)

	ctx := testkit.NewIdentity("user-A", "a@example.com", "user").ContextFor(context.Background(), "-")
	if _, err := h.MeList(ctx, &MeListSubscriptionsRequest{}); err != nil {
		t.Fatalf("MeList: %v", err)
	}
	if len(tp.ensureCalls) != 0 {
		t.Fatalf("EnsureTenantForUser must not be called when flag is off, got %v", tp.ensureCalls)
	}
}

// TestMeList_LazyTenantFlagOnAddsPersonalTenant asserts that with the flag
// on the personal tenant is provisioned and its subscriptions surface in
// the caller's owner set alongside the legacy user-owned rows.
func TestMeList_LazyTenantFlagOnAddsPersonalTenant(t *testing.T) {
	repo := newStubSubRepo(
		userSub("u-1", "user-A", models.SubActive),
		tenantSub("p-1", "tenant-personal-A", models.SubActive),
		tenantSub("x-1", "tenant-other", models.SubActive), // not owned, must not surface
	)
	tp := &stubTenantProvider{
		memberships:    map[string][]iface.TenantMembership{"user-A": nil},
		personalByUser: map[string]*iface.Tenant{"user-A": {UUID: "tenant-personal-A"}},
	}
	h := newHandlerForListWithFlag(repo, tp, true)

	ctx := testkit.NewIdentity("user-A", "a@example.com", "user").ContextFor(context.Background(), "-")
	resp, err := h.MeList(ctx, &MeListSubscriptionsRequest{})
	if err != nil {
		t.Fatalf("MeList: %v", err)
	}
	if len(tp.ensureCalls) != 1 || tp.ensureCalls[0] != "user-A" {
		t.Fatalf("EnsureTenantForUser calls: %v", tp.ensureCalls)
	}
	if resp.Body.Total != 2 {
		t.Fatalf("expected user-owned + personal-tenant rows (2), got %d: %+v", resp.Body.Total, resp.Body.Items)
	}
	got := map[string]bool{}
	for _, s := range resp.Body.Items {
		got[s.UUID] = true
	}
	if !got["u-1"] || !got["p-1"] || got["x-1"] {
		t.Fatalf("unexpected fan-out: %+v", got)
	}
}

// TestMeList_LazyTenantFlagOnSurfacesEnsureError asserts that a failing
// EnsureTenantForUser propagates as the request error rather than silently
// scoping the user's data away.
func TestMeList_LazyTenantFlagOnSurfacesEnsureError(t *testing.T) {
	repo := newStubSubRepo(userSub("u-1", "user-A", models.SubActive))
	tp := &erroringEnsureTenantProvider{stubTenantProvider: stubTenantProvider{
		memberships: map[string][]iface.TenantMembership{"user-A": nil},
	}, ensureErr: errors.New("mongo down")}
	h := newHandlerForListWithFlag(repo, tp, true)

	ctx := testkit.NewIdentity("user-A", "a@example.com", "user").ContextFor(context.Background(), "-")
	_, err := h.MeList(ctx, &MeListSubscriptionsRequest{})
	if err == nil {
		t.Fatal("expected EnsureTenantForUser failure to surface")
	}
}

// erroringEnsureTenantProvider overrides EnsureTenantForUser to return an
// error — used to assert the handler does not silently swallow provisioning
// failures under the Phase 2 flag.
type erroringEnsureTenantProvider struct {
	stubTenantProvider
	ensureErr error
}

func (e *erroringEnsureTenantProvider) EnsureTenantForUser(context.Context, string) (*iface.Tenant, error) {
	return nil, e.ensureErr
}
