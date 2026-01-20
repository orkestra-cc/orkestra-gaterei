package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/orkestra/backend/internal/auth/services"
	"github.com/orkestra/backend/internal/shared/config"
	userModels "github.com/orkestra/backend/internal/user/models"
)

// DevTokenHandler handles development token generation endpoints
type DevTokenHandler struct {
	jwtService services.JWTService
	cfg        *config.Config
	logger     *slog.Logger
}

// NewDevTokenHandler creates a new DevTokenHandler
func NewDevTokenHandler(jwtService services.JWTService, cfg *config.Config) *DevTokenHandler {
	return &DevTokenHandler{
		jwtService: jwtService,
		cfg:        cfg,
		logger:     slog.Default().With(slog.String("handler", "dev_token")),
	}
}

// ValidRoles contains all valid roles for token generation
var ValidRoles = []string{"developer", "ceo", "administrator", "manager", "operator", "guest"}

// isValidRole checks if a role is valid
func isValidRole(role string) bool {
	for _, r := range ValidRoles {
		if r == role {
			return true
		}
	}
	return false
}

// GenerateTokenRequest is the request body for token generation
type GenerateTokenRequest struct {
	Body struct {
		Role   string `json:"role" validate:"required" doc:"Role for the generated token (developer, ceo, administrator, manager, operator, guest)"`
		Expiry string `json:"expiry,omitempty" doc:"Token expiry duration (e.g., '15m', '1h', '24h'). Default: 15m, Max: 24h"`
	}
}

// GenerateTokenResponse is the response for token generation
type GenerateTokenResponse struct {
	Body struct {
		AccessToken string    `json:"accessToken" doc:"JWT access token"`
		Role        string    `json:"role" doc:"Role assigned to the token"`
		Email       string    `json:"email" doc:"Synthetic email used for the token"`
		ExpiresAt   time.Time `json:"expiresAt" doc:"Token expiration timestamp"`
		ExpiresIn   int64     `json:"expiresIn" doc:"Seconds until token expires"`
		Curl        string    `json:"curl" doc:"Example curl command to use this token"`
	}
}

// GenerateToken generates a JWT token for a specified role (dev/staging only)
func (h *DevTokenHandler) GenerateToken(ctx context.Context, req *GenerateTokenRequest) (*GenerateTokenResponse, error) {
	// Defense in depth: double-check we're not in production
	if h.cfg.IsProduction() {
		h.logger.Error("Attempted to generate dev token in production")
		return nil, fmt.Errorf("dev token generation is disabled in production")
	}

	// Validate role
	if !isValidRole(req.Body.Role) {
		return nil, fmt.Errorf("invalid role '%s'. Valid roles: %v", req.Body.Role, ValidRoles)
	}

	// Parse expiry duration (default 15 minutes, max 24 hours)
	expiry := 15 * time.Minute
	if req.Body.Expiry != "" {
		parsed, err := time.ParseDuration(req.Body.Expiry)
		if err != nil {
			return nil, fmt.Errorf("invalid expiry format: %w. Use formats like '15m', '1h', '24h'", err)
		}
		if parsed > 24*time.Hour {
			return nil, fmt.Errorf("expiry cannot exceed 24 hours")
		}
		if parsed < 1*time.Minute {
			return nil, fmt.Errorf("expiry must be at least 1 minute")
		}
		expiry = parsed
	}

	// Create synthetic user (no database write)
	syntheticEmail := fmt.Sprintf("%s@orkestra.dev", req.Body.Role)
	user := &userModels.User{
		UUID:          fmt.Sprintf("dev-%s-%d", req.Body.Role, time.Now().Unix()),
		Email:         syntheticEmail,
		FullName:      fmt.Sprintf("Dev %s", req.Body.Role),
		Role:          req.Body.Role,
		IsActive:      true,
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Generate access token
	token, err := h.jwtService.GenerateAccessToken(user)
	if err != nil {
		h.logger.Error("Failed to generate dev token", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	expiresAt := time.Now().Add(expiry)

	// Log token generation in staging for audit
	if h.cfg.IsStaging() {
		h.logger.Info("Dev token generated in staging",
			slog.String("role", req.Body.Role),
			slog.String("expiry", expiry.String()),
		)
	}

	resp := &GenerateTokenResponse{}
	resp.Body.AccessToken = token
	resp.Body.Role = req.Body.Role
	resp.Body.Email = syntheticEmail
	resp.Body.ExpiresAt = expiresAt
	resp.Body.ExpiresIn = int64(expiry.Seconds())
	resp.Body.Curl = fmt.Sprintf("curl -H 'Authorization: Bearer %s' http://localhost:3000/api/v1/users", token)

	return resp, nil
}

// ListRolesRequest is the request for listing roles (empty)
type ListRolesRequest struct{}

// ListRolesResponse is the response for listing available roles
type ListRolesResponse struct {
	Body struct {
		Roles       []string `json:"roles" doc:"Available roles for token generation"`
		Environment string   `json:"environment" doc:"Current environment"`
	}
}

// ListRoles returns the available roles for token generation
func (h *DevTokenHandler) ListRoles(ctx context.Context, req *ListRolesRequest) (*ListRolesResponse, error) {
	// Defense in depth: double-check we're not in production
	if h.cfg.IsProduction() {
		h.logger.Error("Attempted to list dev roles in production")
		return nil, fmt.Errorf("dev token generation is disabled in production")
	}

	resp := &ListRolesResponse{}
	resp.Body.Roles = ValidRoles
	resp.Body.Environment = h.cfg.GetEnvironment()

	return resp, nil
}
