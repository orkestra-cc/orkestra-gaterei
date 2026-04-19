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

	// Short-circuit activation when the environment skips email verification
	// (dev / smoke tests / AUTH_REQUIRE_EMAIL_VERIFICATION=false). Without
	// this branch the tenant would remain in `provisioning` forever because
	// no VerifyEmail call ever fires. In production-like envs user.EmailVerified
	// is false here and the activator runs later from the verify-email hook.
	if user.EmailVerified {
		if err := s.tenant.ActivateTenant(ctx, tenant.UUID); err != nil {
			s.logger.Warn("onboarding: immediate activation failed",
				slog.String("userUUID", user.UUID),
				slog.String("tenantUUID", tenant.UUID),
				slog.String("error", err.Error()),
			)
		}
	}

	return &RegisterResult{
		UserUUID:             user.UUID,
		TenantUUID:           tenant.UUID,
		TenantSlug:           tenant.Slug,
		RequiresVerification: !user.EmailVerified,
	}, nil
}

// ActivateOnVerify flips every `provisioning` tenant the user owns into
// `active`. Invoked from PasswordAuthService via the OnboardingActivator
// callback after a successful email verification. The hook is best-effort:
// returning an error is logged by the auth service but does not roll back
// the verification itself.
//
// Why iterate memberships: a user may own multiple tenants (e.g. an agency
// that spun up several external tenants from the same login). We activate
// every provisioning tenant they own; tenants they merely belong to but
// don't own are not touched. Status is read fresh from the tenant service
// so suspended/archived tenants are never resurrected by this path.
func (s *Service) ActivateOnVerify(ctx context.Context, userUUID string) error {
	if userUUID == "" {
		return errors.New("onboarding: ActivateOnVerify requires userUUID")
	}
	memberships, err := s.tenant.ListUserMemberships(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("onboarding: list memberships: %w", err)
	}
	for _, m := range memberships {
		if !m.IsOwner {
			continue
		}
		t, err := s.tenant.GetTenant(ctx, m.TenantUUID)
		if err != nil || t == nil {
			s.logger.Warn("onboarding: skip activate, tenant lookup failed",
				slog.String("userUUID", userUUID),
				slog.String("tenantUUID", m.TenantUUID),
			)
			continue
		}
		if t.Status != iface.TenantStatusProvisioning {
			continue
		}
		if err := s.tenant.ActivateTenant(ctx, m.TenantUUID); err != nil {
			s.logger.Warn("onboarding: activate tenant failed",
				slog.String("userUUID", userUUID),
				slog.String("tenantUUID", m.TenantUUID),
				slog.String("error", err.Error()),
			)
			continue
		}
		s.logger.Info("onboarding: tenant activated on email verification",
			slog.String("userUUID", userUUID),
			slog.String("tenantUUID", m.TenantUUID),
		)
	}
	return nil
}
