package handlers

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/orkestra/backend/internal/shared/middleware"
)

// assertTenantOwnsClient returns a 404 when the caller's active tenant
// does not match the client's `orgUUID` (legacy field; retained during the
// Phase 3 migration where Client records will be folded into Tier-2
// Tenants). Clients with an empty `orgUUID` are treated as operator-managed
// (not tenant-bound) and are allowed. Returning 404 (not 403) avoids
// leaking existence of out-of-scope records.
func assertTenantOwnsClient(ctx context.Context, clientOrgUUID string) error {
	if clientOrgUUID == "" {
		return nil
	}
	tenantID, hasTenant := middleware.GetTenantID(ctx)
	if !hasTenant {
		return nil
	}
	if clientOrgUUID != tenantID {
		return huma.Error404NotFound("not found", nil)
	}
	return nil
}
