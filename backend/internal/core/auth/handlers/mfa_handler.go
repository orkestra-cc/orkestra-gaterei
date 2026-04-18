package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	authModels "github.com/orkestra/backend/internal/core/auth/models"
	"github.com/orkestra/backend/internal/core/auth/services"
	"github.com/orkestra/backend/internal/shared/iface"
)

// MFAHandler binds the MFA service to its HTTP surface. All endpoints live
// under /v1/auth/mfa or /v1/auth/me/mfa and require an authenticated user
// (no org context needed, so RequireGlobal() is the correct gate).
type MFAHandler struct {
	mfa          services.MFAService
	jwt          services.JWTService
	users        iface.UserProvider
	cookieName   string
	cookieDomain string
	cookieSecure bool
}

// NewMFAHandler wires the dependencies. Cookie config is needed by the
// /v1/auth/mfa/verify endpoint which issues a refreshed token pair.
func NewMFAHandler(
	mfa services.MFAService,
	jwt services.JWTService,
	users iface.UserProvider,
	cookieName, cookieDomain string,
	cookieSecure bool,
) *MFAHandler {
	if cookieName == "" {
		cookieName = "access_token"
	}
	return &MFAHandler{
		mfa:          mfa,
		jwt:          jwt,
		users:        users,
		cookieName:   cookieName,
		cookieDomain: cookieDomain,
		cookieSecure: cookieSecure,
	}
}

// --- Enrollment ---

type MFAEnrollBeginResponse struct {
	Body struct {
		ChallengeID     string `json:"challengeId"`
		Secret          string `json:"secret"`
		ProvisioningURI string `json:"provisioningUri"`
	}
}

func (h *MFAHandler) EnrollBegin(ctx context.Context, _ *struct{}) (*MFAEnrollBeginResponse, error) {
	userUUID, _ := ctx.Value("userUUID").(string)
	if userUUID == "" {
		return nil, huma.Error401Unauthorized("authentication required")
	}
	user, err := h.users.GetUserByID(ctx, userUUID)
	if err != nil || user == nil {
		return nil, huma.Error401Unauthorized("user not found")
	}
	begin, err := h.mfa.BeginEnrollment(ctx, user)
	if err != nil {
		return nil, mapMFAError(err)
	}
	resp := &MFAEnrollBeginResponse{}
	resp.Body.ChallengeID = begin.ChallengeID
	resp.Body.Secret = begin.SecretBase32
	resp.Body.ProvisioningURI = begin.ProvisioningURI
	return resp, nil
}

type MFAEnrollConfirmRequest struct {
	Body struct {
		ChallengeID string `json:"challengeId" doc:"Challenge ID returned by /enroll/begin"`
		Code        string `json:"code" doc:"6-digit TOTP code from the authenticator app"`
	}
}

type MFAEnrollConfirmResponse struct {
	Body struct {
		Success     bool     `json:"success"`
		BackupCodes []string `json:"backupCodes"`
	}
}

func (h *MFAHandler) EnrollConfirm(ctx context.Context, req *MFAEnrollConfirmRequest) (*MFAEnrollConfirmResponse, error) {
	userUUID, _ := ctx.Value("userUUID").(string)
	if userUUID == "" {
		return nil, huma.Error401Unauthorized("authentication required")
	}
	codes, err := h.mfa.ConfirmEnrollment(ctx, userUUID, req.Body.ChallengeID, req.Body.Code)
	if err != nil {
		return nil, mapMFAError(err)
	}
	resp := &MFAEnrollConfirmResponse{}
	resp.Body.Success = true
	resp.Body.BackupCodes = codes
	return resp, nil
}

// --- Status ---

type MFAStatusResponse struct {
	Body struct {
		Status               string `json:"status"`
		Type                 string `json:"type,omitempty"`
		BackupCodesRemaining int    `json:"backupCodesRemaining"`
	}
}

func (h *MFAHandler) Status(ctx context.Context, _ *struct{}) (*MFAStatusResponse, error) {
	userUUID, _ := ctx.Value("userUUID").(string)
	if userUUID == "" {
		return nil, huma.Error401Unauthorized("authentication required")
	}
	snap, err := h.mfa.Status(ctx, userUUID)
	if err != nil {
		return nil, mapMFAError(err)
	}
	resp := &MFAStatusResponse{}
	resp.Body.Status = string(snap.Status)
	resp.Body.Type = string(snap.Type)
	resp.Body.BackupCodesRemaining = snap.BackupCodesRemaining
	return resp, nil
}

// --- Remove ---

type MFARemoveRequest struct {
	Body struct {
		Code string `json:"code" doc:"Live TOTP code — confirms the removal is intentional"`
	}
}

type MFARemoveResponse struct {
	Body struct {
		Success bool `json:"success"`
	}
}

func (h *MFAHandler) Remove(ctx context.Context, req *MFARemoveRequest) (*MFARemoveResponse, error) {
	userUUID, _ := ctx.Value("userUUID").(string)
	if userUUID == "" {
		return nil, huma.Error401Unauthorized("authentication required")
	}
	// Block D will replace this in-handler check with RequireStepUp middleware.
	if err := h.mfa.Verify(ctx, userUUID, req.Body.Code); err != nil {
		return nil, mapMFAError(err)
	}
	if err := h.mfa.RemoveFactor(ctx, userUUID, userUUID); err != nil {
		return nil, mapMFAError(err)
	}
	resp := &MFARemoveResponse{}
	resp.Body.Success = true
	return resp, nil
}

// --- Verify (self-service step-up) ---

type MFAVerifyRequest struct {
	Body struct {
		ChallengeID string `json:"challengeId,omitempty" doc:"Optional — reserved for Block B login flow"`
		Code        string `json:"code" doc:"6-digit TOTP code or a backup code"`
		UseBackup   bool   `json:"useBackup,omitempty" doc:"Set true to consume a backup code instead of TOTP"`
	}
}

type MFAVerifyResponse struct {
	SetCookie string `header:"Set-Cookie"`
	Body      struct {
		Success     bool   `json:"success"`
		AccessToken string `json:"accessToken"`
		TokenType   string `json:"tokenType"`
		ExpiresIn   int64  `json:"expiresIn"`
	}
}

// Verify mints a new access token annotated with amr:["pwd","otp"] (or
// ["oauth","otp"]) and last_otp_at=now. Block A only supports the
// self-service path where the caller already has a valid "pwd" or "oauth"
// token; the Block B login path will supply a challengeId tied to a
// partially-authenticated session.
func (h *MFAHandler) Verify(ctx context.Context, req *MFAVerifyRequest) (*MFAVerifyResponse, error) {
	userUUID, _ := ctx.Value("userUUID").(string)
	if userUUID == "" {
		return nil, huma.Error401Unauthorized("authentication required")
	}

	if req.Body.UseBackup {
		if err := h.mfa.VerifyBackupCode(ctx, userUUID, req.Body.Code); err != nil {
			return nil, mapMFAError(err)
		}
	} else {
		if err := h.mfa.Verify(ctx, userUUID, req.Body.Code); err != nil {
			return nil, mapMFAError(err)
		}
	}

	user, err := h.users.GetUserByID(ctx, userUUID)
	if err != nil || user == nil {
		return nil, huma.Error401Unauthorized("user not found")
	}

	amr := priorAMRWithOTP(ctx)
	lastOTPAt := nowUnix()
	token, err := h.jwt.GenerateAccessTokenWithAMR(user, amr, lastOTPAt)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to mint stepped-up token")
	}

	resp := &MFAVerifyResponse{}
	resp.Body.Success = true
	resp.Body.AccessToken = token
	resp.Body.TokenType = "Bearer"
	resp.Body.ExpiresIn = 15 * 60 // mirrors jwtService.accessExpiry
	return resp, nil
}

// priorAMRWithOTP returns the caller's existing amr plus "otp", deduplicated.
// When the prior token has no amr (dev tokens, tokens minted before Block A)
// we default to ["pwd","otp"] so the resulting token still looks coherent.
// The "claims" context key is populated by AuthMiddleware.setUserContext.
func priorAMRWithOTP(ctx context.Context) []string {
	var prior []string
	if claims, ok := ctx.Value("claims").(*authModels.JWTClaims); ok && claims != nil {
		prior = claims.AMR
	}
	if len(prior) == 0 {
		prior = []string{"pwd"}
	}
	for _, v := range prior {
		if v == "otp" {
			return prior
		}
	}
	return append(prior, "otp")
}

func nowUnix() int64 {
	return time.Now().Unix()
}

// --- error mapping ---

func mapMFAError(err error) error {
	switch {
	case errors.Is(err, services.ErrMFAInvalidCode):
		return huma.Error401Unauthorized("invalid mfa code")
	case errors.Is(err, services.ErrMFAChallengeMismatch):
		return huma.Error400BadRequest("challenge does not match requested action")
	case errors.Is(err, services.ErrMFANotEnrolled):
		return huma.Error400BadRequest("mfa is not enrolled for this user")
	default:
		return huma.Error400BadRequest("mfa request failed")
	}
}

// --- registration ---

func (h *MFAHandler) RegisterProtectedRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "mfa-enroll-begin",
		Method:      http.MethodPost,
		Path:        "/v1/auth/mfa/enroll/begin",
		Summary:     "Begin MFA (TOTP) enrollment",
		Tags:        []string{"Authentication", "MFA"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, h.EnrollBegin)

	huma.Register(api, huma.Operation{
		OperationID: "mfa-enroll-confirm",
		Method:      http.MethodPost,
		Path:        "/v1/auth/mfa/enroll/confirm",
		Summary:     "Confirm MFA enrollment and receive backup codes",
		Tags:        []string{"Authentication", "MFA"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, h.EnrollConfirm)

	huma.Register(api, huma.Operation{
		OperationID: "mfa-status",
		Method:      http.MethodGet,
		Path:        "/v1/auth/me/mfa",
		Summary:     "Return the current user's MFA enrollment status",
		Tags:        []string{"Authentication", "MFA"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, h.Status)

	huma.Register(api, huma.Operation{
		OperationID: "mfa-remove",
		Method:      http.MethodPost,
		Path:        "/v1/auth/me/mfa/remove",
		Summary:     "Remove the current user's MFA factor",
		Tags:        []string{"Authentication", "MFA"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, h.Remove)

	huma.Register(api, huma.Operation{
		OperationID: "mfa-verify",
		Method:      http.MethodPost,
		Path:        "/v1/auth/mfa/verify",
		Summary:     "Verify a TOTP or backup code; returns a stepped-up access token",
		Tags:        []string{"Authentication", "MFA"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, h.Verify)
}
