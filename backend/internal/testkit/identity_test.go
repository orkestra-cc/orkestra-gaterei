package testkit_test

import (
	"context"
	"testing"

	"github.com/orkestra/backend/internal/core/auth/models"
	"github.com/orkestra/backend/internal/shared/middleware"
	"github.com/orkestra/backend/internal/shared/tenantrepo"
	"github.com/orkestra/backend/internal/testkit"
)

// TestContextKeysRoundTrip is the load-bearing test for this package: it
// asserts that a context populated by testkit is readable by the exact public
// accessors production code uses. If anyone renames a context key on either
// side, this test fails immediately instead of every org-scoped integration
// test silently becoming a no-op.
func TestContextKeysRoundTrip(t *testing.T) {
	id := testkit.NewIdentity("user-1", "alice@example.com", "administrator").
		WithOrg("org-a", []string{"org_admin"}, true).
		WithOrg("org-b", []string{"org_member"}, false)

	ctx := id.ContextFor(context.Background(), "org-b")

	if got, ok := middleware.GetUserUUID(ctx); !ok || got != "user-1" {
		t.Errorf("GetUserUUID: got (%q, %v), want (\"user-1\", true)", got, ok)
	}
	if got, ok := middleware.GetUserEmail(ctx); !ok || got != "alice@example.com" {
		t.Errorf("GetUserEmail: got (%q, %v)", got, ok)
	}
	if got, ok := middleware.GetSystemRole(ctx); !ok || got != "administrator" {
		t.Errorf("GetSystemRole: got (%q, %v)", got, ok)
	}
	if got, ok := middleware.GetOrgID(ctx); !ok || got != "org-b" {
		t.Errorf("GetOrgID: got (%q, %v), want (\"org-b\", true)", got, ok)
	}
	roles, ok := middleware.GetOrgRoles(ctx)
	if !ok || len(roles) != 1 || roles[0] != "org_member" {
		t.Errorf("GetOrgRoles: got (%v, %v), want ([org_member], true)", roles, ok)
	}
	mbrs, ok := middleware.GetMemberships(ctx)
	if !ok || len(mbrs) != 2 {
		t.Errorf("GetMemberships: got %d memberships, want 2", len(mbrs))
	}
}

// TestDefaultOrgFallback: when activeOrgID is empty, the default org takes
// over. This mirrors the JWT's `dorg` claim being used when the X-Org-ID
// header is absent.
func TestDefaultOrgFallback(t *testing.T) {
	id := testkit.NewIdentity("user-1", "a@b", "manager").
		WithOrg("org-a", []string{"org_owner"}, true)
	ctx := id.ContextFor(context.Background(), "")
	got, ok := middleware.GetOrgID(ctx)
	if !ok || got != "org-a" {
		t.Fatalf("expected default org-a, got (%q, %v)", got, ok)
	}
}

// TestGlobalSentinel: passing "-" leaves the context org-less, matching
// RequireGlobal routes.
func TestGlobalSentinel(t *testing.T) {
	id := testkit.NewIdentity("user-1", "a@b", "super_admin").
		WithOrg("org-a", []string{"org_owner"}, true)
	ctx := id.ContextFor(context.Background(), "-")
	if got, ok := middleware.GetOrgID(ctx); ok {
		t.Fatalf("expected no orgID for global sentinel, got %q", got)
	}
}

// TestInvalidActiveOrgPanics: passing an org the identity isn't a member of
// panics — the real middleware returns 403 in this case; a test that
// configures it is asserting a reality that can't happen in production.
func TestInvalidActiveOrgPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for cross-tenant activeOrgID, got none")
		}
	}()
	id := testkit.NewIdentity("user-1", "a@b", "manager").
		WithOrg("org-a", []string{"org_member"}, true)
	id.ContextFor(context.Background(), "org-evil")
}

// TestClaimsSnapshot: the Claims helper returns a populated *JWTClaims that
// matches the identity. Useful for tests that want to assert on claim fields
// directly rather than through the context.
func TestClaimsSnapshot(t *testing.T) {
	id := testkit.NewIdentity("user-1", "e@x", "developer").
		WithOrg("org-1", []string{"org_admin"}, true)
	claims := id.Claims()
	if claims.UserUUID != "user-1" || claims.SystemRole != "developer" {
		t.Errorf("identity mismatch: %+v", claims)
	}
	if claims.DefaultOrgID != "org-1" {
		t.Errorf("expected default org-1, got %q", claims.DefaultOrgID)
	}
	if len(claims.Memberships) != 1 || claims.Memberships[0].OrgUUID != "org-1" {
		t.Errorf("expected 1 membership for org-1, got %+v", claims.Memberships)
	}
}

// TestTenantRepoInterop: confirm tenantrepo.Scope can see the injected orgID
// — the primary consumer of this testkit. If this test breaks, every
// downstream integration test that uses tenantrepo is also broken.
func TestTenantRepoInterop(t *testing.T) {
	id := testkit.NewIdentity("user-1", "a@b", "manager").
		WithOrg("org-7", []string{"org_member"}, true)
	ctx := id.ContextFor(context.Background(), "")

	got, ok := tenantrepo.CurrentOrgID(ctx)
	if !ok || got != "org-7" {
		t.Fatalf("tenantrepo.CurrentOrgID: got (%q, %v), want (\"org-7\", true)", got, ok)
	}
}

// Compile-time assurance the OrgMembership type we use matches the one the
// auth middleware actually reads — if the two diverge (e.g. auth adds a
// required field), the compiler catches it here.
var _ []models.OrgMembership = (&models.JWTClaims{}).Memberships
