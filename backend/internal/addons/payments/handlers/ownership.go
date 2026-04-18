package handlers

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/orkestra/backend/internal/shared/iface"
	"github.com/orkestra/backend/internal/shared/middleware"
	"github.com/orkestra/backend/internal/shared/module"
)

// resolveOwnership returns the ClientOwnershipProvider currently registered
// by the subscriptions module, or nil if subscriptions is disabled. Lazily
// looked up on every request because modules can be hot-toggled.
func resolveOwnership(svcReg *module.ServiceRegistry) iface.ClientOwnershipProvider {
	if svcReg == nil {
		return nil
	}
	p, _ := module.GetTyped[iface.ClientOwnershipProvider](svcReg, module.ServiceClientOwnership)
	return p
}

// assertTenantOwnsClient enforces that the requesting user's active tenant
// matches the client's `orgUUID` (legacy field; will be replaced with
// TenantUUID in Phase 3). Degrades safely when:
//   - subscriptions (and therefore the provider) is disabled, or
//   - the client has no tenant binding (operator-managed clients, v1 default), or
//   - the request has no tenant context (global/service callers).
//
// Returns a 404 on mismatch so existence of out-of-scope records isn't leaked.
func assertTenantOwnsClient(ctx context.Context, svcReg *module.ServiceRegistry, clientUUID string) error {
	if clientUUID == "" {
		return nil
	}
	provider := resolveOwnership(svcReg)
	if provider == nil {
		return nil
	}
	clientOrgUUID, err := provider.GetClientOrgUUID(ctx, clientUUID)
	if err != nil {
		// Unknown client — treat as not-found for the caller.
		return nil
	}
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
