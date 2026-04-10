package models

import (
	"time"
)

type TokenPair struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	TokenType    string    `json:"tokenType"`
	ExpiresIn    int64     `json:"expiresIn"`
	SessionID    string    `json:"sessionId"`
	DeviceID     string    `json:"deviceId,omitempty"`
	Scope        []string  `json:"scope,omitempty"`
	IssuedAt     time.Time `json:"issuedAt"`
	RefreshCount int       `json:"refreshCount,omitempty"`
}

// OrgMembership is the subset of a tenant membership embedded in JWT claims.
// Only org id and role names are embedded — permissions are resolved per
// request by the authz module so revocation is instant.
type OrgMembership struct {
	OrgUUID string   `json:"oid"`
	Roles   []string `json:"r"`
}

type JWTClaims struct {
	// Standard JWT claims
	UserUUID   string `json:"sub"`           // User UUID (subject)
	Email      string `json:"email"`         // User email
	SystemRole string `json:"srole"`         // Global system role (developer/administrator/user)
	TokenType  string `json:"type"`          // "access" or "refresh"
	ExpiresAt  int64  `json:"exp"`           // Expiration timestamp
	IssuedAt   int64  `json:"iat"`           // Issued at timestamp
	NotBefore  int64  `json:"nbf,omitempty"` // Not before timestamp
	Issuer     string `json:"iss"`           // Token issuer
	Audience   string `json:"aud,omitempty"` // Token audience

	// Multi-tenancy claims
	Memberships  []OrgMembership `json:"mbr,omitempty"`  // orgs the user belongs to
	DefaultOrgID string          `json:"dorg,omitempty"` // selected when X-Org-ID header is absent

	// Security and session claims
	SessionID   string  `json:"sid"`            // Session identifier
	DeviceID    string  `json:"did"`            // Device identifier
	IPAddress   string  `json:"ip,omitempty"`   // IP address (for binding)
	Fingerprint string  `json:"fp,omitempty"`   // Device fingerprint
	RiskScore   float64 `json:"risk,omitempty"` // Risk assessment score

	// OAuth and provider claims
	OAuthProvider string `json:"provider,omitempty"` // Primary OAuth provider
	ProviderID    string `json:"pid,omitempty"`      // Provider user ID

	// OAuth scopes
	Scope []string `json:"scope,omitempty"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken,omitempty"`
	SessionID    string `json:"sessionId,omitempty"`
	DeviceID     string `json:"deviceId,omitempty"`
	AllDevices   bool   `json:"allDevices"`
	Reason       string `json:"reason,omitempty"`
}

type TokenValidationResult struct {
	Valid            bool
	Claims           *JWTClaims
	Error            error
	ExpiresAt        time.Time
	RiskScore        float64
	RequiresMFA      bool
	RequiresReauth   bool
	SecurityWarnings []string
}

type OAuthState struct {
	State       string    `json:"state"`
	RedirectURL string    `json:"redirectUrl"`
	DeviceID    string    `json:"deviceId,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
	// PKCE fields for OAuth 2.1 compliance
	CodeChallenge       string `json:"codeChallenge,omitempty"`
	CodeChallengeMethod string `json:"codeChallengeMethod,omitempty"`
	Nonce               string `json:"nonce,omitempty"`
	// Enhanced security fields
	IPAddress   string  `json:"ipAddress,omitempty"`
	UserAgent   string  `json:"userAgent,omitempty"`
	Fingerprint string  `json:"fingerprint,omitempty"`
	RiskScore   float64 `json:"riskScore,omitempty"`
}

// Enhanced token request types
type TokenRequest struct {
	GrantType    string           `json:"grantType" validate:"required"`
	Code         string           `json:"code,omitempty"`
	RedirectURI  string           `json:"redirectUri,omitempty"`
	RefreshToken string           `json:"refreshToken,omitempty"`
	Scope        []string         `json:"scope,omitempty"`
	DeviceInfo   *DeviceInfo      `json:"deviceInfo,omitempty"`
	SecurityInfo *SecurityContext `json:"securityInfo,omitempty"`
	PKCE         *PKCEData        `json:"pkce,omitempty"`
}

type PKCEData struct {
	CodeVerifier  string `json:"codeVerifier" validate:"required"`
	CodeChallenge string `json:"codeChallenge" validate:"required"`
	Method        string `json:"method" validate:"required,oneof=S256 plain"`
}

// Risk assessment types
type RiskFactors struct {
	UnknownDevice    bool     `json:"unknownDevice"`
	UnknownLocation  bool     `json:"unknownLocation"`
	UnusualTime      bool     `json:"unusualTime"`
	HighRiskIP       bool     `json:"highRiskIP"`
	VPNDetected      bool     `json:"vpnDetected"`
	TorDetected      bool     `json:"torDetected"`
	BotDetected      bool     `json:"botDetected"`
	MultipleFailures bool     `json:"multipleFailures"`
	RiskScore        float64  `json:"riskScore"`
	RequiresMFA      bool     `json:"requiresMfa"`
	RequiresReauth   bool     `json:"requiresReauth"`
	Recommendations  []string `json:"recommendations"`
}
