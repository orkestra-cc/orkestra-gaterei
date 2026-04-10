package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/orkestra/backend/internal/core/authz/models"
	"github.com/orkestra/backend/internal/core/authz/services"
	"github.com/orkestra/backend/internal/shared/middleware"
)

type Handler struct {
	svc *services.Service
}

func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

// --- Input/output envelopes ---

type permissionsOutput struct {
	Body models.PermissionCatalogResponse
}

type rolesOutput struct {
	Body models.RoleListResponse
}

type roleOutput struct {
	Body *models.Role
}

type bindingsOutput struct {
	Body models.BindingListResponse
}

type bindingOutput struct {
	Body *models.Binding
}

type effectiveOutput struct {
	Body models.EffectivePermissionsResponse
}

type listRolesInput struct {
	OrgID string `path:"orgId"`
}

type createRoleInput struct {
	OrgID string `path:"orgId"`
	Body  models.CreateRoleInput
}

type deleteRoleInput struct {
	OrgID string `path:"orgId"`
	Role  string `path:"roleId"`
}

type createBindingInput struct {
	OrgID string `path:"orgId"`
	Body  models.CreateBindingInput
}

type deleteBindingInput struct {
	OrgID   string `path:"orgId"`
	Binding string `path:"bindingId"`
}

type listBindingsInput struct {
	OrgID string `path:"orgId"`
}

type effectiveInput struct {
	OrgID string `path:"orgId"`
}

// --- Routes ---

// RegisterGlobalRoutes registers permission-catalog routes that are read-only
// and don't need an org context (the catalog is shared across tenants).
func (h *Handler) RegisterGlobalRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-permissions",
		Method:      http.MethodGet,
		Path:        "/v1/authz/permissions",
		Summary:     "List the permission catalog (system-generated)",
		Tags:        []string{"Authorization"},
	}, h.listPermissions)
}

// RegisterScopedRoutes registers per-org role and binding admin routes.
func (h *Handler) RegisterScopedRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-roles",
		Method:      http.MethodGet,
		Path:        "/v1/orgs/{orgId}/authz/roles",
		Summary:     "List roles (system + custom)",
		Tags:        []string{"Authorization"},
	}, h.listRoles)

	huma.Register(api, huma.Operation{
		OperationID: "create-role",
		Method:      http.MethodPost,
		Path:        "/v1/orgs/{orgId}/authz/roles",
		Summary:     "Create a custom role",
		Tags:        []string{"Authorization"},
	}, h.createRole)

	huma.Register(api, huma.Operation{
		OperationID: "delete-role",
		Method:      http.MethodDelete,
		Path:        "/v1/orgs/{orgId}/authz/roles/{roleId}",
		Summary:     "Delete a custom role",
		Tags:        []string{"Authorization"},
	}, h.deleteRole)

	huma.Register(api, huma.Operation{
		OperationID: "list-bindings",
		Method:      http.MethodGet,
		Path:        "/v1/orgs/{orgId}/authz/bindings",
		Summary:     "List role bindings in the org",
		Tags:        []string{"Authorization"},
	}, h.listBindings)

	huma.Register(api, huma.Operation{
		OperationID: "create-binding",
		Method:      http.MethodPost,
		Path:        "/v1/orgs/{orgId}/authz/bindings",
		Summary:     "Grant a role to a user with optional expiration",
		Tags:        []string{"Authorization"},
	}, h.createBinding)

	huma.Register(api, huma.Operation{
		OperationID: "delete-binding",
		Method:      http.MethodDelete,
		Path:        "/v1/orgs/{orgId}/authz/bindings/{bindingId}",
		Summary:     "Revoke a role binding",
		Tags:        []string{"Authorization"},
	}, h.deleteBinding)

	huma.Register(api, huma.Operation{
		OperationID: "get-effective-permissions",
		Method:      http.MethodGet,
		Path:        "/v1/orgs/{orgId}/authz/me",
		Summary:     "Get the current user's effective permissions in the org",
		Tags:        []string{"Authorization"},
	}, h.getEffective)
}

// --- Handler implementations ---

func (h *Handler) listPermissions(ctx context.Context, _ *struct{}) (*permissionsOutput, error) {
	perms, err := h.svc.ListPermissions(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("list permissions failed", err)
	}
	return &permissionsOutput{Body: models.PermissionCatalogResponse{Permissions: perms}}, nil
}

func (h *Handler) listRoles(ctx context.Context, in *listRolesInput) (*rolesOutput, error) {
	roles, err := h.svc.ListRoles(ctx, in.OrgID)
	if err != nil {
		return nil, huma.Error500InternalServerError("list roles failed", err)
	}
	return &rolesOutput{Body: models.RoleListResponse{Roles: roles}}, nil
}

func (h *Handler) createRole(ctx context.Context, in *createRoleInput) (*roleOutput, error) {
	role, err := h.svc.CreateRole(ctx, in.OrgID, in.Body)
	if err != nil {
		return nil, huma.Error400BadRequest("create role failed: " + err.Error())
	}
	return &roleOutput{Body: role}, nil
}

func (h *Handler) deleteRole(ctx context.Context, in *deleteRoleInput) (*struct{}, error) {
	if err := h.svc.DeleteRole(ctx, in.Role); err != nil {
		return nil, huma.Error400BadRequest("delete role failed: " + err.Error())
	}
	return &struct{}{}, nil
}

func (h *Handler) listBindings(ctx context.Context, in *listBindingsInput) (*bindingsOutput, error) {
	bindings, err := h.svc.ListBindings(ctx, in.OrgID)
	if err != nil {
		return nil, huma.Error500InternalServerError("list bindings failed", err)
	}
	return &bindingsOutput{Body: models.BindingListResponse{Bindings: bindings}}, nil
}

func (h *Handler) createBinding(ctx context.Context, in *createBindingInput) (*bindingOutput, error) {
	grantedBy, _ := middleware.GetUserUUID(ctx)
	b, err := h.svc.CreateBinding(ctx, in.OrgID, grantedBy, in.Body)
	if err != nil {
		return nil, huma.Error400BadRequest("create binding failed: " + err.Error())
	}
	return &bindingOutput{Body: b}, nil
}

func (h *Handler) deleteBinding(ctx context.Context, in *deleteBindingInput) (*struct{}, error) {
	if err := h.svc.DeleteBinding(ctx, in.Binding); err != nil {
		return nil, huma.Error400BadRequest("delete binding failed: " + err.Error())
	}
	return &struct{}{}, nil
}

func (h *Handler) getEffective(ctx context.Context, in *effectiveInput) (*effectiveOutput, error) {
	userUUID, ok := middleware.GetUserUUID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	perms, err := h.svc.GetEffectivePermissions(ctx, userUUID, in.OrgID)
	if err != nil {
		return nil, huma.Error500InternalServerError("effective permissions failed", err)
	}
	systemRole, _ := middleware.GetSystemRole(ctx)
	return &effectiveOutput{Body: models.EffectivePermissionsResponse{
		OrgID:       in.OrgID,
		Permissions: perms,
		SystemRole:  systemRole,
	}}, nil
}
