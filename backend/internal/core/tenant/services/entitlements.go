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

// GrantCapability creates an active entitlement for the owner on the
// capability. If an active entitlement already exists, it is revoked first
// so the "at most one active row per (owner, capability)" invariant holds.
// The replacement pattern lets subscription upgrades/downgrades land as a
// pair of (revoke old, insert new) rows — the history stays auditable.
//
// Signature matches iface.AccessProvider.GrantCapability so any module
// holding that interface handle can grant entitlements without importing
// the concrete tenant service.
func (s *Service) GrantCapability(ctx context.Context, in iface.GrantCapabilityInput) error {
	if in.Owner.IsZero() {
		return errors.New("tenant: GrantCapability requires Owner.Kind and Owner.UUID")
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

	// For tenant-owned grants, confirm the tenant exists so stale grants
	// don't silently accumulate. User-owned grants don't validate the
	// owner here — the user module is the source of truth and a missing
	// user would surface at consumption time.
	if in.Owner.Kind == iface.OwnerKindTenant {
		if _, err := s.repo.GetTenantByUUID(ctx, in.Owner.UUID); err != nil {
			return fmt.Errorf("tenant: GrantCapability: %w", err)
		}
	}

	now := time.Now()
	// Revoke any existing active row for the same (owner, capability).
	// Ignored if none exists (idempotent for first-time grants).
	if err := s.repo.RevokeActiveEntitlement(ctx, in.Owner, in.CapabilityID, now); err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("tenant: GrantCapability: revoke existing: %w", err)
	}

	ent := &models.Entitlement{
		UUID:         uuid.NewString(),
		OwnerKind:    in.Owner.Kind,
		OwnerUUID:    in.Owner.UUID,
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

// RevokeCapability marks the active entitlement for the (owner, capability)
// pair as revoked. Returns nil even if no active row exists (idempotent from
// the caller's point of view — e.g. a double webhook delivery).
func (s *Service) RevokeCapability(ctx context.Context, owner iface.Owner, capabilityID string) error {
	if owner.IsZero() || capabilityID == "" {
		return errors.New("tenant: RevokeCapability requires Owner and CapabilityID")
	}
	err := s.repo.RevokeActiveEntitlement(ctx, owner, capabilityID, time.Now())
	if errors.Is(err, repository.ErrNotFound) {
		return nil
	}
	return err
}

// HasCapability reports whether the owner currently has an active
// entitlement to the given capability. Implements
// iface.AccessProvider.HasCapability.
//
// Internal (operator-side) tenants short-circuit to true: capabilities are
// the monetization seam for external clients, and internal tenants host
// the platform — they don't subscribe. Per-user owners always consult the
// projection (no operator user is granted capabilities by tier).
func (s *Service) HasCapability(ctx context.Context, owner iface.Owner, capabilityID string) (bool, error) {
	if owner.IsZero() || capabilityID == "" {
		return false, nil
	}
	if owner.Kind == iface.OwnerKindTenant {
		if t, err := s.repo.GetTenantByUUID(ctx, owner.UUID); err == nil && t.Kind == models.TenantKindInternal {
			return true, nil
		}
	}
	return s.repo.HasActiveEntitlement(ctx, owner, capabilityID)
}

// ListEntitlements returns the active entitlements held by an owner.
func (s *Service) ListEntitlements(ctx context.Context, owner iface.Owner) ([]models.Entitlement, error) {
	return s.repo.ListActiveByOwner(ctx, owner)
}

// ListCapabilityIDs returns the deduplicated capability IDs the owner
// currently has active entitlements for, in deterministic (sorted) order.
// Thin projection over ListEntitlements so Cedar's principal builder does
// not reason about the entitlement metadata. Implements
// iface.AccessProvider.ListCapabilityIDs.
func (s *Service) ListCapabilityIDs(ctx context.Context, owner iface.Owner) ([]string, error) {
	if owner.IsZero() {
		return nil, nil
	}
	ents, err := s.repo.ListActiveByOwner(ctx, owner)
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

// SetTenantStripeCustomerID persists the Stripe customer identifier on the
// tenant row. Called by the subscriptions renewal service after Stripe mints a
// customer on the first charge for an external tenant. Idempotent — applying
// the same value is a no-op write.
func (s *Service) SetTenantStripeCustomerID(ctx context.Context, tenantUUID, stripeCustomerID string) error {
	if tenantUUID == "" {
		return errors.New("tenant: SetTenantStripeCustomerID requires tenantUUID")
	}
	return s.repo.UpdateTenant(ctx, tenantUUID, bson.M{"stripeCustomerID": stripeCustomerID})
}
