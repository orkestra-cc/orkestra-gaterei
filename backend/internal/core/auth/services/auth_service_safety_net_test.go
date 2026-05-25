package services

// Safety-net tests pinning the *current* behaviour of two methods that the
// upcoming refactor will mutate:
//
//   - GetOAuthLinks: hardcodes RequiresMFA=false ignoring real MFA state —
//     Phase 1.3 computes it from MFAFactorRepo.
//   - RecordSecurityEvent / GetSecurityEvents: no-op + empty list —
//     Phase 2.1 replaces with real persistence.
//
// When those phases land, the failing tests in this file are the signal to
// delete or update — they exist precisely to make sure the refactor does
// what we think it does.
//
// The AddOAuthLink stub had its sentinel test removed alongside the stub
// itself in Phase 1.2 — link creation goes through the OAuth flow exposed
// by self_user_auth_handler, not a service-level entry point.

import (
	"context"
	"testing"
	"time"

	"github.com/orkestra/backend/internal/core/auth/models"
	userModels "github.com/orkestra/backend/internal/core/user/models"
)

// TestGetOAuthLinks_RequiresMFAFalseWithoutFactor: a user with no enrolled
// factor (nor repo wired) reports RequiresMFA=false so the unlink path
// surfaces the password-reconfirm modal instead of step-up.
func TestGetOAuthLinks_RequiresMFAFalseWithoutFactor(t *testing.T) {
	t.Parallel()
	fake := newAdminUnlinkUserFake()
	fake.seed(&userModels.User{
		UUID:         "u-multi",
		Role:         "administrator",
		PasswordHash: "x",
		OAuthLinks: []userModels.OAuthLink{
			{Provider: "google", ProviderID: "g-1", Email: "u@x.com", IsActive: true, IsPrimary: true},
			{Provider: "github", ProviderID: "gh-1", Email: "u@x.com", IsActive: true},
		},
	})
	// mfaFactorRepo intentionally nil — emulates the test wiring before
	// MFA was bolted on. RequiresMFA must stay false in this degraded mode.
	svc := &authService{userService: fake}

	resp, err := svc.GetOAuthLinks(context.Background(), "u-multi")
	if err != nil {
		t.Fatalf("GetOAuthLinks: %v", err)
	}
	if len(resp.Links) != 2 || !resp.CanUnlink {
		t.Errorf("Links/CanUnlink mismatch: %+v", resp)
	}
	if resp.RequiresMFA {
		t.Errorf("RequiresMFA must be false when no MFA factor is wired")
	}
}

// TestGetOAuthLinks_RequiresMFATrueWithEnrolledTOTP: a user with a TOTP
// factor surfaces RequiresMFA=true so the SPA routes the unlink through
// the step-up modal, not the password-reconfirm one.
func TestGetOAuthLinks_RequiresMFATrueWithEnrolledTOTP(t *testing.T) {
	t.Parallel()
	fake := newAdminUnlinkUserFake()
	fake.seed(&userModels.User{
		UUID:         "u-totp",
		Role:         "administrator",
		PasswordHash: "x",
		OAuthLinks: []userModels.OAuthLink{
			{Provider: "google", ProviderID: "g-1", Email: "u@x.com", IsActive: true},
		},
	})
	enrolled := time.Now().Add(-7 * 24 * time.Hour)
	factors := newFakeFactorRepo()
	_ = factors.Insert(context.Background(), &models.MFAFactorDoc{
		UUID:       "fact-1",
		UserUUID:   "u-totp",
		Type:       models.MFAFactorTOTP,
		VerifiedAt: &enrolled,
	})
	svc := &authService{userService: fake, mfaFactorRepo: factors}

	resp, err := svc.GetOAuthLinks(context.Background(), "u-totp")
	if err != nil {
		t.Fatalf("GetOAuthLinks: %v", err)
	}
	if !resp.RequiresMFA {
		t.Errorf("user with TOTP factor must report RequiresMFA=true")
	}
}

// TestGetOAuthLinks_RequiresMFATrueWithWebAuthn: ditto via the embedded
// webauthnCredentials array on the WebAuthn factor row.
func TestGetOAuthLinks_RequiresMFATrueWithWebAuthn(t *testing.T) {
	t.Parallel()
	fake := newAdminUnlinkUserFake()
	fake.seed(&userModels.User{
		UUID:         "u-wa",
		Role:         "operator",
		PasswordHash: "x",
		OAuthLinks: []userModels.OAuthLink{
			{Provider: "google", ProviderID: "g-1", Email: "u@x.com", IsActive: true},
		},
	})
	factors := newFakeFactorRepo()
	_ = factors.Insert(context.Background(), &models.MFAFactorDoc{
		UUID:     "fact-wa",
		UserUUID: "u-wa",
		Type:     models.MFAFactorWebAuthn,
		WebAuthnCredentials: []models.WebAuthnCredential{
			{CredentialID: []byte{0x01, 0x02}, Name: "Touch ID", CreatedAt: time.Now()},
		},
	})
	svc := &authService{userService: fake, mfaFactorRepo: factors}

	resp, _ := svc.GetOAuthLinks(context.Background(), "u-wa")
	if !resp.RequiresMFA {
		t.Errorf("user with WebAuthn credential must report RequiresMFA=true")
	}
}

// TestGetOAuthLinks_RequiresMFAFalseWithEmptyWebAuthnArray: a WebAuthn factor
// row that exists but carries zero credentials must NOT count — the embedded
// array is the source of truth for "is a passkey enrolled".
func TestGetOAuthLinks_RequiresMFAFalseWithEmptyWebAuthnArray(t *testing.T) {
	t.Parallel()
	fake := newAdminUnlinkUserFake()
	fake.seed(&userModels.User{
		UUID:         "u-wa-empty",
		Role:         "operator",
		PasswordHash: "x",
		OAuthLinks: []userModels.OAuthLink{
			{Provider: "google", ProviderID: "g-1", IsActive: true},
		},
	})
	factors := newFakeFactorRepo()
	_ = factors.Insert(context.Background(), &models.MFAFactorDoc{
		UUID:                "fact-wa-empty",
		UserUUID:            "u-wa-empty",
		Type:                models.MFAFactorWebAuthn,
		WebAuthnCredentials: nil,
	})
	svc := &authService{userService: fake, mfaFactorRepo: factors}

	resp, _ := svc.GetOAuthLinks(context.Background(), "u-wa-empty")
	if resp.RequiresMFA {
		t.Errorf("WebAuthn factor row with 0 credentials must not flip RequiresMFA")
	}
}

// TestGetOAuthLinks_SingleLinkCannotUnlink: with only one link, CanUnlink is
// false. This is the only piece of GetOAuthLinks' logic that Phase 1.3 does
// NOT touch.
func TestGetOAuthLinks_SingleLinkCannotUnlink(t *testing.T) {
	t.Parallel()
	fake := newAdminUnlinkUserFake()
	fake.seed(&userModels.User{
		UUID: "u-sole",
		OAuthLinks: []userModels.OAuthLink{
			{Provider: "google", ProviderID: "g-1", Email: "u@x.com", IsActive: true, IsPrimary: true},
		},
	})
	svc := &authService{userService: fake}

	resp, err := svc.GetOAuthLinks(context.Background(), "u-sole")
	if err != nil {
		t.Fatalf("GetOAuthLinks: %v", err)
	}
	if resp.CanUnlink {
		t.Errorf("CanUnlink with a single link should be false")
	}
}

// TestGetOAuthLinks_PropagatesUserServiceError: a missing user surfaces as an
// error from the user provider; GetOAuthLinks must not swallow it. The fake
// returns errNotFound when no user is seeded, which is the closest off-the-
// shelf failure mode without growing the shared helper.
func TestGetOAuthLinks_PropagatesUserServiceError(t *testing.T) {
	t.Parallel()
	svc := &authService{userService: newAdminUnlinkUserFake()}

	_, err := svc.GetOAuthLinks(context.Background(), "no-such-user")
	if err == nil {
		t.Fatal("expected error from underlying user provider")
	}
}

// TestRecordSecurityEvent_CurrentlyNoOpReturnsNil locks the current placeholder.
// Phase 2.1 implements real persistence; this test gets rewritten to assert
// the event landed in auth_security_events.
func TestRecordSecurityEvent_CurrentlyNoOpReturnsNil(t *testing.T) {
	t.Parallel()
	svc := &authService{}

	err := svc.RecordSecurityEvent(context.Background(), &models.SecurityEvent{
		ID:        "ev-1",
		UserUUID:  "u-1",
		EventType: "test_event",
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Errorf("placeholder must return nil, got %v", err)
	}
}

// TestGetSecurityEvents_CurrentlyReturnsEmptySlice — same as above, the
// reader-side companion that Phase 2.3 replaces with a Mongo query.
func TestGetSecurityEvents_CurrentlyReturnsEmptySlice(t *testing.T) {
	t.Parallel()
	svc := &authService{}

	events, err := svc.GetSecurityEvents(context.Background(), "u-1", 100)
	if err != nil {
		t.Errorf("placeholder must return nil error, got %v", err)
	}
	if len(events) != 0 {
		t.Errorf("placeholder must return an empty slice, got %d entries", len(events))
	}
}
