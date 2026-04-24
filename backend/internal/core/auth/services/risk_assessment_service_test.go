package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/orkestra/backend/internal/core/auth/models"
	"github.com/orkestra/backend/internal/core/auth/repository"
)

// stubSessionRepo satisfies repository.AuthSessionRepository. The three
// methods the risk scorer consumes return configurable values; every
// other method panics so accidental use surfaces as a loud test failure
// instead of a silent zero value.
type stubSessionRepo struct {
	mostRecent *models.AuthSessionDoc
	mostErr    error

	fpCount int64
	fpErr   error

	ipCount int64
	ipErr   error
}

func (s *stubSessionRepo) GetMostRecentSessionByUser(_ context.Context, _ string) (*models.AuthSessionDoc, error) {
	return s.mostRecent, s.mostErr
}

func (s *stubSessionRepo) CountSessionsByUserAndFingerprint(_ context.Context, _, _ string, _ time.Time) (int64, error) {
	return s.fpCount, s.fpErr
}

func (s *stubSessionRepo) CountSessionsByUserAndIP(_ context.Context, _, _ string, _ time.Time) (int64, error) {
	return s.ipCount, s.ipErr
}

// Unused — panic on accidental use.
func (s *stubSessionRepo) CreateSession(context.Context, *models.AuthSessionDoc) error {
	panic("unexpected CreateSession")
}
func (s *stubSessionRepo) GetByUUID(context.Context, string) (*models.AuthSessionDoc, error) {
	panic("unexpected GetByUUID")
}
func (s *stubSessionRepo) GetByUserAndDevice(context.Context, string, string) (*models.AuthSessionDoc, error) {
	panic("unexpected GetByUserAndDevice")
}
func (s *stubSessionRepo) GetActiveSessionsByUser(context.Context, string) ([]*models.AuthSessionDoc, error) {
	panic("unexpected GetActiveSessionsByUser")
}
func (s *stubSessionRepo) UpdateLastActivity(context.Context, string) error {
	panic("unexpected UpdateLastActivity")
}
func (s *stubSessionRepo) UpdateRiskScore(context.Context, string, float64, string) error {
	panic("unexpected UpdateRiskScore")
}
func (s *stubSessionRepo) AddSecurityEvent(context.Context, string, *models.SecurityEventLog) error {
	panic("unexpected AddSecurityEvent")
}
func (s *stubSessionRepo) UpdateDeviceInfo(context.Context, string, *models.DeviceInfo) error {
	panic("unexpected UpdateDeviceInfo")
}
func (s *stubSessionRepo) TerminateSession(context.Context, string) error {
	panic("unexpected TerminateSession")
}
func (s *stubSessionRepo) TerminateSessionByDevice(context.Context, string, string) error {
	panic("unexpected TerminateSessionByDevice")
}
func (s *stubSessionRepo) TerminateAllUserSessions(context.Context, string) error {
	panic("unexpected TerminateAllUserSessions")
}
func (s *stubSessionRepo) TerminateExpiredSessions(context.Context) (int64, error) {
	panic("unexpected TerminateExpiredSessions")
}
func (s *stubSessionRepo) DeleteAllByUser(context.Context, string) (int64, error) {
	panic("unexpected DeleteAllByUser")
}
func (s *stubSessionRepo) GetSessionStats(context.Context, string) (*repository.SessionStats, error) {
	panic("unexpected GetSessionStats")
}
func (s *stubSessionRepo) GetActiveDevices(context.Context, string) ([]*repository.DeviceSession, error) {
	panic("unexpected GetActiveDevices")
}
func (s *stubSessionRepo) GetSessionsByLocation(context.Context, string, string) ([]*models.AuthSessionDoc, error) {
	panic("unexpected GetSessionsByLocation")
}
func (s *stubSessionRepo) GetHighRiskSessions(context.Context, float64) ([]*models.AuthSessionDoc, error) {
	panic("unexpected GetHighRiskSessions")
}
func (s *stubSessionRepo) RenameDevice(context.Context, string, string, string) error {
	panic("unexpected RenameDevice")
}
func (s *stubSessionRepo) GetDeviceSessionHistory(context.Context, string, string, int) ([]*models.AuthSessionDoc, error) {
	panic("unexpected GetDeviceSessionHistory")
}
func (s *stubSessionRepo) GetRecentSecurityEvents(context.Context, string, string, time.Time) ([]*models.SecurityEventLog, error) {
	panic("unexpected GetRecentSecurityEvents")
}
func (s *stubSessionRepo) GetSuspiciousSessions(context.Context, string) ([]*models.AuthSessionDoc, error) {
	panic("unexpected GetSuspiciousSessions")
}

// Compile-time guard that the stub really satisfies the repository
// interface. If a new method is added upstream the test file won't
// compile — a loud reminder to extend the stub.
var _ repository.AuthSessionRepository = (*stubSessionRepo)(nil)

// newScorer constructs a scorer with the stub repo and a pinned clock so
// rapid_ip_change assertions don't depend on wall time.
func newScorer(t *testing.T, repo *stubSessionRepo, now time.Time) *riskAssessmentService {
	t.Helper()
	svc := NewRiskAssessmentService(repo, nil).(*riskAssessmentService)
	svc.clock = func() time.Time { return now }
	return svc
}

// --- tests ---

func TestRiskLevelForScore(t *testing.T) {
	cases := map[float64]string{
		0.0:  RiskLevelLow,
		0.29: RiskLevelLow,
		0.3:  RiskLevelMedium,
		0.49: RiskLevelMedium,
		0.5:  RiskLevelHigh,
		0.69: RiskLevelHigh,
		0.7:  RiskLevelCritical,
		1.0:  RiskLevelCritical,
	}
	for in, want := range cases {
		if got := RiskLevelForScore(in); got != want {
			t.Errorf("RiskLevelForScore(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestAssessLoginRisk_ZeroScoreOnFirstLogin(t *testing.T) {
	// No prior session → first-ever login → every factor is inert.
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	scorer := newScorer(t, &stubSessionRepo{mostRecent: nil}, now)
	got, err := scorer.AssessLoginRisk(context.Background(), "u1", &models.SecurityContext{
		IPAddress:   "8.8.8.8",
		Fingerprint: "fp-abc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Score != 0.0 {
		t.Errorf("first login score: want 0.0, got %v", got.Score)
	}
	if got.Level != RiskLevelLow {
		t.Errorf("first login level: want %q, got %q", RiskLevelLow, got.Level)
	}
	if len(got.Factors) != 0 {
		t.Errorf("first login should have no factors, got %+v", got.Factors)
	}
}

func TestAssessLoginRisk_ZeroScoreWhenCtxNil(t *testing.T) {
	scorer := newScorer(t, &stubSessionRepo{}, time.Now())
	got, err := scorer.AssessLoginRisk(context.Background(), "u1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Score != 0.0 {
		t.Errorf("nil ctx score: want 0.0, got %v", got.Score)
	}
}

func TestAssessLoginRisk_NewFingerprintAndIP(t *testing.T) {
	// User has prior history but this fingerprint + IP are both new.
	// Expected factors: new_device_fingerprint (0.3) + new_ip (0.2) = 0.5
	// → high.
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	repo := &stubSessionRepo{
		mostRecent: &models.AuthSessionDoc{
			UUID:      "prior-session",
			UserUUID:  "u1",
			IPAddress: "10.0.0.5", // different but not within rapid window
			CreatedAt: now.Add(-48 * time.Hour),
		},
		fpCount: 0, // new fingerprint
		ipCount: 0, // new IP
	}
	scorer := newScorer(t, repo, now)
	got, err := scorer.AssessLoginRisk(context.Background(), "u1", &models.SecurityContext{
		IPAddress:   "8.8.8.8",
		Fingerprint: "fp-new",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := WeightNewDeviceFingerprint + WeightNewIP
	if got.Score != want {
		t.Errorf("score: want %v, got %v", want, got.Score)
	}
	if got.Level != RiskLevelHigh {
		t.Errorf("level: want %q, got %q", RiskLevelHigh, got.Level)
	}
	if len(got.Factors) != 2 {
		t.Fatalf("expected 2 factors, got %d: %+v", len(got.Factors), got.Factors)
	}
	if got.Factors[0].Details["factor"] != "new_device_fingerprint" {
		t.Errorf("factor[0] wrong: %+v", got.Factors[0])
	}
	if got.Factors[1].Details["factor"] != "new_ip" {
		t.Errorf("factor[1] wrong: %+v", got.Factors[1])
	}
}

func TestAssessLoginRisk_RapidIPChangeAllThreeFactors(t *testing.T) {
	// Prior session started 2 minutes ago from a different IP, and both
	// the current IP and fingerprint are new. All three factors fire:
	// 0.3 + 0.2 + 0.4 = 0.9 → critical.
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	repo := &stubSessionRepo{
		mostRecent: &models.AuthSessionDoc{
			UserUUID:  "u1",
			IPAddress: "10.0.0.5",
			CreatedAt: now.Add(-2 * time.Minute),
		},
		fpCount: 0,
		ipCount: 0,
	}
	scorer := newScorer(t, repo, now)
	got, _ := scorer.AssessLoginRisk(context.Background(), "u1", &models.SecurityContext{
		IPAddress:   "8.8.8.8",
		Fingerprint: "fp-new",
	})
	want := WeightNewDeviceFingerprint + WeightNewIP + WeightRapidIPChange
	if got.Score != want {
		t.Errorf("score: want %v, got %v", want, got.Score)
	}
	if got.Level != RiskLevelCritical {
		t.Errorf("level: want %q, got %q", RiskLevelCritical, got.Level)
	}
	if len(got.Factors) != 3 {
		t.Fatalf("expected 3 factors, got %d", len(got.Factors))
	}
	if got.Factors[2].Details["factor"] != "rapid_ip_change" {
		t.Errorf("factor[2] wrong: %+v", got.Factors[2])
	}
}

func TestAssessLoginRisk_RapidSameIPDoesNotFire(t *testing.T) {
	// Prior session 2m ago from the SAME IP → tab refresh / race. Not
	// a risk signal. Known fingerprint + IP → zero score.
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	repo := &stubSessionRepo{
		mostRecent: &models.AuthSessionDoc{
			UserUUID:  "u1",
			IPAddress: "8.8.8.8",
			CreatedAt: now.Add(-2 * time.Minute),
		},
		fpCount: 4,
		ipCount: 12,
	}
	scorer := newScorer(t, repo, now)
	got, _ := scorer.AssessLoginRisk(context.Background(), "u1", &models.SecurityContext{
		IPAddress:   "8.8.8.8",
		Fingerprint: "fp-known",
	})
	if got.Score != 0.0 {
		t.Errorf("same-IP rapid should not fire: got %+v", got)
	}
}

func TestAssessLoginRisk_RapidDifferentIPOutsideWindowDoesNotFire(t *testing.T) {
	// Prior session 15 minutes ago from a different IP → outside the
	// 5-minute rapid window. Only new_ip (0.2) fires because the IP is
	// new to the user.
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	repo := &stubSessionRepo{
		mostRecent: &models.AuthSessionDoc{
			UserUUID:  "u1",
			IPAddress: "10.0.0.5",
			CreatedAt: now.Add(-15 * time.Minute),
		},
		fpCount: 4, // known fingerprint
		ipCount: 0, // new IP
	}
	scorer := newScorer(t, repo, now)
	got, _ := scorer.AssessLoginRisk(context.Background(), "u1", &models.SecurityContext{
		IPAddress:   "8.8.8.8",
		Fingerprint: "fp-known",
	})
	if got.Score != WeightNewIP {
		t.Errorf("outside rapid window: want %v, got %v (factors=%+v)", WeightNewIP, got.Score, got.Factors)
	}
}

func TestAssessLoginRisk_EmptyFingerprintSkipsFactor(t *testing.T) {
	// Login path doesn't supply a fingerprint (e.g. early mobile client).
	// new_device_fingerprint cannot fire; new_ip still does.
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	repo := &stubSessionRepo{
		mostRecent: &models.AuthSessionDoc{
			UserUUID:  "u1",
			IPAddress: "10.0.0.5",
			CreatedAt: now.Add(-24 * time.Hour),
		},
		ipCount: 0,
	}
	scorer := newScorer(t, repo, now)
	got, _ := scorer.AssessLoginRisk(context.Background(), "u1", &models.SecurityContext{
		IPAddress: "8.8.8.8",
		// Fingerprint intentionally empty
	})
	if got.Score != WeightNewIP {
		t.Errorf("missing fingerprint: want %v, got %v", WeightNewIP, got.Score)
	}
}

func TestAssessLoginRisk_RepoErrorFallsBackToZero(t *testing.T) {
	// A Mongo error on the prior-session lookup must not block login.
	// The scorer logs a warn and returns zero score so the login path
	// continues with the pre-C1 default.
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	repo := &stubSessionRepo{mostErr: errors.New("mongo: timeout")}
	scorer := newScorer(t, repo, now)
	got, err := scorer.AssessLoginRisk(context.Background(), "u1", &models.SecurityContext{
		IPAddress: "8.8.8.8",
	})
	if err != nil {
		t.Fatalf("scorer must not propagate repo errors: %v", err)
	}
	if got.Score != 0.0 {
		t.Errorf("score on repo error: want 0.0, got %v", got.Score)
	}
}

func TestAssessLoginRisk_FactorCountErrorDoesNotAbort(t *testing.T) {
	// A mid-scoring repo error (fpErr / ipErr) should skip that factor
	// but continue evaluating the rest.
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	repo := &stubSessionRepo{
		mostRecent: &models.AuthSessionDoc{
			UserUUID:  "u1",
			IPAddress: "10.0.0.5",
			CreatedAt: now.Add(-24 * time.Hour),
		},
		fpErr:   errors.New("count failed"),
		ipCount: 0, // new IP still fires
	}
	scorer := newScorer(t, repo, now)
	got, _ := scorer.AssessLoginRisk(context.Background(), "u1", &models.SecurityContext{
		IPAddress:   "8.8.8.8",
		Fingerprint: "fp-something",
	})
	if got.Score != WeightNewIP {
		t.Errorf("partial failure: want new_ip only (%v), got %v", WeightNewIP, got.Score)
	}
}
