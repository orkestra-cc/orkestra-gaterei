package handlers

import (
	"context"
	"testing"
)

// TestAggregators_DegradeWhenRegistryNil verifies the Phase 2 aggregator
// endpoints return an empty list (not 500) when the handler has no registry
// wired. Matches the spec: "missing addon = empty list, not error".
//
// The handler is constructed with svc=nil which is fine because the
// aggregator paths never touch svc — they route through the registry.
func TestAggregators_DegradeWhenRegistryNil(t *testing.T) {
	h := New(nil, nil)

	subOut, err := h.listTenantSubscriptionsAdmin(context.Background(), &tenantIDPath{TenantID: "tenant-1"})
	if err != nil {
		t.Fatalf("subscriptions aggregator: unexpected error: %v", err)
	}
	if subOut == nil || len(subOut.Body.Subscriptions) != 0 {
		t.Fatalf("subscriptions aggregator: want empty slice, got %+v", subOut)
	}

	payOut, err := h.listTenantPaymentsAdmin(context.Background(), &tenantIDPath{TenantID: "tenant-1"})
	if err != nil {
		t.Fatalf("payments aggregator: unexpected error: %v", err)
	}
	if payOut == nil || len(payOut.Body.Payments) != 0 {
		t.Fatalf("payments aggregator: want empty slice, got %+v", payOut)
	}
}
