// Package handlers wires the onboarding service to Huma request/response
// shapes. The module owns a single public endpoint so this file is
// intentionally minimal — adding richer flows (plan selection, invite
// redemption, SCIM provisioning) lands later in Phase 3.
package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/orkestra/backend/internal/addons/onboarding/services"
	authServices "github.com/orkestra/backend/internal/core/auth/services"
	"github.com/orkestra/backend/internal/shared/utils"
)

// Handler exposes the onboarding HTTP surface.
type Handler struct {
	svc *services.Service
}

// New wires the handler to the orchestration service.
func New(svc *services.Service) *Handler {
	return &Handler{svc: svc}
}

// OnboardingRegisterBody is the wire-level payload for the public signup
// endpoint. Named concretely (rather than inlined as an anonymous struct)
// so Huma's schema registry emits a unique component name — Huma collapses
// anonymous bodies by their field shape, which collides with the auth
// module's RegisterRequest body when both have Email/Password/FullName.
type OnboardingRegisterBody struct {
	Email      string `json:"email" doc:"Email address" format:"email"`
	Password   string `json:"password" doc:"New password (min 10 chars)"`
	FullName   string `json:"fullName" doc:"Full name"`
	TenantName string `json:"tenantName" doc:"Display name for the new tenant"`
	TenantSlug string `json:"tenantSlug,omitempty" doc:"Optional URL slug; derived from tenantName when empty"`
	Plan       string `json:"plan,omitempty" doc:"Optional plan label (informational only; entitlements are driven by subscriptions)"`
}

// OnboardingRegisterRequest is the Huma request wrapper.
type OnboardingRegisterRequest struct {
	Body OnboardingRegisterBody
}

// OnboardingRegisterResultBody is the wire-level response payload.
type OnboardingRegisterResultBody struct {
	Success              bool   `json:"success"`
	UserUUID             string `json:"userUuid"`
	TenantUUID           string `json:"tenantUuid"`
	TenantSlug           string `json:"tenantSlug"`
	Message              string `json:"message"`
	RequiresVerification bool   `json:"requiresVerification"`
}

// OnboardingRegisterResponse is the Huma response wrapper.
type OnboardingRegisterResponse struct {
	Body OnboardingRegisterResultBody
}

// Register handles POST /v1/onboarding/register. No auth required.
func (h *Handler) Register(ctx context.Context, req *OnboardingRegisterRequest) (*OnboardingRegisterResponse, error) {
	ip := clientIPFromCtx(ctx)
	res, err := h.svc.Register(ctx, services.RegisterInput{
		Email:      req.Body.Email,
		Password:   req.Body.Password,
		FullName:   req.Body.FullName,
		TenantName: req.Body.TenantName,
		TenantSlug: req.Body.TenantSlug,
		Plan:       req.Body.Plan,
		IP:         ip,
	})
	if err != nil {
		return nil, mapError(err)
	}
	out := &OnboardingRegisterResponse{}
	out.Body.Success = true
	out.Body.UserUUID = res.UserUUID
	out.Body.TenantUUID = res.TenantUUID
	out.Body.TenantSlug = res.TenantSlug
	out.Body.RequiresVerification = res.RequiresVerification
	if res.RequiresVerification {
		out.Body.Message = "Account created. Please check your email to verify your address before signing in."
	} else {
		out.Body.Message = "Account created. You can now sign in."
	}
	return out, nil
}

// Register wires the endpoint on the public (unauthenticated) API.
func Register(api huma.API, h *Handler) {
	huma.Register(api, huma.Operation{
		OperationID: "onboarding-register",
		Method:      http.MethodPost,
		Path:        "/v1/onboarding/register",
		Summary:     "Self-service external-tenant signup",
		Description: "Creates a new external (Tier-2) tenant and its owner user in a single anonymous call. When AUTH_REQUIRE_EMAIL_VERIFICATION is true, a verification email is sent and the caller must click the link before logging in.",
		Tags:        []string{"Onboarding"},
	}, h.Register)
}

// mapError translates the underlying service errors to Huma HTTP responses.
// Rate-limit and validation errors from PasswordAuthService carry their
// own HTTP status through shared/errors; everything else falls through
// as a 400 so the frontend can surface a field-level message without
// leaking implementation details.
func mapError(err error) error {
	switch {
	case errors.Is(err, authServices.ErrNotificationDown):
		return huma.Error503ServiceUnavailable("email delivery is not configured")
	case errors.Is(err, authServices.ErrInvalidCredentials):
		return huma.Error400BadRequest("invalid credentials")
	default:
		return huma.Error400BadRequest(err.Error())
	}
}

// clientIPFromCtx extracts the caller IP by reaching into the request the
// Huma middleware stashes on the context under "http_request". Mirrors the
// helper in auth/handlers/password_handler.go; duplicated here rather than
// exported so the two handler packages stay independent.
func clientIPFromCtx(ctx context.Context) string {
	if r, ok := ctx.Value("http_request").(*http.Request); ok && r != nil {
		return utils.GetClientIP(r)
	}
	return ""
}
