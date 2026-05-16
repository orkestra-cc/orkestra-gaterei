package logging

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/orkestra/backend/internal/core/logging/handlers"
)

// RegisterRoutes mounts the admin observability endpoints. All four
// are administrator-gated at the API level via the bearerAuth
// security scheme + the operator-protected router (middleware adds
// RequireRole(administrator) for any operation tagged with the
// `administrator` scope).
//
// ADR-0005 Phase F — the surface is intentionally narrow: read all,
// set global, set per-module, unset per-module, reset to env. Bulk
// edits / time-bounded overrides / scheduled reverts are deferred.
func RegisterRoutes(api huma.API, h *handlers.LogLevelHandler) {
	huma.Register(api, huma.Operation{
		OperationID: "admin-observability-loglevels-get",
		Method:      http.MethodGet,
		Path:        "/v1/admin/observability/log-levels",
		Summary:     "Get effective log levels",
		Description: "Returns the global threshold + per-module overrides + the catalog of known modules. ADR-0005 Phase F.",
		Tags:        []string{"Observability"},
		Security:    []map[string][]string{{"bearerAuth": {"administrator"}}},
	}, h.Get)

	huma.Register(api, huma.Operation{
		OperationID: "admin-observability-loglevels-set-global",
		Method:      http.MethodPut,
		Path:        "/v1/admin/observability/log-levels/global",
		Summary:     "Set the global log level",
		Tags:        []string{"Observability"},
		Security:    []map[string][]string{{"bearerAuth": {"administrator"}}},
	}, h.SetGlobal)

	huma.Register(api, huma.Operation{
		OperationID: "admin-observability-loglevels-set-module",
		Method:      http.MethodPut,
		Path:        "/v1/admin/observability/log-levels/{module}",
		Summary:     "Set a per-module log level override",
		Tags:        []string{"Observability"},
		Security:    []map[string][]string{{"bearerAuth": {"administrator"}}},
	}, h.SetModule)

	huma.Register(api, huma.Operation{
		OperationID: "admin-observability-loglevels-unset-module",
		Method:      http.MethodDelete,
		Path:        "/v1/admin/observability/log-levels/{module}",
		Summary:     "Remove a per-module override (fall back to global)",
		Tags:        []string{"Observability"},
		Security:    []map[string][]string{{"bearerAuth": {"administrator"}}},
	}, h.UnsetModule)

	huma.Register(api, huma.Operation{
		OperationID: "admin-observability-loglevels-reset",
		Method:      http.MethodPost,
		Path:        "/v1/admin/observability/log-levels/reset",
		Summary:     "Revert global + per-module to boot env defaults",
		Tags:        []string{"Observability"},
		Security:    []map[string][]string{{"bearerAuth": {"administrator"}}},
	}, h.Reset)
}
