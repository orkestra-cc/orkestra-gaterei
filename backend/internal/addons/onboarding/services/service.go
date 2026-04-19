// Package services holds the onboarding orchestration service — the glue
// that turns an anonymous signup request into (a) a new user and (b) a
// freshly-provisioned external tenant owned by that user. Thin wrapper
// over PasswordAuthService.Register + TenantProvider.CreateExternalTenant
// so the HTTP handler stays trivial.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	authServices "github.com/orkestra/backend/internal/core/auth/services"
	"github.com/orkestra/backend/internal/shared/iface"
)

// RegisterInput is the payload for the anonymous onboarding endpoint.
type RegisterInput struct {
	Email      string
	Password   string
	FullName   string
	TenantName string
	// TenantSlug is optional — derived from TenantName when empty.
	TenantSlug string
	// Plan is an informational label on the tenant; defaults to "free".
	Plan string
	IP   string
}

// RegisterResult bundles what the handler needs to render a response:
// the new user UUID, the new tenant UUID/slug, and whether the caller
// still needs to click a verification link before they can log in.
type RegisterResult struct {
	UserUUID             string
	TenantUUID           string
	TenantSlug           string
	RequiresVerification bool
}

// Service orchestrates anonymous self-service signup.
type Service struct {
	passwordAuth *authServices.PasswordAuthService
	tenant       iface.TenantProvider
	logger       *slog.Logger
}

// New wires the collaborators.
func New(passwordAuth *authServices.PasswordAuthService, tenant iface.TenantProvider, logger *slog.Logger) *Service {
	return &Service{passwordAuth: passwordAuth, tenant: tenant, logger: logger}
}

// Register runs the full anonymous onboarding flow:
//
//  1. PasswordAuthService.Register creates the user, sends the verification
//     email (when AUTH_REQUIRE_EMAIL_VERIFICATION=true), and — if this is the
//     very first account on a fresh install — promotes them to super_admin.
//  2. TenantProvider.CreateExternalTenant provisions a Tier-2 tenant owned
//     by the new user. Kind is forced external and SignupChannel to
//     self_serve so analytics can tell onboarding cohorts apart from
//     admin-created tenants.
//
// If step 2 fails after step 1 succeeded, the user is left in the system
// without a tenant. That's recoverable (an admin can attach them to a
// tenant manually, or a retry endpoint can be added) and safer than
// destroying an account the caller has already received an email for.
func (s *Service) Register(ctx context.Context, in RegisterInput) (*RegisterResult, error) {
	if strings.TrimSpace(in.TenantName) == "" {
		return nil, errors.New("onboarding: tenantName is required")
	}

	user, err := s.passwordAuth.Register(ctx, authServices.RegisterInput{
		Email:    in.Email,
		Password: in.Password,
		FullName: in.FullName,
		IP:       in.IP,
	})
	if err != nil {
		return nil, err
	}

	tenant, err := s.tenant.ProvisionExternalTenant(ctx, user.UUID, iface.OnboardingTenantInput{
		Name: in.TenantName,
		Slug: in.TenantSlug,
		Plan: in.Plan,
	})
	if err != nil {
		s.logger.Error("onboarding: tenant provisioning failed after user creation",
			slog.String("userUUID", user.UUID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("onboarding: provision tenant: %w", err)
	}

	return &RegisterResult{
		UserUUID:             user.UUID,
		TenantUUID:           tenant.UUID,
		TenantSlug:           tenant.Slug,
		RequiresVerification: !user.EmailVerified,
	}, nil
}
