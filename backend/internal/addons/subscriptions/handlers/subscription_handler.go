package handlers

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/orkestra/backend/internal/addons/subscriptions/models"
	"github.com/orkestra/backend/internal/addons/subscriptions/repository"
	"github.com/orkestra/backend/internal/addons/subscriptions/services"
	"github.com/orkestra/backend/internal/shared/iface"
	"github.com/orkestra/backend/internal/shared/middleware"
)

type SubscriptionHandler struct {
	subs     *services.SubscriptionService
	renewal  *services.RenewalService
	invoices repository.InvoiceRepository
	activity *services.ActivityService
	// tenants powers the self-subscribe ownership check. nil in setups that
	// run subscriptions without the tenant module — self-subscribe returns
	// 503 in that case (the route refuses to guess ownership from thin air).
	tenants iface.TenantProvider
}

func NewSubscriptionHandler(
	subs *services.SubscriptionService,
	renewal *services.RenewalService,
	invoices repository.InvoiceRepository,
	activity *services.ActivityService,
	tenants iface.TenantProvider,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		subs:     subs,
		renewal:  renewal,
		invoices: invoices,
		activity: activity,
		tenants:  tenants,
	}
}

// CreateSubscriptionInput is the operator-admin payload. ADR-0001 Phase 1
// removed the legacy SubscriptionClient indirection — subscriptions point
// directly at an external tenant.
type CreateSubscriptionInput struct {
	TenantUUID  string `json:"tenantUUID" doc:"UUID of the external tenant the subscription is for"`
	ServiceUUID string `json:"serviceUUID"`
	TierCode    string `json:"tierCode"`
}

type CreateSubscriptionRequest struct {
	Body CreateSubscriptionInput
}
type SubscriptionResponse struct {
	Body models.Subscription
}
type GetSubscriptionRequest struct {
	ID string `path:"id"`
}
type ListSubscriptionsRequest struct {
	TenantUUID  string `query:"tenantUUID"`
	ServiceUUID string `query:"serviceUUID"`
	Status      string `query:"status" enum:"active,past_due,suspended,cancelled,expired"`
}
type ListSubscriptionsResponse struct {
	Body struct {
		Items []models.Subscription `json:"items"`
		Total int                   `json:"total"`
	}
}
type CancelSubscriptionRequest struct {
	ID   string `path:"id"`
	Body struct {
		AtPeriodEnd bool `json:"atPeriodEnd"`
	}
}
type ReactivateSubscriptionRequest struct {
	ID string `path:"id"`
}
type RetryChargeRequest struct {
	ID string `path:"id"`
}

func (h *SubscriptionHandler) Create(ctx context.Context, in *CreateSubscriptionRequest) (*SubscriptionResponse, error) {
	if in.Body.TenantUUID == "" {
		return nil, huma.Error400BadRequest("tenantUUID is required")
	}
	if in.Body.ServiceUUID == "" || in.Body.TierCode == "" {
		return nil, huma.Error400BadRequest("serviceUUID and tierCode are required")
	}
	sub, err := h.subs.Create(ctx, in.Body.TenantUUID, in.Body.ServiceUUID, in.Body.TierCode, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	return &SubscriptionResponse{Body: *sub}, nil
}

func (h *SubscriptionHandler) Get(ctx context.Context, in *GetSubscriptionRequest) (*SubscriptionResponse, error) {
	sub, err := h.subs.Get(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if err := h.guardSubscriptionOwnership(ctx, sub); err != nil {
		return nil, err
	}
	return &SubscriptionResponse{Body: *sub}, nil
}

func (h *SubscriptionHandler) List(ctx context.Context, in *ListSubscriptionsRequest) (*ListSubscriptionsResponse, error) {
	items, err := h.subs.List(ctx, repository.SubscriptionFilters{
		TenantUUID:  in.TenantUUID,
		ServiceUUID: in.ServiceUUID,
		Status:      models.SubStatus(in.Status),
	})
	if err != nil {
		return nil, err
	}
	resp := &ListSubscriptionsResponse{}
	resp.Body.Items = items
	resp.Body.Total = len(items)
	return resp, nil
}

func (h *SubscriptionHandler) Cancel(ctx context.Context, in *CancelSubscriptionRequest) (*SubscriptionResponse, error) {
	existing, err := h.subs.Get(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if err := h.guardSubscriptionOwnership(ctx, existing); err != nil {
		return nil, err
	}
	sub, err := h.subs.Cancel(ctx, in.ID, in.Body.AtPeriodEnd, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	return &SubscriptionResponse{Body: *sub}, nil
}

func (h *SubscriptionHandler) Reactivate(ctx context.Context, in *ReactivateSubscriptionRequest) (*SubscriptionResponse, error) {
	existing, err := h.subs.Get(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if err := h.guardSubscriptionOwnership(ctx, existing); err != nil {
		return nil, err
	}
	sub, err := h.subs.Reactivate(ctx, in.ID, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	return &SubscriptionResponse{Body: *sub}, nil
}

func (h *SubscriptionHandler) RetryCharge(ctx context.Context, in *RetryChargeRequest) (*SubscriptionResponse, error) {
	existing, err := h.subs.Get(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if err := h.guardSubscriptionOwnership(ctx, existing); err != nil {
		return nil, err
	}
	sub, err := h.renewal.RetryNow(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	return &SubscriptionResponse{Body: *sub}, nil
}

type ListInvoicesRequest struct {
	ID string `path:"id"`
}
type ListInvoicesResponse struct {
	Body struct {
		Items []models.SubscriptionInvoice `json:"items"`
		Total int                          `json:"total"`
	}
}

func (h *SubscriptionHandler) ListInvoices(ctx context.Context, in *ListInvoicesRequest) (*ListInvoicesResponse, error) {
	sub, err := h.subs.Get(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if err := h.guardSubscriptionOwnership(ctx, sub); err != nil {
		return nil, err
	}
	items, err := h.invoices.List(ctx, repository.InvoiceFilters{SubscriptionUUID: in.ID})
	if err != nil {
		return nil, err
	}
	resp := &ListInvoicesResponse{}
	resp.Body.Items = items
	resp.Body.Total = len(items)
	return resp, nil
}

type ListActivityRequest struct {
	ID    string `path:"id"`
	Limit int64  `query:"limit" default:"100" maximum:"500"`
}
type ListActivityResponse struct {
	Body struct {
		Items []models.ActivityLog `json:"items"`
		Total int                  `json:"total"`
	}
}

func (h *SubscriptionHandler) ListActivity(ctx context.Context, in *ListActivityRequest) (*ListActivityResponse, error) {
	sub, err := h.subs.Get(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if err := h.guardSubscriptionOwnership(ctx, sub); err != nil {
		return nil, err
	}
	items, err := h.activity.List(ctx, in.ID, in.Limit)
	if err != nil {
		return nil, err
	}
	resp := &ListActivityResponse{}
	resp.Body.Items = items
	resp.Body.Total = len(items)
	return resp, nil
}

// SelfSubscribeInput is the wire-level payload for the self-service
// subscribe endpoint. Tenant is identified by UUID (typed by the caller
// who read it off their own memberships in the JWT) so the route can
// be used from global context without an X-Tenant-ID header, and the
// ownership check runs against the same provider the JWT builder does.
type SelfSubscribeInput struct {
	TenantUUID  string `json:"tenantUuid" doc:"UUID of the tenant the caller owns"`
	ServiceCode string `json:"serviceCode" doc:"Stable SKU code of a catalog service (e.g. 'pro')"`
	TierCode    string `json:"tierCode" doc:"Tier code within the service (e.g. 'monthly')"`
}

// SelfSubscribeRequest is the Huma wrapper. The body shape is concretely
// named so the OpenAPI schema registry keeps it distinct from the
// operator-facing CreateSubscriptionInput.
type SelfSubscribeRequest struct {
	Body SelfSubscribeInput
}

// SelfSubscribe creates a subscription for a tenant the caller owns.
// Anonymous self-service checkout (public Stripe checkout session) is
// deferred — today we require an authenticated caller, gated to owners
// only. The entitlement syncer grants capabilities on activation so the
// tenant can hit RequireCapability-gated addons immediately; the first
// payment attempt still happens on the next renewal tick.
func (h *SubscriptionHandler) SelfSubscribe(ctx context.Context, req *SelfSubscribeRequest) (*SubscriptionResponse, error) {
	if h.tenants == nil {
		return nil, huma.Error503ServiceUnavailable("tenant provider not configured")
	}
	userUUID, ok := middleware.GetUserUUID(ctx)
	if !ok || userUUID == "" {
		return nil, huma.Error401Unauthorized("authentication required")
	}
	if req.Body.TenantUUID == "" || req.Body.ServiceCode == "" || req.Body.TierCode == "" {
		return nil, huma.Error400BadRequest("tenantUuid, serviceCode and tierCode are required")
	}

	memberships, err := h.tenants.ListUserMemberships(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	ownsTenant := false
	for _, m := range memberships {
		if m.TenantUUID == req.Body.TenantUUID && m.IsOwner {
			ownsTenant = true
			break
		}
	}
	if !ownsTenant {
		return nil, huma.Error403Forbidden("caller is not owner of requested tenant")
	}

	sub, err := h.subs.CreateForTenant(ctx, req.Body.TenantUUID, req.Body.ServiceCode, req.Body.TierCode, userUUID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrSubscriptionServiceCode):
			return nil, huma.Error404NotFound("service not found")
		case errors.Is(err, services.ErrSubscriptionTierNotFound):
			return nil, huma.Error404NotFound("tier not found on service")
		default:
			return nil, err
		}
	}
	return &SubscriptionResponse{Body: *sub}, nil
}

// guardSubscriptionOwnership enforces that the request's active tenant
// matches the subscription's TenantUUID. Subscriptions without a tenant
// binding are treated as not-owned and visible only to platform admins
// (whose tokens already bypass the X-Tenant-ID match check via the
// system.tenants.admin permission). Returns 404 on mismatch so existence
// of out-of-scope records is not leaked.
func (h *SubscriptionHandler) guardSubscriptionOwnership(ctx context.Context, sub *models.Subscription) error {
	if sub == nil || sub.TenantUUID == "" {
		return nil
	}
	tenantID, hasTenant := middleware.GetTenantID(ctx)
	if !hasTenant {
		return nil
	}
	if sub.TenantUUID != tenantID {
		return huma.Error404NotFound("not found", nil)
	}
	return nil
}

// meOwnedTenants returns the set of TenantUUIDs the calling user owns,
// resolved by a fresh ListUserMemberships call against TenantProvider.
// Memberships are read live (not from JWT context) so a freshly granted
// or revoked ownership is reflected without waiting for token refresh.
//
// Returns 401 when the caller is anonymous, 503 when the tenant provider
// is not wired, and an empty set when the caller owns no tenants.
func (h *SubscriptionHandler) meOwnedTenants(ctx context.Context) (map[string]struct{}, error) {
	if h.tenants == nil {
		return nil, huma.Error503ServiceUnavailable("tenant provider not configured")
	}
	userUUID, ok := middleware.GetUserUUID(ctx)
	if !ok || userUUID == "" {
		return nil, huma.Error401Unauthorized("authentication required")
	}
	memberships, err := h.tenants.ListUserMemberships(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	owned := make(map[string]struct{}, len(memberships))
	for _, m := range memberships {
		if m.IsOwner {
			owned[m.TenantUUID] = struct{}{}
		}
	}
	return owned, nil
}

// meGuardSubscription resolves the subscription by UUID and returns it iff
// the caller owns its bound tenant. Returns 404 (not 403) on ownership
// mismatch so the existence of out-of-scope subscriptions does not leak
// to a fishing client.
func (h *SubscriptionHandler) meGuardSubscription(ctx context.Context, subscriptionUUID string) (*models.Subscription, error) {
	owned, err := h.meOwnedTenants(ctx)
	if err != nil {
		return nil, err
	}
	sub, err := h.subs.Get(ctx, subscriptionUUID)
	if err != nil {
		return nil, err
	}
	if _, ok := owned[sub.TenantUUID]; !ok {
		return nil, huma.Error404NotFound("not found")
	}
	return sub, nil
}

// --- Self-service handlers ---

// MeListSubscriptionsRequest filters the caller's subscriptions to one
// owned tenant when TenantUUID is non-empty; otherwise returns
// subscriptions across every tenant the caller owns.
type MeListSubscriptionsRequest struct {
	TenantUUID string `query:"tenantUuid" doc:"Optional — restrict to one owned tenant"`
	Status     string `query:"status" enum:"active,past_due,suspended,cancelled,expired"`
}

// MeList returns subscriptions across every tenant the caller owns. The
// repository is queried per-tenant and the results are merged in memory;
// for the demo MVP a Tier-2 user is rarely an owner of more than a handful
// of tenants, so a fan-out without an aggregate index is acceptable.
func (h *SubscriptionHandler) MeList(ctx context.Context, in *MeListSubscriptionsRequest) (*ListSubscriptionsResponse, error) {
	owned, err := h.meOwnedTenants(ctx)
	if err != nil {
		return nil, err
	}
	resp := &ListSubscriptionsResponse{}
	resp.Body.Items = []models.Subscription{}

	if in.TenantUUID != "" {
		if _, ok := owned[in.TenantUUID]; !ok {
			// Pretend the tenant does not exist rather than confirming
			// it does and refusing — same leak-prevention rationale as
			// the 404 on subscription mismatch.
			resp.Body.Total = 0
			return resp, nil
		}
		owned = map[string]struct{}{in.TenantUUID: {}}
	}

	for tenantUUID := range owned {
		items, err := h.subs.List(ctx, repository.SubscriptionFilters{
			TenantUUID: tenantUUID,
			Status:     models.SubStatus(in.Status),
		})
		if err != nil {
			return nil, err
		}
		resp.Body.Items = append(resp.Body.Items, items...)
	}
	resp.Body.Total = len(resp.Body.Items)
	return resp, nil
}

type MeSubscriptionRequest struct {
	ID string `path:"id"`
}

func (h *SubscriptionHandler) MeGet(ctx context.Context, in *MeSubscriptionRequest) (*SubscriptionResponse, error) {
	sub, err := h.meGuardSubscription(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	return &SubscriptionResponse{Body: *sub}, nil
}

type MeCancelSubscriptionRequest struct {
	ID   string `path:"id"`
	Body struct {
		AtPeriodEnd bool `json:"atPeriodEnd"`
	}
}

func (h *SubscriptionHandler) MeCancel(ctx context.Context, in *MeCancelSubscriptionRequest) (*SubscriptionResponse, error) {
	if _, err := h.meGuardSubscription(ctx, in.ID); err != nil {
		return nil, err
	}
	sub, err := h.subs.Cancel(ctx, in.ID, in.Body.AtPeriodEnd, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	return &SubscriptionResponse{Body: *sub}, nil
}

func (h *SubscriptionHandler) MeReactivate(ctx context.Context, in *MeSubscriptionRequest) (*SubscriptionResponse, error) {
	if _, err := h.meGuardSubscription(ctx, in.ID); err != nil {
		return nil, err
	}
	sub, err := h.subs.Reactivate(ctx, in.ID, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	return &SubscriptionResponse{Body: *sub}, nil
}

func (h *SubscriptionHandler) MeListInvoices(ctx context.Context, in *MeSubscriptionRequest) (*ListInvoicesResponse, error) {
	if _, err := h.meGuardSubscription(ctx, in.ID); err != nil {
		return nil, err
	}
	items, err := h.invoices.List(ctx, repository.InvoiceFilters{SubscriptionUUID: in.ID})
	if err != nil {
		return nil, err
	}
	resp := &ListInvoicesResponse{}
	resp.Body.Items = items
	resp.Body.Total = len(items)
	return resp, nil
}

type MeListActivityRequest struct {
	ID    string `path:"id"`
	Limit int64  `query:"limit" default:"100" maximum:"500"`
}

func (h *SubscriptionHandler) MeListActivity(ctx context.Context, in *MeListActivityRequest) (*ListActivityResponse, error) {
	if _, err := h.meGuardSubscription(ctx, in.ID); err != nil {
		return nil, err
	}
	items, err := h.activity.List(ctx, in.ID, in.Limit)
	if err != nil {
		return nil, err
	}
	resp := &ListActivityResponse{}
	resp.Body.Items = items
	resp.Body.Total = len(items)
	return resp, nil
}

// actorFrom returns the UUID of the authenticated user for audit-log
// attribution. Falls back to "system" when the middleware has not populated
// the context (background jobs, tests).
func actorFrom(ctx context.Context) string {
	if uuid, ok := middleware.GetUserUUID(ctx); ok && uuid != "" {
		return uuid
	}
	return "system"
}
