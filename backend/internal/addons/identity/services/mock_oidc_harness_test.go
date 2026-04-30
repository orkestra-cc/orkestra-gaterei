package services

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// TestMockOIDCHarness_FullVerify proves the mock provider is compatible
// with coreos/go-oidc end-to-end: discovery works, the JWKS is fetchable,
// and an ID token signed by the mock verifies with the right audience +
// nonce. This is the one piece of the Phase-3.3 flow we can exercise
// without a live MongoDB / Redis rig.
func TestMockOIDCHarness_FullVerify(t *testing.T) {
	t.Parallel()

	mock, err := StartMockOIDC(MockOIDCOptions{
		ClientID: "orkestra-test",
		Subject:  "user-42",
		Email:    "alice@example.com",
		Name:     "Alice",
		TokenTTL: time.Minute,
	})
	if err != nil {
		t.Fatalf("StartMockOIDC: %v", err)
	}
	defer mock.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, mock.IssuerURL)
	if err != nil {
		t.Fatalf("oidc.NewProvider: %v", err)
	}

	nonce := "nonce-abc"
	mock.MintIDToken(nonce)

	// Hit /token the way an oauth2 client would — mock ignores the code.
	resp, err := http.PostForm(mock.IssuerURL+"/token",
		url.Values{"code": {"any-code"}, "grant_type": {"authorization_code"}})
	if err != nil {
		t.Fatalf("POST /token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mock /token status: %d", resp.StatusCode)
	}

	// Rather than parse the JSON blob by hand, pull the id_token out via a
	// minimal decoder — keeps the test focused on what the harness
	// promises, not on JSON plumbing.
	body, err := readAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	rawIDToken := extractIDToken(string(body))
	if rawIDToken == "" {
		t.Fatalf("response did not contain id_token: %s", body)
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: "orkestra-test"})
	idTok, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if idTok.Nonce != nonce {
		t.Fatalf("nonce: want %q, got %q", nonce, idTok.Nonce)
	}

	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Sub   string `json:"sub"`
	}
	if err := idTok.Claims(&claims); err != nil {
		t.Fatalf("claims: %v", err)
	}
	if claims.Email != "alice@example.com" || claims.Name != "Alice" || claims.Sub != "user-42" {
		t.Fatalf("claims: got %+v", claims)
	}
}

// readAll is a shim for io.ReadAll that avoids an extra import. Tests
// shouldn't reach for io directly — keep dependencies minimal.
func readAll(r interface {
	Read(p []byte) (int, error)
}) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			out = append(out, buf[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				return out, nil
			}
			return out, err
		}
	}
}

// extractIDToken pulls "id_token":"..." from a JSON blob without
// unmarshalling — keeps the test light. Returns "" if not found.
func extractIDToken(jsonBlob string) string {
	const needle = `"id_token":"`
	i := strings.Index(jsonBlob, needle)
	if i < 0 {
		return ""
	}
	rest := jsonBlob[i+len(needle):]
	end := strings.IndexByte(rest, '"')
	if end < 0 {
		return ""
	}
	return rest[:end]
}
