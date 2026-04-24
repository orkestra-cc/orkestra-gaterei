// Package services — risk assessment.
//
// Section C item #1 of the 2026-04-24 auth roadmap replaces the previous
// zero-score stub with a real login-risk scorer. The score is composed
// from three factors derived from existing auth_sessions history:
//
//   new_device_fingerprint — user has prior sessions but none from this
//   new_ip                 — user has prior sessions but none from this IP
//   rapid_ip_change        — last prior session was <rapidWindow ago
//                            from a different IP
//
// First-ever logins score 0.0 (no baseline to compare against) — the
// scorer is deliberately conservative on bootstrap to avoid false
// positives on sign-up.
package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/orkestra/backend/internal/core/auth/models"
	"github.com/orkestra/backend/internal/core/auth/repository"
)

// RiskAssessmentService handles risk assessment functionality.
type RiskAssessmentService interface {
	AssessRisk(ctx context.Context, userUUID string, securityCtx *models.SecurityContext) (*models.RiskAssessment, error)
	AssessLoginRisk(ctx context.Context, userUUID string, securityCtx *models.SecurityContext) (*models.RiskAssessment, error)
}

// Factor weights. The three weights sum to 0.9 so a login that trips all
// three still leaves headroom for C4/C5 additions (impossible-travel,
// failed-login density) without forcing a rebalance. Each weight maps
// intuitively to the severity of the underlying signal: rapid IP change
// is the strongest single indicator of a stolen credential, an unknown
// IP on a known device is the weakest. Exported so tests and the Cedar
// attribute plumb can reference the same numbers.
const (
	WeightNewDeviceFingerprint = 0.3
	WeightNewIP                = 0.2
	WeightRapidIPChange        = 0.4

	// rapidWindow is how recently the prior session must have started for
	// a different-IP follow-up to count as "rapid". Five minutes is tight
	// enough that legitimate mobile tower handovers don't fire (those
	// typically carry the same session anyway) while still catching the
	// credential-stuffing case where an attacker races the legitimate
	// user's session.
	rapidWindow = 5 * time.Minute

	// historyLookback bounds the count queries so an account with years
	// of history doesn't scan the whole collection on every login. Six
	// months is long enough to cover seasonal travel and short enough
	// that a device retired >6m ago reads as new (intended — stale
	// fingerprints shouldn't whitelist a returning attacker).
	historyLookback = 180 * 24 * time.Hour
)

// Risk-level thresholds. Exposed as constants so the step-up middleware
// (C2) and the Cedar `principal.risk_level` attribute use the same
// mapping the scorer emits.
const (
	RiskLevelLow      = "low"      // [0.0, 0.3)
	RiskLevelMedium   = "medium"   // [0.3, 0.5)
	RiskLevelHigh     = "high"     // [0.5, 0.7)
	RiskLevelCritical = "critical" // [0.7, 1.0]
)

// RiskLevelForScore maps a [0,1] risk score to its bucket. Shared
// between the scorer and downstream consumers so bucket boundaries stay
// in one place.
func RiskLevelForScore(score float64) string {
	switch {
	case score >= 0.7:
		return RiskLevelCritical
	case score >= 0.5:
		return RiskLevelHigh
	case score >= 0.3:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

type riskAssessmentService struct {
	sessions repository.AuthSessionRepository
	logger   *slog.Logger
	// clock is injectable so the rapid_ip_change test can pin time.Now()
	// against a deterministic prior-session createdAt.
	clock func() time.Time
}

// NewRiskAssessmentService builds the scorer. sessions is required; a
// nil repository disables all Mongo-backed factors and the scorer
// returns a zero-score assessment (no bias, no crash). logger is
// optional — nil falls back to slog.Default.
func NewRiskAssessmentService(sessions repository.AuthSessionRepository, logger *slog.Logger) RiskAssessmentService {
	if logger == nil {
		logger = slog.Default()
	}
	return &riskAssessmentService{
		sessions: sessions,
		logger:   logger,
		clock:    time.Now,
	}
}

// AssessRisk is the general-purpose entry point. Today it mirrors
// AssessLoginRisk — keeping both surfaces available so refresh-time and
// login-time callers can diverge later without an interface break.
func (r *riskAssessmentService) AssessRisk(ctx context.Context, userUUID string, securityCtx *models.SecurityContext) (*models.RiskAssessment, error) {
	return r.AssessLoginRisk(ctx, userUUID, securityCtx)
}

// AssessLoginRisk scores a login attempt against the user's session
// history. Returns a zero-score assessment when userUUID is empty, when
// the session repo is nil, or when the user has no prior sessions — in
// all three cases there's no baseline to compare against and emitting a
// non-zero score would be a false positive.
func (r *riskAssessmentService) AssessLoginRisk(ctx context.Context, userUUID string, securityCtx *models.SecurityContext) (*models.RiskAssessment, error) {
	now := r.clock().UTC()
	assessment := &models.RiskAssessment{
		Score:      0.0,
		Level:      RiskLevelLow,
		Factors:    []models.RiskFactor{},
		AssessedAt: now,
	}

	if userUUID == "" || r.sessions == nil || securityCtx == nil {
		return assessment, nil
	}

	// All three factors require prior session history. Query the most
	// recent session once; absence of a row means first-ever login —
	// score stays 0.
	prior, err := r.sessions.GetMostRecentSessionByUser(ctx, userUUID)
	if err != nil {
		// A lookup failure must not block login. Log and fall through to
		// zero score so operators can alert on the signal without the
		// scorer becoming a single point of failure.
		r.logger.Warn("risk: prior session lookup failed",
			slog.String("user_uuid", userUUID),
			slog.String("error", err.Error()))
		return assessment, nil
	}
	if prior == nil {
		return assessment, nil
	}

	since := now.Add(-historyLookback)

	// Factor 1 — new device fingerprint. Fires only when the login path
	// supplied a fingerprint; missing fingerprint (e.g. older mobile
	// clients) simply can't trigger this factor.
	if securityCtx.Fingerprint != "" {
		count, err := r.sessions.CountSessionsByUserAndFingerprint(ctx, userUUID, securityCtx.Fingerprint, since)
		if err != nil {
			r.logger.Warn("risk: fingerprint count failed",
				slog.String("user_uuid", userUUID),
				slog.String("error", err.Error()))
		} else if count == 0 {
			assessment.Factors = append(assessment.Factors, models.RiskFactor{
				Type:        "device",
				Description: "login from a fingerprint never seen on this account",
				Weight:      WeightNewDeviceFingerprint,
				Severity:    "medium",
				Details:     map[string]interface{}{"factor": "new_device_fingerprint"},
			})
			assessment.Score += WeightNewDeviceFingerprint
		}
	}

	// Factor 2 — new IP.
	if securityCtx.IPAddress != "" {
		count, err := r.sessions.CountSessionsByUserAndIP(ctx, userUUID, securityCtx.IPAddress, since)
		if err != nil {
			r.logger.Warn("risk: ip count failed",
				slog.String("user_uuid", userUUID),
				slog.String("error", err.Error()))
		} else if count == 0 {
			assessment.Factors = append(assessment.Factors, models.RiskFactor{
				Type:        "location",
				Description: "login from an IP never seen on this account",
				Weight:      WeightNewIP,
				Severity:    "low",
				Details:     map[string]interface{}{"factor": "new_ip"},
			})
			assessment.Score += WeightNewIP
		}
	}

	// Factor 3 — rapid IP change. Fires only when the prior session was
	// started inside rapidWindow AND its IP differs from the current IP.
	// Same-IP rapid re-login (refreshing a tab) is benign.
	if securityCtx.IPAddress != "" && prior.IPAddress != "" &&
		prior.IPAddress != securityCtx.IPAddress &&
		now.Sub(prior.CreatedAt) < rapidWindow {
		assessment.Factors = append(assessment.Factors, models.RiskFactor{
			Type:        "behavior",
			Description: "login from a different IP less than 5 minutes after the prior session",
			Weight:      WeightRapidIPChange,
			Severity:    "high",
			Details: map[string]interface{}{
				"factor":            "rapid_ip_change",
				"priorIP":           prior.IPAddress,
				"priorCreatedAt":    prior.CreatedAt,
				"secondsSincePrior": int(now.Sub(prior.CreatedAt).Seconds()),
			},
		})
		assessment.Score += WeightRapidIPChange
	}

	if assessment.Score > 1.0 {
		assessment.Score = 1.0
	}
	assessment.Level = RiskLevelForScore(assessment.Score)
	return assessment, nil
}
