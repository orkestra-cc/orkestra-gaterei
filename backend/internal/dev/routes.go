package dev

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/orkestra/backend/internal/dev/handlers"
)

// RegisterRoutes registers development-only routes
// IMPORTANT: Only call this in non-production environments
func RegisterRoutes(api huma.API, devTokenHandler *handlers.DevTokenHandler) {
	// Token generation endpoint (hidden from Scalar/OpenAPI docs)
	huma.Register(api, huma.Operation{
		OperationID: "generate-dev-token",
		Method:      http.MethodPost,
		Path:        "/api/v1/dev/token",
		Summary:     "Generate development JWT token",
		Description: "Generates a JWT token for testing purposes. Only available in development and staging environments. Creates a synthetic user (no database writes).",
		Tags:        []string{"Development"},
		Hidden:      true,
	}, devTokenHandler.GenerateToken)

	// List available roles (hidden from Scalar/OpenAPI docs)
	huma.Register(api, huma.Operation{
		OperationID: "list-dev-roles",
		Method:      http.MethodGet,
		Path:        "/api/v1/dev/token/roles",
		Summary:     "List available roles for token generation",
		Description: "Returns the list of valid roles that can be used when generating development tokens.",
		Tags:        []string{"Development"},
		Hidden:      true,
	}, devTokenHandler.ListRoles)
}
