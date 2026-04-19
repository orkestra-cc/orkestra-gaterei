package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/orkestra/backend/internal/core/tenant/models"
	"github.com/orkestra/backend/internal/core/tenant/repository"
	"github.com/orkestra/backend/internal/shared/iface"
	"go.mongodb.org/mongo-driver/bson"
)

// GrantCapability creates an active entitlement for the tenant on the
// capability. If an active entitlement already exists, it is revoked first so
// the "at most one active row per (tenant, capability)" invariant holds. The
// replacement pattern lets subscription upgrades/downgrades land as a pair of
// (revoke old, insert new) rows — the history stays auditable.
//
// Signature matches iface.TenantProvider.GrantCapability so any module
// holding that interface handle can grant entitlements without importing
// the concrete tenant service.
func (s *Service) GrantCapability(ctx context.Context, in iface.GrantCapabilityInput) error {
	if in.TenantUUID == "" {
		return errors.New("tenant: GrantCapability requires TenantUUID")
	}
	if in.CapabilityID == "" {
		return errors.New("tenant: GrantCapability requires CapabilityID")
	}
	source := models.EntitlementSource(in.Source)
	if !source.Valid() {
		return fmt.Errorf("tenant: GrantCapability invalid source %q", in.Source)
	}
	if source == models.EntitlementSourceTrial && in.ExpiresAt == nil {
		return errors.New("tenant: trial entitlements must set ExpiresAt")
	}

	// Confirm the tenant exists so stale grants don't silently accumulate.
	if _, err := s.repo.GetTenantByUUID(ctx, in.TenantUUID); err != nil {
		return fmt.Errorf("tenant: GrantCapability: %w", err)
	}

	now := time.Now()
	// Revoke any existing active row for the same (tenant, capability).
	// Ignored if none exists (idempotent for first-time grants).
	if err := s.repo.RevokeActiveEntitlement(ctx, in.TenantUUID, in.CapabilityID, now); err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("tenant: GrantCapability: revoke existing: %w", err)
	}

	ent := &models.Entitlement{
		UUID:         uuid.NewString(),
		TenantUUID:   in.TenantUUID,
		CapabilityID: in.CapabilityID,
		Source:       source,
		SourceRef:    in.SourceRef,
		GrantedBy:    in.GrantedBy,
		GrantedAt:    now,
		ExpiresAt:    in.ExpiresAt,
		Metadata:     in.Metadata,
	}
	if err := s.repo.CreateEntitlement(ctx, ent); err != nil {
		return fmt.Errorf("tenant: GrantCapability: insert: %w", err)
	}
	return nil
}

// RevokeCapability marks the active entitlement for the (tenant, capability)
// pair as revoked. Returns nil even if no active row exists (idempotent from
// the caller's point of view — e.g. a double webhook delivery).
func (s *Service) RevokeCapability(ctx context.Context, tenantUUID, capabilityID string) error {
	if tenantUUID == "" || capabilityID == "" {
		return errors.New("tenant: RevokeCapability requires TenantUUID and CapabilityID")
	}
	err := s.repo.RevokeActiveEntitlement(ctx, tenantUUID, capabilityID, time.Now())
	if errors.Is(err, repository.ErrNotFound) {
		return nil
	}
	return err
}

// HasCapability reports whether the tenant currently has an active
// entitlement to the given capability. Implements the
// iface.TenantProvider.HasCapability contract.
//
// Internal (operator-side) tenants short-circuit to true: capabilities are
// the monetization seam for external client tenants, and internal tenants
// host the platform — they don't subscribe. This keeps operator tooling
// and dev workflows working without minting entitlement rows for every
// registered capability on internal tenant creation. External tenants
// always consult the projection.
func (s *Service) HasCapability(ctx context.Context, tenantUUID, capabilityID string) (bool, error) {
	if tenantUUID == "" || capabilityID == "" {
		return false, nil
	}
	if t, err := s.repo.GetTenantByUUID(ctx, tenantUUID); err == nil && t.Kind == models.TenantKindInternal {
		return true, nil
	}
	return s.repo.HasActiveEntitlement(ctx, tenantUUID, capabilityID)
}

// ListEntitlements returns the active entitlements held by a tenant.
func (s *Service) ListEntitlements(ctx context.Context, tenantUUID string) ([]models.Entitlement, error) {
	return s.repo.ListActiveByTenant(ctx, tenantUUID)
}

// ListCapabilityIDs returns the deduplicated capability IDs the tenant
// currently has active entitlements for, in deterministic (sorted) order.
// Thin projection over ListEntitlements so Cedar's principal builder does
// not reason about the entitlement metadata. Implements the
// iface.TenantProvider.ListCapabilityIDs contract.
func (s *Service) ListCapabilityIDs(ctx context.Context, tenantUUID string) ([]string, error) {
	if tenantUUID == "" {
		return nil, nil
	}
	ents, err := s.repo.ListActiveByTenant(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	if len(ents) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(ents))
	ids := make([]string, 0, len(ents))
	for _, e := range ents {
		if e.CapabilityID == "" {
			continue
		}
		if _, dup := seen[e.CapabilityID]; dup {
			continue
		}
		seen[e.CapabilityID] = struct{}{}
		ids = append(ids, e.CapabilityID)
	}
	sort.Strings(ids)
	return ids, nil
}

// Capability source constants mirror iface.GrantCapabilityInput.Source
// values expected by GrantCapability. Kept in this file so internal callers
// can reference them without going through iface.
const (
	CapabilitySourceSubscription = string(models.EntitlementSourceSubscription)
	CapabilitySourceGrant        = string(models.EntitlementSourceGrant)
	CapabilitySourceTrial        = string(models.EntitlementSourceTrial)
)

// ActivateTenant flips a tenant's lifecycle state to `active`. Thin wrapper
// over MarkTenantActive so it can satisfy iface.TenantProvider without the
// caller reaching into the concrete tenant service. Idempotent at the repo
// layer — re-activating an already-active tenant is a no-op write.
func (s *Service) ActivateTenant(ctx context.Context, tenantUUID string) error {
	if tenantUUID == "" {
		return errors.New("tenant: ActivateTenant requires tenantUUID")
	}
	return s.MarkTenantActive(ctx, tenantUUID)
}

// ProvisionExternalTenant is the onboarding entry point for anonymous
// self-service signup. Delegates to CreateExternalTenant for the
// Tier-2 bookkeeping (status=provisioning, signupChannel=self_serve,
// owner membership) and returns the DTO shape so callers crossing the
// module boundary don't import models.Tenant. Implements
// iface.TenantProvider.ProvisionExternalTenant.
func (s *Service) ProvisionExternalTenant(ctx context.Context, ownerUserUUID string, in iface.OnboardingTenantInput) (*iface.Tenant, error) {
	if ownerUserUUID == "" {
		return nil, errors.New("tenant: ProvisionExternalTenant requires ownerUserUUID")
	}
	if in.Name == "" {
		return nil, errors.New("tenant: ProvisionExternalTenant requires Name")
	}
	t, err := s.CreateExternalTenant(ctx, ownerUserUUID, in.Name, in.Slug, models.SignupChannelSelfServe, nil)
	if err != nil {
		return nil, fmt.Errorf("tenant: ProvisionExternalTenant: %w", err)
	}
	if in.Plan != "" && t.Plan != in.Plan {
		if err := s.repo.UpdateTenant(ctx, t.UUID, bson.M{"plan": in.Plan}); err != nil {
			return nil, fmt.Errorf("tenant: ProvisionExternalTenant: set plan: %w", err)
		}
		t.Plan = in.Plan
	}
	kind := string(t.Kind)
	if kind == "" {
		kind = iface.TenantKindExternal
	}
	status := string(t.Status)
	if status == "" {
		status = iface.TenantStatusProvisioning
	}
	return &iface.Tenant{
		UUID:   t.UUID,
		Kind:   kind,
		Status: status,
		Name:   t.Name,
		Slug:   t.Slug,
		Plan:   t.Plan,
	}, nil
}
