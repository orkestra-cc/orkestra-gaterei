package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/orkestra/backend/internal/core/tenant/models"
	"github.com/orkestra/backend/internal/core/tenant/services"
	"github.com/orkestra/backend/internal/shared/middleware"
)

type Handler struct {
	svc *services.Service
}

func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

// --- Request/response envelopes ---

type listMyTenantsOutput struct {
	Body struct {
		Memberships []memberDTO `json:"memberships"`
	}
}

type memberDTO struct {
	TenantID   string   `json:"tenantId"`
	Name       string   `json:"name"`
	Slug       string   `json:"slug"`
	Plan       string   `json:"plan"`
	Kind       string   `json:"kind"`
	Roles      []string `json:"roles"`
	IsOwner    bool     `json:"isOwner"`
}

type createTenantInput struct {
	Body models.CreateTenantInput
}

type tenantOutput struct {
	Body *models.Tenant
}

type tenantIDPath struct {
	TenantID string `path:"tenantId"`
}

type updateTenantInput struct {
	TenantID string `path:"tenantId"`
	Body     models.UpdateTenantInput
}

type updatePlanInput struct {
	TenantID string `path:"tenantId"`
	Body     models.UpdatePlanInput
}

type membershipListOutput struct {
	Body struct {
		Members []models.TenantMembership `json:"members"`
	}
}

type inviteInput struct {
	TenantID string `path:"tenantId"`
	Body     models.InviteInput
}

type inviteOutput struct {
	Body *models.TenantInvite
}

type acceptInviteInput struct {
	Body models.AcceptInviteInput
}

// --- Route registration ---

// RegisterGlobalRoutes registers routes that do not require a tenant context
// (listing your tenants, creating a new tenant, accepting an invite).
func (h *Handler) RegisterGlobalRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-my-tenants",
		Method:      http.MethodGet,
		Path:        "/v1/tenants",
		Summary:     "List tenants the current user belongs to",
		Tags:        []string{"Tenants"},
	}, h.listMyTenants)

	huma.Register(api, huma.Operation{
		OperationID: "create-tenant",
		Method:      http.MethodPost,
		Path:        "/v1/tenants",
		Summary:     "Create a new tenant (caller becomes the owner)",
		Tags:        []string{"Tenants"},
	}, h.createTenant)

	huma.Register(api, huma.Operation{
		OperationID: "accept-invite",
		Method:      http.MethodPost,
		Path:        "/v1/tenants/accept-invite",
		Summary:     "Accept a pending tenant invitation",
		Tags:        []string{"Tenants"},
	}, h.acceptInvite)
}

// RegisterScopedReadRoutes registers read-only per-tenant routes. Safe to
// mount behind the tenant.read permission without MFA.
func (h *Handler) RegisterScopedReadRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-tenant",
		Method:      http.MethodGet,
		Path:        "/v1/tenants/{tenantId}",
		Summary:     "Get a tenant by id",
		Tags:        []string{"Tenants"},
	}, h.getTenant)

	huma.Register(api, huma.Operation{
		OperationID: "list-members",
		Method:      http.MethodGet,
		Path:        "/v1/tenants/{tenantId}/members",
		Summary:     "List tenant members",
		Tags:        []string{"Tenants"},
	}, h.listMembers)
}

// RegisterScopedMutationRoutes registers per-tenant mutations. MFA required
// per Block B — each can change permissions, plan, or destroy the tenant.
func (h *Handler) RegisterScopedMutationRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "update-tenant",
		Method:      http.MethodPatch,
		Path:        "/v1/tenants/{tenantId}",
		Summary:     "Update tenant name, slug or settings",
		Tags:        []string{"Tenants"},
	}, h.updateTenant)

	huma.Register(api, huma.Operation{
		OperationID: "delete-tenant",
		Method:      http.MethodDelete,
		Path:        "/v1/tenants/{tenantId}",
		Summary:     "Archive the tenant (owner only)",
		Tags:        []string{"Tenants"},
	}, h.deleteTenant)

	huma.Register(api, huma.Operation{
		OperationID: "update-plan",
		Method:      http.MethodPatch,
		Path:        "/v1/tenants/{tenantId}/plan",
		Summary:     "Change plan and features",
		Tags:        []string{"Tenants"},
	}, h.updatePlan)

	huma.Register(api, huma.Operation{
		OperationID: "remove-member",
		Method:      http.MethodDelete,
		Path:        "/v1/tenants/{tenantId}/members/{userUUID}",
		Summary:     "Remove a member from the tenant",
		Tags:        []string{"Tenants"},
	}, h.removeMember)

	huma.Register(api, huma.Operation{
		OperationID: "create-invite",
		Method:      http.MethodPost,
		Path:        "/v1/tenants/{tenantId}/invites",
		Summary:     "Invite a user to the tenant",
		Tags:        []string{"Tenants"},
	}, h.createInvite)
}

// --- Handler implementations ---

func (h *Handler) listMyTenants(ctx context.Context, _ *struct{}) (*listMyTenantsOutput, error) {
	userUUID, ok := middleware.GetUserUUID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	mbrs, err := h.svc.ListUserMemberships(ctx, userUUID)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list memberships", err)
	}
	out := &listMyTenantsOutput{}
	for _, m := range mbrs {
		t, err := h.svc.GetTenantModel(ctx, m.TenantUUID)
		if err != nil {
			continue
		}
		out.Body.Memberships = append(out.Body.Memberships, memberDTO{
			TenantID: m.TenantUUID,
			Name:     m.TenantName,
			Slug:     m.TenantSlug,
			Plan:     t.Plan,
			Kind:     m.TenantKind,
			Roles:    m.Roles,
			IsOwner:  m.IsOwner,
		})
	}
	return out, nil
}

func (h *Handler) createTenant(ctx context.Context, in *createTenantInput) (*tenantOutput, error) {
	userUUID, ok := middleware.GetUserUUID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	t, err := h.svc.CreateTenant(ctx, userUUID, in.Body)
	if err != nil {
		return nil, huma.Error400BadRequest("failed to create tenant: " + err.Error())
	}
	return &tenantOutput{Body: t}, nil
}

func (h *Handler) getTenant(ctx context.Context, in *tenantIDPath) (*tenantOutput, error) {
	t, err := h.svc.GetTenantModel(ctx, in.TenantID)
	if err != nil {
		return nil, huma.Error404NotFound("tenant not found")
	}
	return &tenantOutput{Body: t}, nil
}

func (h *Handler) updateTenant(ctx context.Context, in *updateTenantInput) (*tenantOutput, error) {
	if err := h.svc.UpdateTenant(ctx, in.TenantID, in.Body); err != nil {
		return nil, huma.Error400BadRequest("update failed: " + err.Error())
	}
	t, err := h.svc.GetTenantModel(ctx, in.TenantID)
	if err != nil {
		return nil, huma.Error404NotFound("tenant not found")
	}
	return &tenantOutput{Body: t}, nil
}

func (h *Handler) deleteTenant(ctx context.Context, in *tenantIDPath) (*struct{}, error) {
	if err := h.svc.DeleteTenant(ctx, in.TenantID); err != nil {
		return nil, huma.Error400BadRequest("delete failed: " + err.Error())
	}
	return &struct{}{}, nil
}

func (h *Handler) updatePlan(ctx context.Context, in *updatePlanInput) (*tenantOutput, error) {
	if err := h.svc.UpdatePlan(ctx, in.TenantID, in.Body); err != nil {
		return nil, huma.Error400BadRequest("plan update failed: " + err.Error())
	}
	t, err := h.svc.GetTenantModel(ctx, in.TenantID)
	if err != nil {
		return nil, huma.Error404NotFound("tenant not found")
	}
	return &tenantOutput{Body: t}, nil
}

func (h *Handler) listMembers(ctx context.Context, in *tenantIDPath) (*membershipListOutput, error) {
	members, err := h.svc.ListMembers(ctx, in.TenantID)
	if err != nil {
		return nil, huma.Error500InternalServerError("list failed", err)
	}
	out := &membershipListOutput{}
	out.Body.Members = members
	return out, nil
}

type removeMemberInput struct {
	TenantID string `path:"tenantId"`
	UserUUID string `path:"userUUID"`
}

func (h *Handler) removeMember(ctx context.Context, in *removeMemberInput) (*struct{}, error) {
	if err := h.svc.RemoveMember(ctx, in.TenantID, in.UserUUID); err != nil {
		return nil, huma.Error400BadRequest("remove failed: " + err.Error())
	}
	return &struct{}{}, nil
}

func (h *Handler) createInvite(ctx context.Context, in *inviteInput) (*inviteOutput, error) {
	userUUID, _ := middleware.GetUserUUID(ctx)
	inv, err := h.svc.CreateInvite(ctx, in.TenantID, userUUID, in.Body)
	if err != nil {
		return nil, huma.Error400BadRequest("invite failed: " + err.Error())
	}
	return &inviteOutput{Body: inv}, nil
}

func (h *Handler) acceptInvite(ctx context.Context, in *acceptInviteInput) (*tenantOutput, error) {
	userUUID, ok := middleware.GetUserUUID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	t, err := h.svc.AcceptInvite(ctx, userUUID, in.Body.Token)
	if err != nil {
		return nil, huma.Error400BadRequest("accept failed: " + err.Error())
	}
	return &tenantOutput{Body: t}, nil
}

// --- Platform admin routes ---

type adminTenantListItem struct {
	models.Tenant
	MemberCount int `json:"memberCount"`
}

type adminTenantListInput struct {
	IncludeDeleted bool `query:"includeDeleted"`
}

type adminTenantListOutput struct {
	Body struct {
		Tenants []adminTenantListItem `json:"tenants"`
	}
}

type adminInviteListInput struct {
	TenantID        string `path:"tenantId"`
	IncludeAccepted bool   `query:"includeAccepted"`
}

type adminInviteListOutput struct {
	Body struct {
		Invites []models.TenantInvite `json:"invites"`
	}
}

type adminRevokeInviteInput struct {
	TenantID string `path:"tenantId"`
	InviteID string `path:"inviteId"`
}

// RegisterAdminRoutes registers platform-admin routes under /v1/admin/tenants.
// Gated by system.tenants.admin in module.go.
func (h *Handler) RegisterAdminRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-all-tenants-admin",
		Method:      http.MethodGet,
		Path:        "/v1/admin/tenants",
		Summary:     "List every tenant (platform admin)",
		Tags:        []string{"Tenants Admin"},
	}, h.listAllTenantsAdmin)

	huma.Register(api, huma.Operation{
		OperationID: "get-tenant-admin",
		Method:      http.MethodGet,
		Path:        "/v1/admin/tenants/{tenantId}",
		Summary:     "Get a tenant (platform admin)",
		Tags:        []string{"Tenants Admin"},
	}, h.getTenant)

	huma.Register(api, huma.Operation{
		OperationID: "update-tenant-admin",
		Method:      http.MethodPatch,
		Path:        "/v1/admin/tenants/{tenantId}",
		Summary:     "Update a tenant (platform admin)",
		Tags:        []string{"Tenants Admin"},
	}, h.updateTenant)

	huma.Register(api, huma.Operation{
		OperationID: "delete-tenant-admin",
		Method:      http.MethodDelete,
		Path:        "/v1/admin/tenants/{tenantId}",
		Summary:     "Archive a tenant (platform admin)",
		Tags:        []string{"Tenants Admin"},
	}, h.deleteTenant)

	huma.Register(api, huma.Operation{
		OperationID: "update-tenant-plan-admin",
		Method:      http.MethodPatch,
		Path:        "/v1/admin/tenants/{tenantId}/plan",
		Summary:     "Change a tenant's plan (platform admin)",
		Tags:        []string{"Tenants Admin"},
	}, h.updatePlan)

	huma.Register(api, huma.Operation{
		OperationID: "list-tenant-members-admin",
		Method:      http.MethodGet,
		Path:        "/v1/admin/tenants/{tenantId}/members",
		Summary:     "List tenant members (platform admin)",
		Tags:        []string{"Tenants Admin"},
	}, h.listMembers)

	huma.Register(api, huma.Operation{
		OperationID: "remove-tenant-member-admin",
		Method:      http.MethodDelete,
		Path:        "/v1/admin/tenants/{tenantId}/members/{userUUID}",
		Summary:     "Remove a member from a tenant (platform admin)",
		Tags:        []string{"Tenants Admin"},
	}, h.removeMember)

	huma.Register(api, huma.Operation{
		OperationID: "list-tenant-invites-admin",
		Method:      http.MethodGet,
		Path:        "/v1/admin/tenants/{tenantId}/invites",
		Summary:     "List pending tenant invites (platform admin)",
		Tags:        []string{"Tenants Admin"},
	}, h.listInvitesAdmin)

	huma.Register(api, huma.Operation{
		OperationID: "create-tenant-invite-admin",
		Method:      http.MethodPost,
		Path:        "/v1/admin/tenants/{tenantId}/invites",
		Summary:     "Create a tenant invite (platform admin)",
		Tags:        []string{"Tenants Admin"},
	}, h.createInvite)

	huma.Register(api, huma.Operation{
		OperationID: "revoke-tenant-invite-admin",
		Method:      http.MethodDelete,
		Path:        "/v1/admin/tenants/{tenantId}/invites/{inviteId}",
		Summary:     "Revoke a pending tenant invite (platform admin)",
		Tags:        []string{"Tenants Admin"},
	}, h.revokeInviteAdmin)
}

func (h *Handler) listAllTenantsAdmin(ctx context.Context, in *adminTenantListInput) (*adminTenantListOutput, error) {
	views, err := h.svc.ListAllTenants(ctx, in.IncludeDeleted)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list tenants", err)
	}
	out := &adminTenantListOutput{}
	out.Body.Tenants = make([]adminTenantListItem, 0, len(views))
	for _, v := range views {
		out.Body.Tenants = append(out.Body.Tenants, adminTenantListItem{Tenant: *v.Tenant, MemberCount: v.MemberCount})
	}
	return out, nil
}

func (h *Handler) listInvitesAdmin(ctx context.Context, in *adminInviteListInput) (*adminInviteListOutput, error) {
	onlyPending := !in.IncludeAccepted
	invs, err := h.svc.ListInvites(ctx, in.TenantID, onlyPending)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list invites", err)
	}
	out := &adminInviteListOutput{}
	out.Body.Invites = invs
	return out, nil
}

func (h *Handler) revokeInviteAdmin(ctx context.Context, in *adminRevokeInviteInput) (*struct{}, error) {
	if err := h.svc.RevokeInvite(ctx, in.TenantID, in.InviteID); err != nil {
		return nil, huma.Error400BadRequest("revoke failed: " + err.Error())
	}
	return &struct{}{}, nil
}
