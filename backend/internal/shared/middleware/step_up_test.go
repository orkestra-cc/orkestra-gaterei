package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authModels "github.com/orkestra/backend/internal/core/auth/models"
)

// runStepUp wires a minimal AuthMiddleware, seeds the request context with
// the given claims, and invokes RequireStepUp(maxAge). Returns
// (downstreamRan, httpStatus, body) so the tests stay terse.
func runStepUp(t *testing.T, maxAge time.Duration, claims *authModels.JWTClaims) (bool, int, map[string]any) {
	t.Helper()
	m := newTestMiddleware(&fakeAuthz{}, &fakeTenantProvider{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/admin/anything", nil)
	if claims != nil {
		req = req.WithContext(context.WithValue(req.Context(), ctxClaims, claims))
	}
	rec := httptest.NewRecorder()

	called := false
	handler := m.RequireStepUp(maxAge)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	handler.ServeHTTP(rec, req)

	var body map[string]any
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
	}
	return called, rec.Code, body
}

func TestRequireStepUp_FreshMFAProofPasses(t *testing.T) {
	claims := &authModels.JWTClaims{
		UserUUID:  "u-1",
		AMR:       []string{"pwd", "otp"},
		LastOTPAt: time.Now().Add(-30 * time.Second).Unix(),
	}
	called, status, _ := runStepUp(t, 5*time.Minute, claims)
	if !called {
		t.Errorf("fresh MFA must pass through; downstream not called (status %d)", status)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
}

func TestRequireStepUp_StaleMFAProofRejected(t *testing.T) {
	// last_otp_at older than maxAge → step up.
	claims := &authModels.JWTClaims{
		UserUUID:  "u-1",
		AMR:       []string{"pwd", "otp"},
		LastOTPAt: time.Now().Add(-10 * time.Minute).Unix(),
	}
	called, status, body := runStepUp(t, 5*time.Minute, claims)
	if called {
		t.Error("stale MFA must block downstream")
	}
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", status)
	}
	if code, _ := body["code"].(string); code != "step_up_required" {
		t.Errorf("body.code = %q, want step_up_required", code)
	}
}

func TestRequireStepUp_MissingAMRRejected(t *testing.T) {
	// amr without MFA marker → step up required even if LastOTPAt is set.
	claims := &authModels.JWTClaims{
		UserUUID:  "u-1",
		AMR:       []string{"pwd"},
		LastOTPAt: time.Now().Unix(),
	}
	called, status, body := runStepUp(t, 5*time.Minute, claims)
	if called {
		t.Error("non-MFA amr must block downstream")
	}
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", status)
	}
	if code, _ := body["code"].(string); code != "step_up_required" {
		t.Errorf("body.code = %q, want step_up_required", code)
	}
}

func TestRequireStepUp_MissingLastOTPAtRejected(t *testing.T) {
	// amr has otp but LastOTPAt is zero — we can't confirm freshness so
	// the middleware must reject. Pre-Block-A tokens land here.
	claims := &authModels.JWTClaims{
		UserUUID: "u-1",
		AMR:      []string{"pwd", "otp"},
	}
	called, status, _ := runStepUp(t, 5*time.Minute, claims)
	if called {
		t.Error("zero LastOTPAt must block downstream")
	}
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", status)
	}
}

func TestRequireStepUp_NoClaimsRejectedAsUnauth(t *testing.T) {
	called, status, _ := runStepUp(t, 5*time.Minute, nil)
	if called {
		t.Error("missing claims must block downstream")
	}
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", status)
	}
}

func TestRequireStepUp_DefaultMaxAgeWhenZero(t *testing.T) {
	// Zero maxAge defaults to 5min. A 2-minute-old OTP proof must pass.
	claims := &authModels.JWTClaims{
		UserUUID:  "u-1",
		AMR:       []string{"pwd", "otp"},
		LastOTPAt: time.Now().Add(-2 * time.Minute).Unix(),
	}
	called, status, _ := runStepUp(t, 0, claims)
	if !called {
		t.Errorf("2-min-old proof under default 5min window must pass; status %d", status)
	}
}
