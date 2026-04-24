// Package services — device trust service.
//
// Section C item #3 of the 2026-04-24 auth roadmap. A user who has
// completed MFA can opt into "trust this device for 30 days" on the
// verify page; on subsequent logins from the same device + fingerprint
// the MFA prompt is skipped and a full token pair is minted with
// amr carrying the prior factor + "device_trust".
//
// The service is intentionally narrow — just mark/check/revoke. The
// login-flow integration lives in PasswordAuthService.completeLogin
// (web + mobile password paths) and in the MFA verify handlers (which
// call MarkTrusted when the request's trustDevice flag is set).
package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/orkestra/backend/internal/core/auth/models"
	"github.com/orkestra/backend/internal/core/auth/repository"
)

// DeviceTrustService is the caller-facing façade over
// DeviceTrustRepository. Handlers never touch the repo directly.
type DeviceTrustService interface {
	// MarkTrusted grants "skip MFA on login" to the caller's current
	// device for the configured DeviceTrustDuration. Supersedes any
	// prior active grant for the same (user, deviceID). grantedAMR
	// records the factor that was just verified so the later trusted-
	// device login can echo it into the new token's amr claim.
	MarkTrusted(ctx context.Context, in MarkTrustedInput) error

	// IsTrusted returns true when the user has an active,
	// non-expired, non-revoked trust row for this device AND the
	// supplied fingerprint matches the one recorded at grant time.
	// A fingerprint mismatch counts as untrusted — callers then fall
	// through to the standard MFA prompt.
	IsTrusted(ctx context.Context, userUUID, deviceID, fingerprint string) (bool, *models.DeviceTrustDoc, error)

	// ListActive returns the active trust rows the caller has — used
	// by the self-service UI ("these devices won't ask for MFA on
	// login until their trustedUntil timestamp").
	ListActive(ctx context.Context, userUUID string) ([]*models.DeviceTrustDoc, error)

	// RevokeByDevice drops trust for one device. Ignores "not found"
	// silently so the handler can return 204 regardless of whether
	// the device was trusted in the first place (idempotent).
	RevokeByDevice(ctx context.Context, userUUID, deviceID, reason string) error

	// RevokeAllByUser drops every active trust for the user. Called
	// from password-change, MFA-factor-remove, admin MFA reset, and
	// the user-facing "revoke all trusted devices" button.
	RevokeAllByUser(ctx context.Context, userUUID, reason string) error
}

// MarkTrustedInput bundles the metadata captured at grant time. All
// fields except UserUUID / DeviceID / Fingerprint / GrantedAMR are
// optional — they're persisted verbatim so audit queries can
// reconstruct the session that created the trust.
type MarkTrustedInput struct {
	UserUUID    string
	DeviceID    string
	Fingerprint string
	DeviceName  string
	Platform    string
	IPAddress   string
	UserAgent   string
	GrantedAMR  string
	// Duration, when non-zero, overrides the default trust window.
	// Tests and admin tooling use this to exercise expired grants.
	Duration time.Duration
}

type deviceTrustService struct {
	repo     repository.DeviceTrustRepository
	duration time.Duration
	logger   *slog.Logger
	// clock is injectable so expiry-edge tests don't depend on wall
	// time. Defaults to time.Now.
	clock func() time.Time
}

// NewDeviceTrustService builds the service. duration = 0 falls back to
// models.DeviceTrustDuration (30d). logger = nil falls back to slog.Default.
func NewDeviceTrustService(repo repository.DeviceTrustRepository, duration time.Duration, logger *slog.Logger) DeviceTrustService {
	if duration <= 0 {
		duration = models.DeviceTrustDuration
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &deviceTrustService{
		repo:     repo,
		duration: duration,
		logger:   logger,
		clock:    time.Now,
	}
}

func (s *deviceTrustService) MarkTrusted(ctx context.Context, in MarkTrustedInput) error {
	if s.repo == nil {
		return nil
	}
	if in.UserUUID == "" || in.DeviceID == "" {
		// Best-effort: silently skip. A login flow that can't supply
		// both identifiers shouldn't hard-fail — the user is already
		// MFA-verified and the token is issuable.
		return nil
	}
	duration := in.Duration
	if duration <= 0 {
		duration = s.duration
	}
	now := s.clock().UTC()
	doc := &models.DeviceTrustDoc{
		UUID:         uuid.NewString(),
		UserUUID:     in.UserUUID,
		DeviceID:     in.DeviceID,
		Fingerprint:  in.Fingerprint,
		DeviceName:   in.DeviceName,
		Platform:     in.Platform,
		IPAddress:    in.IPAddress,
		UserAgent:    in.UserAgent,
		GrantedAMR:   in.GrantedAMR,
		TrustedAt:    now,
		TrustedUntil: now.Add(duration),
		LastUsedAt:   now,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	return s.repo.Upsert(ctx, doc)
}

func (s *deviceTrustService) IsTrusted(ctx context.Context, userUUID, deviceID, fingerprint string) (bool, *models.DeviceTrustDoc, error) {
	if s.repo == nil || userUUID == "" || deviceID == "" {
		return false, nil, nil
	}
	doc, err := s.repo.GetActive(ctx, userUUID, deviceID)
	if err != nil {
		if err == repository.ErrDeviceTrustNotFound {
			return false, nil, nil
		}
		return false, nil, err
	}
	// Fingerprint cross-check: a spoofed deviceID without the original
	// device's fingerprint must not pass. When the grant was made
	// without a fingerprint (fallback for older clients), accept any
	// request — the device_id was treated as the only identifier at
	// grant time, so requiring it now would lock legitimate users out.
	if doc.Fingerprint != "" && doc.Fingerprint != fingerprint {
		s.logger.Warn("device_trust: fingerprint mismatch, treating as untrusted",
			slog.String("user_uuid", userUUID),
			slog.String("device_id", deviceID))
		return false, nil, nil
	}
	// Update lastUsedAt best-effort; failure doesn't disqualify the
	// session from being trusted.
	if err := s.repo.MarkUsed(ctx, userUUID, deviceID); err != nil {
		s.logger.Warn("device_trust: mark used failed",
			slog.String("user_uuid", userUUID),
			slog.String("device_id", deviceID),
			slog.String("error", err.Error()))
	}
	return true, doc, nil
}

func (s *deviceTrustService) ListActive(ctx context.Context, userUUID string) ([]*models.DeviceTrustDoc, error) {
	if s.repo == nil || userUUID == "" {
		return nil, nil
	}
	return s.repo.ListActiveByUser(ctx, userUUID)
}

func (s *deviceTrustService) RevokeByDevice(ctx context.Context, userUUID, deviceID, reason string) error {
	if s.repo == nil || userUUID == "" || deviceID == "" {
		return nil
	}
	err := s.repo.RevokeByDevice(ctx, userUUID, deviceID, reason)
	if err != nil && err == repository.ErrDeviceTrustNotFound {
		return nil // idempotent — revoking an already-revoked device is a no-op
	}
	return err
}

func (s *deviceTrustService) RevokeAllByUser(ctx context.Context, userUUID, reason string) error {
	if s.repo == nil || userUUID == "" {
		return nil
	}
	count, err := s.repo.RevokeAllByUser(ctx, userUUID, reason)
	if err != nil {
		return err
	}
	if count > 0 {
		s.logger.Info("device_trust: revoked grants for user",
			slog.String("user_uuid", userUUID),
			slog.Int64("count", count),
			slog.String("reason", reason))
	}
	return nil
}
