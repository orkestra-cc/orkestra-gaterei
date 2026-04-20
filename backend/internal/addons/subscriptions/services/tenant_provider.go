package services

import (
	"context"

	"github.com/orkestra/backend/internal/addons/subscriptions/repository"
	"github.com/orkestra/backend/internal/shared/iface"
)

// TenantSubscriptionAdapter implements iface.TenantSubscriptionProvider by
// translating the rich subscriptions_subscriptions documents into the flat
// DTOs the core/tenant aggregator endpoint returns. It is intentionally
// read-only — mutations stay on the dedicated Subscription handlers.
//
// The adapter holds only the repository because that is the sole dependency
// this read path needs; no Client/Service/Tier joins happen here.
type TenantSubscriptionAdapter struct {
	subs repository.SubscriptionRepository
}

// NewTenantSubscriptionAdapter constructs the adapter. Registered under
// module.ServiceTenantSubscriptionProvider by SubscriptionsModule.Init.
func NewTenantSubscriptionAdapter(subs repository.SubscriptionRepository) *TenantSubscriptionAdapter {
	return &TenantSubscriptionAdapter{subs: subs}
}

// ListByTenant returns every subscription bound to the tenant via the
// Phase 1 forward-looking TenantUUID. The repository layer already sorts
// by createdAt desc.
func (a *TenantSubscriptionAdapter) ListByTenant(ctx context.Context, tenantUUID string) ([]iface.TenantSubscription, error) {
	if tenantUUID == "" {
		return []iface.TenantSubscription{}, nil
	}
	rows, err := a.subs.FindByTenantUUID(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	out := make([]iface.TenantSubscription, 0, len(rows))
	for i := range rows {
		r := rows[i]
		out = append(out, iface.TenantSubscription{
			UUID:               r.UUID,
			TenantUUID:         r.TenantUUID,
			ClientUUID:         r.ClientUUID,
			ServiceUUID:        r.ServiceUUID,
			TierCode:           r.TierCode,
			Status:             string(r.Status),
			CurrentPeriodStart: r.CurrentPeriodStart,
			CurrentPeriodEnd:   r.CurrentPeriodEnd,
			NextBillingAt:      r.NextBillingAt,
			CreatedAt:          r.CreatedAt,
		})
	}
	return out, nil
}
