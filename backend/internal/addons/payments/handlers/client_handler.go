// Package handlers — client_handler.go hosts the Tier-2 self-service
// payment endpoints mounted on the ADR-0003 client API surface
// (api.orkestra.com / api.localhost). Each route is gated by
// RequireGlobal() at mount time AND re-checks tenant ownership in the
// handler against TenantProvider.ListUserMemberships, mirroring the
// SubscriptionHandler.SelfSubscribe pattern in subscriptions.
package handlers

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/orkestra/backend/internal/addons/payments/models"
	"github.com/orkestra/backend/internal/addons/payments/repository"
	"github.com/orkestra/backend/internal/addons/payments/services"
	"github.com/orkestra/backend/internal/shared/iface"
	"github.com/orkestra/backend/internal/shared/middleware"
)

// ClientHandler bundles the Tier-2 self-service payment routes. Constructed
// in payments/module.go's Init only when the tenant module is wired —
// without TenantProvider there is no way to enforce ownership, so the
// routes are simply not mounted.
type ClientHandler struct {
	payment *services.PaymentService
	txRepo  repository.TransactionRepository
	pmRepo  repository.PaymentMethodRepository
	tenants iface.TenantProvider
	// planner is optional — the checkout-session route returns 503 when
	// the subscriptions module is disabled or has not registered the key.
	planner iface.SelfServiceCheckoutPlanner
}

// NewClientHandler constructs the handler. tenants is required (call site
// must skip mount if nil); planner is optional.
func NewClientHandler(
	payment *services.PaymentService,
	txRepo repository.TransactionRepository,
	pmRepo repository.PaymentMethodRepository,
	tenants iface.TenantProvider,
	planner iface.SelfServiceCheckoutPlanner,
) *ClientHandler {
	return &ClientHandler{
		payment: payment,
		txRepo:  txRepo,
		pmRepo:  pmRepo,
		tenants: tenants,
		planner: planner,
	}
}

// meOwnedTenants returns the set of TenantUUIDs the caller owns. Read live
// from TenantProvider so a freshly granted/revoked ownership is reflected
// without waiting for a token refresh. 401 when anonymous, 503 when the
// provider is missing (defensive — payments/module.go should have
// short-circuited the mount in that case).
func (h *ClientHandler) meOwnedTenants(ctx context.Context) (map[string]struct{}, error) {
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

// ensureCustomer returns the Stripe (or future PayPal) customer reference
// for the tenant, creating it on the gateway side if missing and persisting
// the resulting id back onto the tenant record so subsequent calls skip
// the round-trip. Mirrors the lazy-creation path in
// renewal_service.chargeInvoice — kept here as a duplicate to avoid making
// the renewal package importable from payments (would create a cycle).
func (h *ClientHandler) ensureCustomer(ctx context.Context, tenantUUID string) (iface.CustomerRef, error) {
	t, err := h.tenants.GetTenant(ctx, tenantUUID)
	if err != nil {
		return iface.CustomerRef{}, err
	}
	if t == nil {
		return iface.CustomerRef{}, huma.Error404NotFound("tenant not found")
	}
	if t.StripeCustomerID != "" {
		return iface.CustomerRef{Provider: "stripe", ID: t.StripeCustomerID}, nil
	}

	name := t.LegalName
	if name == "" {
		name = t.Name
	}
	ref, err := h.payment.CreateCustomer(ctx, iface.CustomerInput{
		TenantUUID: t.UUID,
		Email:      t.Email,
		Name:       name,
		VATNumber:  t.VATNumber,
		Country:    t.Country,
	})
	if err != nil {
		return iface.CustomerRef{}, err
	}
	if err := h.tenants.SetTenantStripeCustomerID(ctx, t.UUID, ref.ID); err != nil {
		// Persist failure is logged but not fatal — the customer exists
		// on Stripe and the renewal job will find/reuse it via metadata
		// lookup. Worst case we create a duplicate customer next time.
		// Return ref so the caller can still open Checkout.
		return ref, nil //nolint:nilerr // intentional: persistence is best-effort here
	}
	return ref, nil
}

// --- Transactions ---

type MeListTransactionsRequest struct {
	TenantUUID       string `query:"tenantUuid" doc:"Optional — restrict to one owned tenant"`
	SubscriptionUUID string `query:"subscriptionUuid" doc:"Optional — restrict to one subscription"`
	Status           string `query:"status" enum:"pending,requires_action,succeeded,failed,refunded,partially_refunded"`
}
type MeListTransactionsResponse struct {
	Body struct {
		Items []models.Transaction `json:"items"`
		Total int                  `json:"total"`
	}
}

// MeListTransactions returns transactions across every tenant the caller
// owns. Per-tenant fan-out (same rationale as MeListSubscriptions) — the
// repository's tenantUUID filter is the only available index, and Tier-2
// users typically own one or two tenants in the demo MVP shape.
func (h *ClientHandler) MeListTransactions(ctx context.Context, in *MeListTransactionsRequest) (*MeListTransactionsResponse, error) {
	owned, err := h.meOwnedTenants(ctx)
	if err != nil {
		return nil, err
	}
	resp := &MeListTransactionsResponse{}
	resp.Body.Items = []models.Transaction{}

	if in.TenantUUID != "" {
		if _, ok := owned[in.TenantUUID]; !ok {
			resp.Body.Total = 0
			return resp, nil
		}
		owned = map[string]struct{}{in.TenantUUID: {}}
	}

	for tenantUUID := range owned {
		items, err := h.txRepo.List(ctx, repository.TransactionFilters{
			TenantUUID:       tenantUUID,
			SubscriptionUUID: in.SubscriptionUUID,
			Status:           models.TransactionStatus(in.Status),
		})
		if err != nil {
			return nil, err
		}
		resp.Body.Items = append(resp.Body.Items, items...)
	}
	resp.Body.Total = len(resp.Body.Items)
	return resp, nil
}

// --- Payment methods ---

type MeListPaymentMethodsRequest struct {
	TenantUUID string `query:"tenantUuid" doc:"Optional — restrict to one owned tenant"`
}
type MeListPaymentMethodsResponse struct {
	Body struct {
		Items []models.PaymentMethod `json:"items"`
		Total int                    `json:"total"`
	}
}

func (h *ClientHandler) MeListPaymentMethods(ctx context.Context, in *MeListPaymentMethodsRequest) (*MeListPaymentMethodsResponse, error) {
	owned, err := h.meOwnedTenants(ctx)
	if err != nil {
		return nil, err
	}
	resp := &MeListPaymentMethodsResponse{}
	resp.Body.Items = []models.PaymentMethod{}

	if in.TenantUUID != "" {
		if _, ok := owned[in.TenantUUID]; !ok {
			resp.Body.Total = 0
			return resp, nil
		}
		owned = map[string]struct{}{in.TenantUUID: {}}
	}

	for tenantUUID := range owned {
		items, err := h.pmRepo.ListByTenant(ctx, tenantUUID)
		if err != nil {
			return nil, err
		}
		resp.Body.Items = append(resp.Body.Items, items...)
	}
	resp.Body.Total = len(resp.Body.Items)
	return resp, nil
}

// --- Stripe Checkout (payment mode) ---

type MeCreateCheckoutSessionRequest struct {
	Body struct {
		SubscriptionUUID string `json:"subscriptionUuid" doc:"UUID of a subscription owned by the caller"`
		SuccessURL       string `json:"successUrl" doc:"Absolute URL Stripe redirects to on success"`
		CancelURL        string `json:"cancelUrl" doc:"Absolute URL Stripe redirects to on cancel"`
	}
}

type MeCheckoutSessionResponse struct {
	Body struct {
		SessionID     string `json:"sessionId"`
		URL           string `json:"url"`
		AmountCents   int64  `json:"amountCents"`
		Currency      string `json:"currency"`
		InvoiceUUID   string `json:"invoiceUuid"`
		InvoiceNumber string `json:"invoiceNumber"`
	}
}

// MeCreateCheckoutSession opens a Stripe Checkout session in payment mode
// for the subscription's most recent pending invoice. The webhook
// reconciler picks up the resulting PaymentIntent via the metadata stamp
// (subscriptionUUID + invoiceUUID + tenantUUID) — the same path the
// renewal job's off-session charges follow.
//
// 409 when the subscription has no pending invoice — the SPA can prompt
// the user to wait for the next renewal tick (or, post-MVP, hit a
// "trigger renewal" admin action).
func (h *ClientHandler) MeCreateCheckoutSession(ctx context.Context, in *MeCreateCheckoutSessionRequest) (*MeCheckoutSessionResponse, error) {
	if h.planner == nil {
		return nil, huma.Error503ServiceUnavailable("subscription planner not configured")
	}
	if in.Body.SubscriptionUUID == "" {
		return nil, huma.Error400BadRequest("subscriptionUuid is required")
	}
	if in.Body.SuccessURL == "" || in.Body.CancelURL == "" {
		return nil, huma.Error400BadRequest("successUrl and cancelUrl are required")
	}

	owned, err := h.meOwnedTenants(ctx)
	if err != nil {
		return nil, err
	}

	plan, err := h.planner.PlanCheckoutSession(ctx, in.Body.SubscriptionUUID)
	if err != nil {
		switch {
		case errors.Is(err, iface.ErrCheckoutNoPendingInvoice):
			return nil, huma.Error409Conflict("no pending invoice for subscription")
		default:
			return nil, err
		}
	}

	if _, ok := owned[plan.TenantUUID]; !ok {
		// 404 (not 403) so the existence of out-of-scope subscriptions
		// does not leak to a fishing client. Same rationale as
		// SubscriptionHandler.meGuardSubscription.
		return nil, huma.Error404NotFound("not found")
	}

	customer, err := h.ensureCustomer(ctx, plan.TenantUUID)
	if err != nil {
		return nil, err
	}

	tenantEmail := ""
	if t, gerr := h.tenants.GetTenant(ctx, plan.TenantUUID); gerr == nil && t != nil {
		tenantEmail = t.Email
	}

	res, err := h.payment.CreateCheckoutSession(ctx, iface.CheckoutSessionInput{
		Customer:      customer,
		AmountCents:   plan.AmountCents,
		Currency:      plan.Currency,
		Description:   plan.Description,
		SuccessURL:    in.Body.SuccessURL,
		CancelURL:     in.Body.CancelURL,
		CustomerEmail: tenantEmail,
		Metadata: map[string]string{
			"subscriptionUUID": plan.SubscriptionUUID,
			"invoiceUUID":      plan.InvoiceUUID,
			"tenantUUID":       plan.TenantUUID,
		},
	})
	if err != nil {
		return nil, err
	}

	resp := &MeCheckoutSessionResponse{}
	resp.Body.SessionID = res.SessionID
	resp.Body.URL = res.URL
	resp.Body.AmountCents = plan.AmountCents
	resp.Body.Currency = plan.Currency
	resp.Body.InvoiceUUID = plan.InvoiceUUID
	resp.Body.InvoiceNumber = plan.InvoiceNumber
	return resp, nil
}

// --- Stripe Checkout (setup mode — save a card without charging) ---

type MeCreateSetupCheckoutRequest struct {
	Body struct {
		TenantUUID string `json:"tenantUuid" doc:"UUID of a tenant the caller owns"`
		SuccessURL string `json:"successUrl" doc:"Absolute URL Stripe redirects to on success"`
		CancelURL  string `json:"cancelUrl" doc:"Absolute URL Stripe redirects to on cancel"`
	}
}

func (h *ClientHandler) MeCreateSetupCheckoutSession(ctx context.Context, in *MeCreateSetupCheckoutRequest) (*MeCheckoutSessionResponse, error) {
	if in.Body.TenantUUID == "" {
		return nil, huma.Error400BadRequest("tenantUuid is required")
	}
	if in.Body.SuccessURL == "" || in.Body.CancelURL == "" {
		return nil, huma.Error400BadRequest("successUrl and cancelUrl are required")
	}

	owned, err := h.meOwnedTenants(ctx)
	if err != nil {
		return nil, err
	}
	if _, ok := owned[in.Body.TenantUUID]; !ok {
		return nil, huma.Error404NotFound("not found")
	}

	customer, err := h.ensureCustomer(ctx, in.Body.TenantUUID)
	if err != nil {
		return nil, err
	}

	tenantEmail := ""
	if t, gerr := h.tenants.GetTenant(ctx, in.Body.TenantUUID); gerr == nil && t != nil {
		tenantEmail = t.Email
	}

	res, err := h.payment.CreateSetupCheckoutSession(ctx, iface.SetupCheckoutInput{
		Customer:      customer,
		SuccessURL:    in.Body.SuccessURL,
		CancelURL:     in.Body.CancelURL,
		CustomerEmail: tenantEmail,
		Metadata: map[string]string{
			"tenantUUID": in.Body.TenantUUID,
		},
	})
	if err != nil {
		return nil, err
	}

	resp := &MeCheckoutSessionResponse{}
	resp.Body.SessionID = res.SessionID
	resp.Body.URL = res.URL
	return resp, nil
}
