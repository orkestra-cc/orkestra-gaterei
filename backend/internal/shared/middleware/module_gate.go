package middleware

import (
	"context"
	"encoding/json"
	"net/http"
)

// ModuleEnabledChecker is satisfied by ModuleConfigService.
// Defined here as an interface to avoid a circular import between
// the middleware and module packages.
type ModuleEnabledChecker interface {
	IsEnabled(ctx context.Context, moduleName string) bool
}

// ModuleGate returns middleware that rejects requests with 503 Service Unavailable
// when the named module is disabled at runtime. Applied to non-core modules only.
func ModuleGate(checker ModuleEnabledChecker, moduleName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !checker.IsEnabled(r.Context(), moduleName) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]string{
					"error":  "module_disabled",
					"module": moduleName,
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
