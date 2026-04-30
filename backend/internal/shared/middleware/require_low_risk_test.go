package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	authModels "github.com/orkestra/backend/internal/core/auth/models"
)

// runLowRisk wires a minimal AuthMiddleware with the given SessionRiskLookup,
// seeds the request context with the supplied claims, and invokes
// RequireLowRisk(threshold). Returns (downstreamRan, httpStatus, body).
func runLowRisk(t *testing.T, threshold float64, lookup SessionRiskLookup, claims *authModels.JWTClaims) (bool, int, map[string]any) {
	t.Helper()
	m := newTestMiddleware(&fakeAuthz{}, &fakeTenantProvider{}, nil)
	m.SetSessionRiskLookup(lookup)

	req := httptest.NewRequest(http.MethodPost, "/v1/admin/mutation", nil)
	if claims != nil {
		req = req.WithContext(context.WithValue(req.Context(), ctxClaims, claims))
	}
	rec := httptest.NewRecorder()

	called := false
	handler := m.RequireLowRisk(threshold)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	handler.ServeHTTP(rec, req)

	var body map[string]any
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
	}
	return called, rec.Code, body
}

func TestRequireLowRisk_BelowThresholdPasses(t *testing.T) {
	claims := &authModels.JWTClaims{UserUUID: "u", SessionID: "s-1"}
	lookup := SessionRiskLookup(func(_ context.Context, _ string) (float64, error) { return 0.2, nil })
	called, status, _ := runLowRisk(t, 0.5, lookup, claims)
	if !called {
		t.Errorf("below threshold must pass through; downstream not called (status %d)", status)
	}
}

func TestRequireLowRisk_AtThresholdBlocks(t *testing.T) {
	// Exactly at the threshold should block — strict >= semantics.
	claims := &authModels.JWTClaims{UserUUID: "u", SessionID: "s-1"}
	lookup := SessionRiskLookup(func(_ context.Context, _ string) (float64, error) { return 0.5, nil })
	called, status, body := runLowRisk(t, 0.5, lookup, claims)
	if called {
		t.Error("score at threshold must block downstream")
	}
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", status)
	}
	if code, _ := body["code"].(string); code != "step_up_required" {
		t.Errorf("body.code = %q, want step_up_required", code)
	}
	if rs, _ := body["riskScore"].(float64); rs != 0.5 {
		t.Errorf("body.riskScore = %v, want 0.5", rs)
	}
	if rt, _ := body["riskThreshold"].(float64); rt != 0.5 {
		t.Errorf("body.riskThreshold = %v, want 0.5", rt)
	}
}

func TestRequireLowRisk_AboveThresholdBlocks(t *testing.T) {
	claims := &authModels.JWTClaims{UserUUID: "u", SessionID: "s-1"}
	lookup := SessionRiskLookup(func(_ context.Context, _ string) (float64, error) { return 0.9, nil })
	called, status, _ := runLowRisk(t, 0.5, lookup, claims)
	if called {
		t.Error("score above threshold must block downstream")
	}
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", status)
	}
}

func TestRequireLowRisk_LookupErrorFailsOpen(t *testing.T) {
	// A Redis/Mongo blip must not lock privileged actions out. The
	// middleware emits a Warn log (not asserted here) and passes the
	// request through.
	claims := &authModels.JWTClaims{UserUUID: "u", SessionID: "s-1"}
	lookup := SessionRiskLookup(func(_ context.Context, _ string) (float64, error) {
		return 0, errors.New("boom")
	})
	called, _, _ := runLowRisk(t, 0.5, lookup, claims)
	if !called {
		t.Error("lookup error must fail open (pass through)")
	}
}

func TestRequireLowRisk_NoLookupFailsOpen(t *testing.T) {
	// Deploys that haven't wired the lookup yet (tests, minimal stack)
	// should just pass through — the gate is additive, not a default-deny.
	claims := &authModels.JWTClaims{UserUUID: "u", SessionID: "s-1"}
	called, _, _ := runLowRisk(t, 0.5, nil, claims)
	if !called {
		t.Error("nil lookup must fail open")
	}
}

func TestRequireLowRisk_NoClaimsFailsOpen(t *testing.T) {
	// No JWT → no session → no score to evaluate. Upstream auth
	// middleware is responsible for authentication; RequireLowRisk
	// augments it and must not itself block unauthenticated calls
	// (they'd already be blocked elsewhere).
	lookup := SessionRiskLookup(func(_ context.Context, _ string) (float64, error) { return 1.0, nil })
	called, _, _ := runLowRisk(t, 0.5, lookup, nil)
	if !called {
		t.Error("missing claims must fail open — upstream auth already gates")
	}
}

func TestRequireLowRisk_NoSessionIDFailsOpen(t *testing.T) {
	// Claims without a SessionID (legacy tokens pre-sid stamping). No
	// session to score → pass through.
	claims := &authModels.JWTClaims{UserUUID: "u"}
	lookup := SessionRiskLookup(func(_ context.Context, _ string) (float64, error) { return 1.0, nil })
	called, _, _ := runLowRisk(t, 0.5, lookup, claims)
	if !called {
		t.Error("missing sid must fail open")
	}
}
