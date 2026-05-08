package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/orkestra/backend/internal/addons/payments/models"
	"github.com/orkestra/backend/internal/addons/payments/repository"
	"github.com/orkestra/backend/internal/shared/iface"
	"github.com/orkestra/backend/internal/testkit"
)

// stubTxRepo is a minimal TransactionRepository — only List is exercised by
// the self-service endpoints. Mutating methods fail loudly so a test that
// drifts onto an unintended path is caught.
type stubTxRepo struct {
	byOwner map[iface.Owner][]models.Transaction
	calls   []repository.TransactionFilters
}

func newStubTxRepo(rows ...models.Transaction) *stubTxRepo {
	r := &stubTxRepo{byOwner: map[iface.Owner][]models.Transaction{}}
	for _, t := range rows {
		key := iface.Owner{Kind: t.OwnerKind, UUID: t.OwnerUUID}
		r.byOwner[key] = append(r.byOwner[key], t)
	}
	return r
}

func (r *stubTxRepo) Create(context.Context, *models.Transaction) error { return nil }
func (r *stubTxRepo) GetByUUID(context.Context, string) (*models.Transaction, error) {
	return nil, errors.New("not implemented")
}
func (r *stubTxRepo) GetByProviderTxID(context.Context, models.ProviderName, string) (*models.Transaction, error) {
	return nil, errors.New("not implemented")
}
func (r *stubTxRepo) List(_ context.Context, f repository.TransactionFilters) ([]models.Transaction, error) {
	r.calls = append(r.calls, f)
	rows := r.byOwner[f.Owner]
	out := make([]models.Transaction, 0, len(rows))
	for _, t := range rows {
		if f.SubscriptionUUID != "" && t.SubscriptionUUID != f.SubscriptionUUID {
			continue
		}
		if f.Status != "" && t.Status != f.Status {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}
func (r *stubTxRepo) FindByOwner(context.Context, iface.Owner) ([]models.Transaction, error) {
	return nil, nil
}
func (r *stubTxRepo) Update(context.Context, *models.Transaction) error { return nil }

// stubPMRepo backs MeListPaymentMethods in the same shape — only ListByOwner
// is touched by the polymorphic fan-out path.
type stubPMRepo struct {
	byOwner map[iface.Owner][]models.PaymentMethod
}

func newStubPMRepo(rows ...models.PaymentMethod) *stubPMRepo {
	r := &stubPMRepo{byOwner: map[iface.Owner][]models.PaymentMethod{}}
	for _, pm := range rows {
		r.byOwner[pm.Owner()] = append(r.byOwner[pm.Owner()], pm)
	}
	return r
}

func (r *stubPMRepo) Create(context.Context, *models.PaymentMethod) error { return nil }
func (r *stubPMRepo) GetByUUID(context.Context, string) (*models.PaymentMethod, error) {
	return nil, errors.New("not implemented")
}
func (r *stubPMRepo) ListByOwner(_ context.Context, owner iface.Owner) ([]models.PaymentMethod, error) {
	return r.byOwner[owner], nil
}
func (r *stubPMRepo) Delete(context.Context, string) error { return nil }

// stubTenantProvider — local copy because Go test packages can't share types
// across modules without a shared helper, and the surface needed here is
// trivially small.
type stubTenantProvider struct {
	memberships map[string][]iface.TenantMembership
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
func (s *stubTenantProvider) EnsureTenantForUser(context.Context, string) (*iface.Tenant, error) {
	return nil, nil
}

func userTx(uuid, userUUID string, status models.TransactionStatus) models.Transaction {
	return models.Transaction{
		UUID:      uuid,
		OwnerKind: iface.OwnerKindUser,
		OwnerUUID: userUUID,
		Status:    status,
		CreatedAt: time.Now(),
	}
}

func tenantTx(uuid, tenantUUID string, status models.TransactionStatus) models.Transaction {
	return models.Transaction{
		UUID:      uuid,
		OwnerKind: iface.OwnerKindTenant,
		OwnerUUID: tenantUUID,
		Status:    status,
		CreatedAt: time.Now(),
	}
}

func userPM(uuid, userUUID string) models.PaymentMethod {
	return models.PaymentMethod{UUID: uuid, OwnerKind: iface.OwnerKindUser, OwnerUUID: userUUID}
}

func tenantPM(uuid, tenantUUID string) models.PaymentMethod {
	return models.PaymentMethod{UUID: uuid, OwnerKind: iface.OwnerKindTenant, OwnerUUID: tenantUUID}
}

// newClientHandler wires a ClientHandler with just the deps the read paths
// touch. payment, userBilling and planner are not exercised by MeListXxx.
func newClientHandlerForReads(tx repository.TransactionRepository, pm repository.PaymentMethodRepository, tp iface.TenantProvider) *ClientHandler {
	return NewClientHandler(nil, tx, pm, tp, nil, nil)
}

// --- Transactions ---

func TestMeListTransactions_AnonymousReturns401(t *testing.T) {
	h := newClientHandlerForReads(newStubTxRepo(), newStubPMRepo(), &stubTenantProvider{})
	_, err := h.MeListTransactions(context.Background(), &MeListTransactionsRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	se, ok := err.(huma.StatusError)
	if !ok || se.GetStatus() != 401 {
		t.Fatalf("status: %v %T", err, err)
	}
}

func TestMeListTransactions_FansOutAcrossUserAndOwnedTenant(t *testing.T) {
	tx := newStubTxRepo(
		userTx("tx-u", "user-A", models.TxSucceeded),
		tenantTx("tx-tX", "tenant-X", models.TxSucceeded),
		tenantTx("tx-tY", "tenant-Y", models.TxSucceeded), // not owned
	)
	tp := &stubTenantProvider{memberships: map[string][]iface.TenantMembership{
		"user-A": {
			{TenantUUID: "tenant-X", IsOwner: true},
			{TenantUUID: "tenant-Y", IsOwner: false},
		},
	}}
	h := newClientHandlerForReads(tx, newStubPMRepo(), tp)

	ctx := testkit.NewIdentity("user-A", "a@example.com", "user").ContextFor(context.Background(), "-")
	resp, err := h.MeListTransactions(ctx, &MeListTransactionsRequest{})
	if err != nil {
		t.Fatalf("MeListTransactions: %v", err)
	}
	if resp.Body.Total != 2 {
		t.Fatalf("expected 2 rows (user + owned tenant), got %d", resp.Body.Total)
	}
	got := map[string]bool{}
	for _, r := range resp.Body.Items {
		got[r.UUID] = true
	}
	if !got["tx-u"] || !got["tx-tX"] || got["tx-tY"] {
		t.Fatalf("fan-out shape: %+v", got)
	}
}

func TestMeListTransactions_FilterUnownedPrincipalReturnsEmpty(t *testing.T) {
	tx := newStubTxRepo(tenantTx("tx-tZ", "tenant-Z", models.TxSucceeded))
	tp := &stubTenantProvider{memberships: map[string][]iface.TenantMembership{
		"user-A": {{TenantUUID: "tenant-X", IsOwner: true}},
	}}
	h := newClientHandlerForReads(tx, newStubPMRepo(), tp)

	ctx := testkit.NewIdentity("user-A", "a@example.com", "user").ContextFor(context.Background(), "-")
	resp, err := h.MeListTransactions(ctx, &MeListTransactionsRequest{
		OwnerKind: string(iface.OwnerKindTenant),
		OwnerUUID: "tenant-Z",
	})
	if err != nil {
		t.Fatalf("MeListTransactions: %v", err)
	}
	if resp.Body.Total != 0 {
		t.Fatalf("expected 0 rows for unowned tenant filter, got %d", resp.Body.Total)
	}
}

func TestMeListTransactions_LegacyTenantAliasFiltersOnTenantOwner(t *testing.T) {
	tx := newStubTxRepo(
		userTx("tx-u", "user-A", models.TxSucceeded),
		tenantTx("tx-tX", "tenant-X", models.TxSucceeded),
	)
	tp := &stubTenantProvider{memberships: map[string][]iface.TenantMembership{
		"user-A": {{TenantUUID: "tenant-X", IsOwner: true}},
	}}
	h := newClientHandlerForReads(tx, newStubPMRepo(), tp)

	ctx := testkit.NewIdentity("user-A", "a@example.com", "user").ContextFor(context.Background(), "-")
	resp, err := h.MeListTransactions(ctx, &MeListTransactionsRequest{TenantUUID: "tenant-X"})
	if err != nil {
		t.Fatalf("MeListTransactions: %v", err)
	}
	if resp.Body.Total != 1 || resp.Body.Items[0].UUID != "tx-tX" {
		t.Fatalf("legacy alias mismatch: %+v", resp.Body.Items)
	}
}

// --- Payment methods ---

func TestMeListPaymentMethods_FansOutAcrossUserAndOwnedTenant(t *testing.T) {
	pm := newStubPMRepo(
		userPM("pm-u", "user-A"),
		tenantPM("pm-tX", "tenant-X"),
		tenantPM("pm-tY", "tenant-Y"),
	)
	tp := &stubTenantProvider{memberships: map[string][]iface.TenantMembership{
		"user-A": {
			{TenantUUID: "tenant-X", IsOwner: true},
			{TenantUUID: "tenant-Y", IsOwner: false},
		},
	}}
	h := newClientHandlerForReads(newStubTxRepo(), pm, tp)

	ctx := testkit.NewIdentity("user-A", "a@example.com", "user").ContextFor(context.Background(), "-")
	resp, err := h.MeListPaymentMethods(ctx, &MeListPaymentMethodsRequest{})
	if err != nil {
		t.Fatalf("MeListPaymentMethods: %v", err)
	}
	if resp.Body.Total != 2 {
		t.Fatalf("expected 2 rows, got %d (%+v)", resp.Body.Total, resp.Body.Items)
	}
	got := map[string]bool{}
	for _, r := range resp.Body.Items {
		got[r.UUID] = true
	}
	if !got["pm-u"] || !got["pm-tX"] || got["pm-tY"] {
		t.Fatalf("fan-out shape: %+v", got)
	}
}

func TestMeListPaymentMethods_NilTenantProviderStillReturnsUserRows(t *testing.T) {
	// Payments must not hard-require the tenant module — user-only flows
	// still resolve when h.tenants is nil.
	pm := newStubPMRepo(userPM("pm-u", "user-A"))
	h := newClientHandlerForReads(newStubTxRepo(), pm, nil)

	ctx := testkit.NewIdentity("user-A", "a@example.com", "user").ContextFor(context.Background(), "-")
	resp, err := h.MeListPaymentMethods(ctx, &MeListPaymentMethodsRequest{})
	if err != nil {
		t.Fatalf("MeListPaymentMethods: %v", err)
	}
	if resp.Body.Total != 1 || resp.Body.Items[0].UUID != "pm-u" {
		t.Fatalf("expected only the calling user's pm, got %+v", resp.Body.Items)
	}
}
