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

type listMyOrgsOutput struct {
	Body struct {
		Memberships []memberDTO `json:"memberships"`
	}
}

type memberDTO struct {
	OrgID   string   `json:"orgId"`
	Name    string   `json:"name"`
	Slug    string   `json:"slug"`
	Plan    string   `json:"plan"`
	Roles   []string `json:"roles"`
	IsOwner bool     `json:"isOwner"`
}

type createOrgInput struct {
	Body models.CreateOrgInput
}

type orgOutput struct {
	Body *models.Org
}

type orgIDPath struct {
	OrgID string `path:"orgId"`
}

type updateOrgInput struct {
	OrgID string `path:"orgId"`
	Body  models.UpdateOrgInput
}

type updatePlanInput struct {
	OrgID string `path:"orgId"`
	Body  models.UpdatePlanInput
}

type membershipListOutput struct {
	Body struct {
		Members []models.Membership `json:"members"`
	}
}

type inviteInput struct {
	OrgID string `path:"orgId"`
	Body  models.InviteInput
}

type inviteOutput struct {
	Body *models.Invite
}

type acceptInviteInput struct {
	Body models.AcceptInviteInput
}

// --- Route registration ---

// RegisterGlobalRoutes registers routes that do not require an org context
// (listing your orgs, creating a new org, accepting an invite).
func (h *Handler) RegisterGlobalRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-my-orgs",
		Method:      http.MethodGet,
		Path:        "/v1/orgs",
		Summary:     "List organizations the current user belongs to",
		Tags:        []string{"Organizations"},
	}, h.listMyOrgs)

	huma.Register(api, huma.Operation{
		OperationID: "create-org",
		Method:      http.MethodPost,
		Path:        "/v1/orgs",
		Summary:     "Create a new organization (caller becomes the owner)",
		Tags:        []string{"Organizations"},
	}, h.createOrg)

	huma.Register(api, huma.Operation{
		OperationID: "accept-invite",
		Method:      http.MethodPost,
		Path:        "/v1/orgs/accept-invite",
		Summary:     "Accept a pending org invitation",
		Tags:        []string{"Organizations"},
	}, h.acceptInvite)
}

// RegisterScopedRoutes registers routes that operate on a specific org.
// They require the caller to be an org administrator.
func (h *Handler) RegisterScopedRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-org",
		Method:      http.MethodGet,
		Path:        "/v1/orgs/{orgId}",
		Summary:     "Get an organization by id",
		Tags:        []string{"Organizations"},
	}, h.getOrg)

	huma.Register(api, huma.Operation{
		OperationID: "update-org",
		Method:      http.MethodPatch,
		Path:        "/v1/orgs/{orgId}",
		Summary:     "Update organization name, slug or settings",
		Tags:        []string{"Organizations"},
	}, h.updateOrg)

	huma.Register(api, huma.Operation{
		OperationID: "delete-org",
		Method:      http.MethodDelete,
		Path:        "/v1/orgs/{orgId}",
		Summary:     "Soft-delete the organization (owner only)",
		Tags:        []string{"Organizations"},
	}, h.deleteOrg)

	huma.Register(api, huma.Operation{
		OperationID: "update-plan",
		Method:      http.MethodPatch,
		Path:        "/v1/orgs/{orgId}/plan",
		Summary:     "Change plan and features",
		Tags:        []string{"Organizations"},
	}, h.updatePlan)

	huma.Register(api, huma.Operation{
		OperationID: "list-members",
		Method:      http.MethodGet,
		Path:        "/v1/orgs/{orgId}/members",
		Summary:     "List org members",
		Tags:        []string{"Organizations"},
	}, h.listMembers)

	huma.Register(api, huma.Operation{
		OperationID: "remove-member",
		Method:      http.MethodDelete,
		Path:        "/v1/orgs/{orgId}/members/{userUUID}",
		Summary:     "Remove a member from the org",
		Tags:        []string{"Organizations"},
	}, h.removeMember)

	huma.Register(api, huma.Operation{
		OperationID: "create-invite",
		Method:      http.MethodPost,
		Path:        "/v1/orgs/{orgId}/invites",
		Summary:     "Invite a user to the org",
		Tags:        []string{"Organizations"},
	}, h.createInvite)
}

// --- Handler implementations ---

func (h *Handler) listMyOrgs(ctx context.Context, _ *struct{}) (*listMyOrgsOutput, error) {
	userUUID, ok := middleware.GetUserUUID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	mbrs, err := h.svc.ListUserMemberships(ctx, userUUID)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list memberships", err)
	}
	out := &listMyOrgsOutput{}
	for _, m := range mbrs {
		o, err := h.svc.GetOrgModel(ctx, m.OrgUUID)
		if err != nil {
			continue
		}
		out.Body.Memberships = append(out.Body.Memberships, memberDTO{
			OrgID:   m.OrgUUID,
			Name:    m.OrgName,
			Slug:    m.OrgSlug,
			Plan:    o.Plan,
			Roles:   m.Roles,
			IsOwner: m.IsOwner,
		})
	}
	return out, nil
}

func (h *Handler) createOrg(ctx context.Context, in *createOrgInput) (*orgOutput, error) {
	userUUID, ok := middleware.GetUserUUID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	org, err := h.svc.CreateOrg(ctx, userUUID, in.Body)
	if err != nil {
		return nil, huma.Error400BadRequest("failed to create org: " + err.Error())
	}
	return &orgOutput{Body: org}, nil
}

func (h *Handler) getOrg(ctx context.Context, in *orgIDPath) (*orgOutput, error) {
	org, err := h.svc.GetOrgModel(ctx, in.OrgID)
	if err != nil {
		return nil, huma.Error404NotFound("org not found")
	}
	return &orgOutput{Body: org}, nil
}

func (h *Handler) updateOrg(ctx context.Context, in *updateOrgInput) (*orgOutput, error) {
	if err := h.svc.UpdateOrg(ctx, in.OrgID, in.Body); err != nil {
		return nil, huma.Error400BadRequest("update failed: " + err.Error())
	}
	org, err := h.svc.GetOrgModel(ctx, in.OrgID)
	if err != nil {
		return nil, huma.Error404NotFound("org not found")
	}
	return &orgOutput{Body: org}, nil
}

func (h *Handler) deleteOrg(ctx context.Context, in *orgIDPath) (*struct{}, error) {
	if err := h.svc.DeleteOrg(ctx, in.OrgID); err != nil {
		return nil, huma.Error400BadRequest("delete failed: " + err.Error())
	}
	return &struct{}{}, nil
}

func (h *Handler) updatePlan(ctx context.Context, in *updatePlanInput) (*orgOutput, error) {
	if err := h.svc.UpdatePlan(ctx, in.OrgID, in.Body); err != nil {
		return nil, huma.Error400BadRequest("plan update failed: " + err.Error())
	}
	org, err := h.svc.GetOrgModel(ctx, in.OrgID)
	if err != nil {
		return nil, huma.Error404NotFound("org not found")
	}
	return &orgOutput{Body: org}, nil
}

func (h *Handler) listMembers(ctx context.Context, in *orgIDPath) (*membershipListOutput, error) {
	members, err := h.svc.ListMembers(ctx, in.OrgID)
	if err != nil {
		return nil, huma.Error500InternalServerError("list failed", err)
	}
	out := &membershipListOutput{}
	out.Body.Members = members
	return out, nil
}

type removeMemberInput struct {
	OrgID    string `path:"orgId"`
	UserUUID string `path:"userUUID"`
}

func (h *Handler) removeMember(ctx context.Context, in *removeMemberInput) (*struct{}, error) {
	if err := h.svc.RemoveMember(ctx, in.OrgID, in.UserUUID); err != nil {
		return nil, huma.Error400BadRequest("remove failed: " + err.Error())
	}
	return &struct{}{}, nil
}

func (h *Handler) createInvite(ctx context.Context, in *inviteInput) (*inviteOutput, error) {
	userUUID, _ := middleware.GetUserUUID(ctx)
	inv, err := h.svc.CreateInvite(ctx, in.OrgID, userUUID, in.Body)
	if err != nil {
		return nil, huma.Error400BadRequest("invite failed: " + err.Error())
	}
	return &inviteOutput{Body: inv}, nil
}

func (h *Handler) acceptInvite(ctx context.Context, in *acceptInviteInput) (*orgOutput, error) {
	userUUID, ok := middleware.GetUserUUID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	org, err := h.svc.AcceptInvite(ctx, userUUID, in.Body.Token)
	if err != nil {
		return nil, huma.Error400BadRequest("accept failed: " + err.Error())
	}
	return &orgOutput{Body: org}, nil
}

// --- Platform admin routes ---

type adminOrgListItem struct {
	models.Org
	MemberCount int `json:"memberCount"`
}

type adminOrgListInput struct {
	IncludeDeleted bool `query:"includeDeleted"`
}

type adminOrgListOutput struct {
	Body struct {
		Orgs []adminOrgListItem `json:"orgs"`
	}
}

type adminInviteListInput struct {
	OrgID           string `path:"orgId"`
	IncludeAccepted bool   `query:"includeAccepted"`
}

type adminInviteListOutput struct {
	Body struct {
		Invites []models.Invite `json:"invites"`
	}
}

type adminRevokeInviteInput struct {
	OrgID    string `path:"orgId"`
	InviteID string `path:"inviteId"`
}

// RegisterAdminRoutes registers platform-admin routes under /v1/admin/orgs.
// The route group is gated by system.tenants.admin in module.go so platform
// operators can manage every tenant without joining each one.
func (h *Handler) RegisterAdminRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-all-orgs-admin",
		Method:      http.MethodGet,
		Path:        "/v1/admin/orgs",
		Summary:     "List every organization (platform admin)",
		Tags:        []string{"Organizations Admin"},
	}, h.listAllOrgsAdmin)

	huma.Register(api, huma.Operation{
		OperationID: "get-org-admin",
		Method:      http.MethodGet,
		Path:        "/v1/admin/orgs/{orgId}",
		Summary:     "Get an organization (platform admin)",
		Tags:        []string{"Organizations Admin"},
	}, h.getOrg)

	huma.Register(api, huma.Operation{
		OperationID: "update-org-admin",
		Method:      http.MethodPatch,
		Path:        "/v1/admin/orgs/{orgId}",
		Summary:     "Update an organization (platform admin)",
		Tags:        []string{"Organizations Admin"},
	}, h.updateOrg)

	huma.Register(api, huma.Operation{
		OperationID: "delete-org-admin",
		Method:      http.MethodDelete,
		Path:        "/v1/admin/orgs/{orgId}",
		Summary:     "Soft-delete an organization (platform admin)",
		Tags:        []string{"Organizations Admin"},
	}, h.deleteOrg)

	huma.Register(api, huma.Operation{
		OperationID: "update-org-plan-admin",
		Method:      http.MethodPatch,
		Path:        "/v1/admin/orgs/{orgId}/plan",
		Summary:     "Change an organization's plan (platform admin)",
		Tags:        []string{"Organizations Admin"},
	}, h.updatePlan)

	huma.Register(api, huma.Operation{
		OperationID: "list-org-members-admin",
		Method:      http.MethodGet,
		Path:        "/v1/admin/orgs/{orgId}/members",
		Summary:     "List organization members (platform admin)",
		Tags:        []string{"Organizations Admin"},
	}, h.listMembers)

	huma.Register(api, huma.Operation{
		OperationID: "remove-org-member-admin",
		Method:      http.MethodDelete,
		Path:        "/v1/admin/orgs/{orgId}/members/{userUUID}",
		Summary:     "Remove a member from an organization (platform admin)",
		Tags:        []string{"Organizations Admin"},
	}, h.removeMember)

	huma.Register(api, huma.Operation{
		OperationID: "list-org-invites-admin",
		Method:      http.MethodGet,
		Path:        "/v1/admin/orgs/{orgId}/invites",
		Summary:     "List pending organization invites (platform admin)",
		Tags:        []string{"Organizations Admin"},
	}, h.listInvitesAdmin)

	huma.Register(api, huma.Operation{
		OperationID: "create-org-invite-admin",
		Method:      http.MethodPost,
		Path:        "/v1/admin/orgs/{orgId}/invites",
		Summary:     "Create an organization invite (platform admin)",
		Tags:        []string{"Organizations Admin"},
	}, h.createInvite)

	huma.Register(api, huma.Operation{
		OperationID: "revoke-org-invite-admin",
		Method:      http.MethodDelete,
		Path:        "/v1/admin/orgs/{orgId}/invites/{inviteId}",
		Summary:     "Revoke a pending organization invite (platform admin)",
		Tags:        []string{"Organizations Admin"},
	}, h.revokeInviteAdmin)
}

func (h *Handler) listAllOrgsAdmin(ctx context.Context, in *adminOrgListInput) (*adminOrgListOutput, error) {
	views, err := h.svc.ListAllOrgs(ctx, in.IncludeDeleted)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list orgs", err)
	}
	out := &adminOrgListOutput{}
	out.Body.Orgs = make([]adminOrgListItem, 0, len(views))
	for _, v := range views {
		out.Body.Orgs = append(out.Body.Orgs, adminOrgListItem{Org: *v.Org, MemberCount: v.MemberCount})
	}
	return out, nil
}

func (h *Handler) listInvitesAdmin(ctx context.Context, in *adminInviteListInput) (*adminInviteListOutput, error) {
	onlyPending := !in.IncludeAccepted
	invs, err := h.svc.ListInvites(ctx, in.OrgID, onlyPending)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list invites", err)
	}
	out := &adminInviteListOutput{}
	out.Body.Invites = invs
	return out, nil
}

func (h *Handler) revokeInviteAdmin(ctx context.Context, in *adminRevokeInviteInput) (*struct{}, error) {
	if err := h.svc.RevokeInvite(ctx, in.OrgID, in.InviteID); err != nil {
		return nil, huma.Error400BadRequest("revoke failed: " + err.Error())
	}
	return &struct{}{}, nil
}
