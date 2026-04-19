// Mock OIDC harness for tests.
//
// Exposes a tiny in-process OIDC provider that signs ID tokens with a
// generated RSA key. Wires three endpoints mandatory for a coreos/go-oidc
// client to complete a flow:
//
//   - GET  /.well-known/openid-configuration — discovery document
//   - GET  /jwks                              — signing keys (RS256)
//   - POST /token                             — code → id_token exchange
//
// The `/authorize` step is not exposed — tests short-circuit it by
// invoking the mock's MintIDToken helper directly and treating the
// returned code as whatever the test hands in at /token. This keeps the
// harness small while still covering the crypto-critical path (discovery
// + verify + nonce + audience).
//
// Usage:
//
//	mock := services.StartMockOIDC(t, services.MockOIDCOptions{
//	    ClientID: "orkestra-test",
//	    Subject:  "user-123",
//	    Email:    "alice@example.com",
//	    Name:     "Alice",
//	})
//	defer mock.Close()
//	// point the identity Service at mock.IssuerURL
//
// Not wired into a _test.go file so it's usable from integration tests in
// sibling packages. Do not instantiate from production code.
package services

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// MockOIDCOptions configures the claims and audience a mock ID token
// will carry. All fields have sensible defaults so tests that only care
// about the happy path can pass `MockOIDCOptions{ClientID: "..."}`.
type MockOIDCOptions struct {
	// ClientID is the OAuth client ID registered with this mock IdP. The
	// ID token's `aud` claim is set to this value, so `oidc.Verifier`
	// accepts it when configured for the same ClientID. Required.
	ClientID string
	// Subject populates the `sub` claim. Defaults to "mock-user".
	Subject string
	// Email populates the `email` claim. Defaults to "mock@example.com".
	Email string
	// Name populates the `name` claim. Defaults to "Mock User".
	Name string
	// TokenTTL is how long the issued ID token remains valid. Defaults
	// to 10 minutes.
	TokenTTL time.Duration
}

// MockOIDC is the live handle returned by StartMockOIDC. Call Close() to
// shut the underlying httptest server down.
type MockOIDC struct {
	IssuerURL string
	server    *httptest.Server
	privKey   *rsa.PrivateKey
	opts      MockOIDCOptions
	// pendingNonce is set by MintIDToken so /token can echo it back on the
	// next exchange. Single-request-at-a-time by design — tests should not
	// race two concurrent flows through the same mock without explicit
	// synchronization.
	pendingNonce string
}

// Close shuts the harness down. Safe to call multiple times.
func (m *MockOIDC) Close() {
	if m.server != nil {
		m.server.Close()
		m.server = nil
	}
}

// MintIDToken prepares the harness to return a signed ID token with the
// given nonce on the next /token POST. Call it right before triggering a
// callback in your test so the sequence is:
//
//	mock.MintIDToken("nonce-from-start")
//	svc.Callback(ctx, CallbackInput{Code: "any", State: stateToken})
func (m *MockOIDC) MintIDToken(nonce string) {
	m.pendingNonce = nonce
}

// StartMockOIDC boots an in-process mock provider and returns a handle.
// Test callers typically defer mock.Close().
func StartMockOIDC(opts MockOIDCOptions) (*MockOIDC, error) {
	if opts.ClientID == "" {
		return nil, fmt.Errorf("mock OIDC: ClientID required")
	}
	if opts.Subject == "" {
		opts.Subject = "mock-user"
	}
	if opts.Email == "" {
		opts.Email = "mock@example.com"
	}
	if opts.Name == "" {
		opts.Name = "Mock User"
	}
	if opts.TokenTTL == 0 {
		opts.TokenTTL = 10 * time.Minute
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("mock OIDC: keygen: %w", err)
	}

	m := &MockOIDC{privKey: priv, opts: opts}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", m.handleDiscovery)
	mux.HandleFunc("/jwks", m.handleJWKS)
	mux.HandleFunc("/token", m.handleToken)
	m.server = httptest.NewServer(mux)
	m.IssuerURL = m.server.URL
	return m, nil
}

func (m *MockOIDC) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	doc := map[string]any{
		"issuer":                                m.IssuerURL,
		"authorization_endpoint":                m.IssuerURL + "/authorize",
		"token_endpoint":                        m.IssuerURL + "/token",
		"jwks_uri":                              m.IssuerURL + "/jwks",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "email", "profile"},
		"claims_supported":                      []string{"sub", "email", "name", "aud", "iss", "nonce"},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(doc)
}

func (m *MockOIDC) handleJWKS(w http.ResponseWriter, r *http.Request) {
	n := base64.RawURLEncoding.EncodeToString(m.privKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(m.privKey.E)).Bytes())
	jwks := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": "mock-key-1",
				"n":   n,
				"e":   e,
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jwks)
}

// handleToken returns an ID token whose `aud` matches the configured
// ClientID and whose `nonce` matches whatever MintIDToken stashed last.
// The `code` in the request body is ignored — the mock trusts the caller.
func (m *MockOIDC) handleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "parse form", http.StatusBadRequest)
		return
	}
	if r.PostForm.Get("code") == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	claims := jwt.MapClaims{
		"iss":   m.IssuerURL,
		"sub":   m.opts.Subject,
		"aud":   m.opts.ClientID,
		"exp":   time.Now().Add(m.opts.TokenTTL).Unix(),
		"iat":   time.Now().Unix(),
		"email": m.opts.Email,
		"name":  m.opts.Name,
	}
	if m.pendingNonce != "" {
		claims["nonce"] = m.pendingNonce
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = "mock-key-1"
	signed, err := tok.SignedString(m.privKey)
	if err != nil {
		http.Error(w, "sign: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]any{
		"access_token": "mock-access",
		"token_type":   "Bearer",
		"expires_in":   3600,
		"id_token":     signed,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// consumeCodeVerifier is a no-op placeholder — tests that want PKCE can
// add verification here. Kept to keep the handler signatures symmetric.
//
//nolint:unused // reserved for future PKCE tests
func (m *MockOIDC) consumeCodeVerifier(verifier string) bool {
	return !strings.ContainsAny(verifier, " \t\n")
}
