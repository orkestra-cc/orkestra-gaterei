package migrations

import (
	"context"
	"errors"
	"testing"
	"time"

	pmtmodels "github.com/orkestra/backend/internal/addons/payments/models"
	"github.com/orkestra/backend/internal/addons/subscriptions/models"
	"github.com/orkestra/backend/internal/addons/subscriptions/repository"
)

// stubSubRepo is the minimal SubscriptionRepository needed by
// resolveTransactionTenant. Only GetByUUID is exercised; the other methods
// return zero values so the compile target (iface conformance) is satisfied.
type stubSubRepo struct {
	byUUID map[string]*models.Subscription
	err    error
}

func (s *stubSubRepo) Create(context.Context, *models.Subscription) error { return nil }
func (s *stubSubRepo) GetByUUID(_ context.Context, uuid string) (*models.Subscription, error) {
	if s.err != nil {
		return nil, s.err
	}
	if v, ok := s.byUUID[uuid]; ok {
		return v, nil
	}
	return nil, errors.New("not found")
}
func (s *stubSubRepo) List(context.Context, repository.SubscriptionFilters) ([]models.Subscription, error) {
	return nil, nil
}
func (s *stubSubRepo) Update(context.Context, *models.Subscription) error { return nil }
func (s *stubSubRepo) FindDue(context.Context, time.Time) ([]models.Subscription, error) {
	return nil, nil
}
func (s *stubSubRepo) FindByTenantUUID(context.Context, string) ([]models.Subscription, error) {
	return nil, nil
}
func (s *stubSubRepo) FindWithoutTenantUUID(context.Context, int64) ([]models.Subscription, error) {
	return nil, nil
}
func (s *stubSubRepo) SetTenantUUID(context.Context, string, string) error { return nil }
func (s *stubSubRepo) UpdateStatus(context.Context, string, models.SubStatus) error {
	return nil
}
func (s *stubSubRepo) Delete(context.Context, string) error { return nil }

// TestResolveTransactionTenant_PrefersSubscription walks the four branches
// resolveTransactionTenant exercises: subscription hit with tenantUUID,
// subscription hit falling back to clientUUID cache, transaction-only
// clientUUID cache hit, and the no-information tail.
func TestResolveTransactionTenant_PrefersSubscription(t *testing.T) {
	_ = stubSubRepo{} // compile-time only; resolveTransactionTenant uses the interface
	tenantByClient := map[string]string{"client-1": "tenant-from-client-1"}

	cases := []struct {
		name         string
		tx           pmtmodels.Transaction
		subs         map[string]*models.Subscription
		wantTenantID string
	}{
		{
			name: "subscription has tenantUUID",
			tx:   pmtmodels.Transaction{SubscriptionUUID: "sub-1"},
			subs: map[string]*models.Subscription{
				"sub-1": {UUID: "sub-1", TenantUUID: "tenant-direct"},
			},
			wantTenantID: "tenant-direct",
		},
		{
			name: "subscription lacks tenantUUID, falls back to client cache",
			tx:   pmtmodels.Transaction{SubscriptionUUID: "sub-2"},
			subs: map[string]*models.Subscription{
				"sub-2": {UUID: "sub-2", ClientUUID: "client-1"},
			},
			wantTenantID: "tenant-from-client-1",
		},
		{
			name:         "no subscription, tx has clientUUID, client cache hit",
			tx:           pmtmodels.Transaction{ClientUUID: "client-1"},
			subs:         map[string]*models.Subscription{},
			wantTenantID: "tenant-from-client-1",
		},
		{
			name:         "no information at all",
			tx:           pmtmodels.Transaction{},
			subs:         map[string]*models.Subscription{},
			wantTenantID: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := BackfillDeps{Subs: &stubSubRepo{byUUID: tc.subs}}
			got, err := resolveTransactionTenant(context.Background(), d, &tc.tx, map[string]string{}, tenantByClient)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantTenantID {
				t.Fatalf("tenantUUID = %q, want %q", got, tc.wantTenantID)
			}
		})
	}
}
