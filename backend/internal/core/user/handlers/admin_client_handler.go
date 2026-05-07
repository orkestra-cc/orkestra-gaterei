package handlers

import (
	"context"
	"errors"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/orkestra/backend/internal/core/user/models"
	"github.com/orkestra/backend/internal/core/user/services"
	"github.com/orkestra/backend/internal/shared/iface"
	"github.com/orkestra/backend/internal/shared/module"
)

// AdminClientUserHandler powers the admin "Clients" page — a list of
// client_users rows joined with each user's tenant memberships.
//
// Tenant memberships are fetched lazily via the service registry rather
// than injected at construction time because the user module initialises
// before tenant (tenant depends on user). At request time both modules
// are wired, so the lookup always succeeds in a real boot. When tenant is
// disabled at runtime the lookup returns nil and Memberships is reported
// as an empty array — the admin UI handles unattached users the same way.
type AdminClientUserHandler struct {
	clientUserService services.UserService
	services          *module.ServiceRegistry
}

// NewAdminClientUserHandler wires the handler with the client-tier user
// service and a reference to the module service registry for the lazy
// tenant-provider lookup.
func NewAdminClientUserHandler(clientUserService services.UserService, services *module.ServiceRegistry) *AdminClientUserHandler {
	return &AdminClientUserHandler{
		clientUserService: clientUserService,
		services:          services,
	}
}

// ListClientUsersAdminRequest mirrors the existing /v1/users filter set.
type ListClientUsersAdminRequest struct {
	Role          string `query:"role" doc:"Filter by user role"`
	IsActive      bool   `query:"isActive" doc:"Filter by active status"`
	EmailVerified bool   `query:"emailVerified" doc:"Filter by email verification status"`
	Search        string `query:"search" doc:"Search in name, email, username"`
	Page          int    `query:"page" default:"1" minimum:"1" doc:"Page number"`
	PageSize      int    `query:"pageSize" default:"50" minimum:"1" maximum:"200" doc:"Number of items per page"`
}

// ListClientUsersAdminResponse wraps the paginated payload in Huma's body
// envelope.
type ListClientUsersAdminResponse struct {
	Body models.AdminClientUserListResponse `json:"body"`
}

// ListClientUsersAdmin handles GET /v1/admin/client-users.
func (h *AdminClientUserHandler) ListClientUsersAdmin(ctx context.Context, req *ListClientUsersAdminRequest) (*ListClientUsersAdminResponse, error) {
	filters := &models.UserFilters{
		Role:   req.Role,
		Search: req.Search,
	}
	if req.IsActive {
		v := req.IsActive
		filters.IsActive = &v
	}
	if req.EmailVerified {
		v := req.EmailVerified
		filters.EmailVerified = &v
	}

	pagination := &models.PaginationParams{Page: req.Page, PageSize: req.PageSize}

	page, err := h.clientUserService.ListUsers(ctx, filters, pagination)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to list client users", err)
	}

	tenantProv, _ := module.GetTyped[iface.TenantProvider](h.services, module.ServiceTenantProvider)

	out := make([]models.AdminClientUserItem, 0, len(page.Users))
	for i := range page.Users {
		u := page.Users[i]
		item := models.AdminClientUserItem{
			ID:            u.ID,
			Email:         u.Email,
			Username:      u.Username,
			FullName:      u.FullName,
			Avatar:        u.Avatar,
			Role:          u.Role,
			IsActive:      u.IsActive,
			EmailVerified: u.EmailVerified,
			LastLogin:     u.LastLogin,
			CreatedAt:     u.CreatedAt,
			Memberships:   []models.AdminUserMembership{},
		}

		if tenantProv != nil {
			memberships, mErr := tenantProv.ListUserMemberships(ctx, u.ID)
			if mErr != nil {
				// Don't fail the whole list because one user's membership
				// fetch errored — log and continue with an empty array.
				slog.WarnContext(ctx, "admin client-users: list memberships failed",
					"userId", u.ID, "error", mErr)
			} else {
				item.Memberships = make([]models.AdminUserMembership, 0, len(memberships))
				for _, m := range memberships {
					item.Memberships = append(item.Memberships, models.AdminUserMembership{
						TenantUUID: m.TenantUUID,
						TenantName: m.TenantName,
						TenantSlug: m.TenantSlug,
						TenantKind: m.TenantKind,
						Roles:      m.Roles,
						IsOwner:    m.IsOwner,
					})
				}
			}
		}

		out = append(out, item)
	}

	return &ListClientUsersAdminResponse{
		Body: models.AdminClientUserListResponse{
			Users:      out,
			Total:      page.Total,
			Page:       page.Page,
			PageSize:   page.PageSize,
			TotalPages: page.TotalPages,
		},
	}, nil
}

// buildAdminItem fetches a client user by id and joins its tenant
// memberships. Shared by GetClientUserAdmin and the create / update
// response paths so the detail page sees the same shape as the list.
func (h *AdminClientUserHandler) buildAdminItem(ctx context.Context, id string) (*models.AdminClientUserItem, error) {
	resp, err := h.clientUserService.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	item := models.AdminClientUserItem{
		ID:            resp.ID,
		Email:         resp.Email,
		Username:      resp.Username,
		FullName:      resp.FullName,
		Avatar:        resp.Avatar,
		Role:          resp.Role,
		IsActive:      resp.IsActive,
		EmailVerified: resp.EmailVerified,
		LastLogin:     resp.LastLogin,
		CreatedAt:     resp.CreatedAt,
		Memberships:   []models.AdminUserMembership{},
		Providers:     resp.Providers,
	}

	if tenantProv, ok := module.GetTyped[iface.TenantProvider](h.services, module.ServiceTenantProvider); ok && tenantProv != nil {
		memberships, mErr := tenantProv.ListUserMemberships(ctx, resp.ID)
		if mErr != nil {
			slog.WarnContext(ctx, "admin client-user: list memberships failed",
				"userId", resp.ID, "error", mErr)
		} else {
			item.Memberships = make([]models.AdminUserMembership, 0, len(memberships))
			for _, m := range memberships {
				item.Memberships = append(item.Memberships, models.AdminUserMembership{
					TenantUUID: m.TenantUUID,
					TenantName: m.TenantName,
					TenantSlug: m.TenantSlug,
					TenantKind: m.TenantKind,
					Roles:      m.Roles,
					IsOwner:    m.IsOwner,
				})
			}
		}
	}
	return &item, nil
}

// GetClientUserAdminRequest mirrors the path-only shape Huma expects.
type GetClientUserAdminRequest struct {
	ID string `path:"id" doc:"Client user UUID"`
}

// GetClientUserAdminResponse wraps a single AdminClientUserItem.
type GetClientUserAdminResponse struct {
	Body models.AdminClientUserItem `json:"body"`
}

// GetClientUserAdmin handles GET /v1/admin/client-users/{id}.
func (h *AdminClientUserHandler) GetClientUserAdmin(ctx context.Context, req *GetClientUserAdminRequest) (*GetClientUserAdminResponse, error) {
	item, err := h.buildAdminItem(ctx, req.ID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound("Client user not found", err)
		}
		if errors.Is(err, services.ErrInvalidInput) {
			return nil, huma.Error400BadRequest("Invalid user id", err)
		}
		return nil, huma.Error500InternalServerError("Failed to load client user", err)
	}
	return &GetClientUserAdminResponse{Body: *item}, nil
}

// UpdateClientUserAdminBody is the slim mutation payload — only the
// fields that an admin would reasonably change on a client user. Driver
// document fields are intentionally omitted; they are managed by the
// user themselves.
type UpdateClientUserAdminBody struct {
	FullName string `json:"fullName,omitempty" validate:"omitempty,min=1,max=100"`
	Username string `json:"username,omitempty" validate:"omitempty,min=3,max=50"`
	Email    string `json:"email,omitempty" validate:"omitempty,email"`
	Phone    string `json:"phone,omitempty" validate:"omitempty,e164"`
	Role     string `json:"role,omitempty" validate:"omitempty,oneof=super_admin administrator developer manager operator guest"`
	IsActive *bool  `json:"isActive,omitempty"`
}

// UpdateClientUserAdminRequest combines the path id and the patch body.
type UpdateClientUserAdminRequest struct {
	ID   string                    `path:"id" doc:"Client user UUID"`
	Body UpdateClientUserAdminBody `json:"body"`
}

// UpdateClientUserAdminResponse echoes the freshly joined item.
type UpdateClientUserAdminResponse struct {
	Body models.AdminClientUserItem `json:"body"`
}

// UpdateClientUserAdmin handles PATCH /v1/admin/client-users/{id}.
func (h *AdminClientUserHandler) UpdateClientUserAdmin(ctx context.Context, req *UpdateClientUserAdminRequest) (*UpdateClientUserAdminResponse, error) {
	input := &models.UpdateUserInput{
		FullName: req.Body.FullName,
		Username: req.Body.Username,
		Email:    req.Body.Email,
		Phone:    req.Body.Phone,
		Role:     req.Body.Role,
		IsActive: req.Body.IsActive,
	}
	if _, err := h.clientUserService.UpdateUser(ctx, req.ID, input); err != nil {
		switch {
		case errors.Is(err, services.ErrUserNotFound):
			return nil, huma.Error404NotFound("Client user not found", err)
		case errors.Is(err, services.ErrEmailNotUnique):
			return nil, huma.Error409Conflict("Email already in use", err)
		case errors.Is(err, services.ErrInvalidInput):
			return nil, huma.Error400BadRequest("Invalid input", err)
		default:
			return nil, huma.Error500InternalServerError("Failed to update client user", err)
		}
	}

	item, err := h.buildAdminItem(ctx, req.ID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to reload client user", err)
	}
	return &UpdateClientUserAdminResponse{Body: *item}, nil
}

// DeleteClientUserAdminRequest is path-only.
type DeleteClientUserAdminRequest struct {
	ID string `path:"id" doc:"Client user UUID"`
}

// DeleteClientUserAdminResponse returns a confirmation message.
type DeleteClientUserAdminResponse struct {
	Body struct {
		Message string `json:"message"`
	}
}

// DeleteClientUserAdmin handles DELETE /v1/admin/client-users/{id}. Uses
// SoftDeleteAndAliasEmail so the freed email can be reused for a fresh
// signup — Tier-2 client emails are intentionally aliased, unlike
// operator-tier soft deletes which preserve the email for audit.
func (h *AdminClientUserHandler) DeleteClientUserAdmin(ctx context.Context, req *DeleteClientUserAdminRequest) (*DeleteClientUserAdminResponse, error) {
	if err := h.clientUserService.SoftDeleteAndAliasEmail(ctx, req.ID); err != nil {
		if errors.Is(err, services.ErrInvalidInput) {
			return nil, huma.Error400BadRequest("Invalid user id", err)
		}
		return nil, huma.Error500InternalServerError("Failed to delete client user", err)
	}
	out := &DeleteClientUserAdminResponse{}
	out.Body.Message = "Client user deleted"
	return out, nil
}

// CreateClientUserAdminBody is the admin-direct create payload. The new
// user is pre-verified (admin vouched for the address) and active.
type CreateClientUserAdminBody struct {
	Email    string `json:"email" validate:"required,email"`
	FullName string `json:"fullName" validate:"required,min=1,max=100"`
	Username string `json:"username,omitempty" validate:"omitempty,min=3,max=50"`
	Phone    string `json:"phone,omitempty" validate:"omitempty,e164"`
	Role     string `json:"role" validate:"required,oneof=super_admin administrator developer manager operator guest"`
	Password string `json:"password" validate:"required,min=10,max=128" doc:"Initial password — admin should share securely; user can change after first login"`
}

// CreateClientUserAdminRequest carries the body.
type CreateClientUserAdminRequest struct {
	Body CreateClientUserAdminBody `json:"body"`
}

// CreateClientUserAdminResponse echoes the created item.
type CreateClientUserAdminResponse struct {
	Body models.AdminClientUserItem `json:"body"`
}

// CreateClientUserAdmin handles POST /v1/admin/client-users. Pre-hashes
// the password against the live policy, then inserts the new client_users
// row with EmailVerified=true so the new user can log in immediately.
func (h *AdminClientUserHandler) CreateClientUserAdmin(ctx context.Context, req *CreateClientUserAdminRequest) (*CreateClientUserAdminResponse, error) {
	hasher, ok := module.GetTyped[iface.PasswordHasher](h.services, module.ServicePasswordService)
	if !ok || hasher == nil {
		return nil, huma.Error503ServiceUnavailable("Password service unavailable")
	}
	if err := hasher.ValidatePolicy(ctx, req.Body.Password, req.Body.Email); err != nil {
		return nil, huma.Error400BadRequest("Password does not meet policy", err)
	}
	hash, err := hasher.Hash(req.Body.Password)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to hash password", err)
	}

	input := &models.CreateUserInput{
		Email:        req.Body.Email,
		FullName:     req.Body.FullName,
		Username:     req.Body.Username,
		Phone:        req.Body.Phone,
		Role:         req.Body.Role,
		PasswordHash: hash,
	}
	created, err := h.clientUserService.CreateUserWithPassword(ctx, input)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrEmailNotUnique):
			return nil, huma.Error409Conflict("Email already in use", err)
		case errors.Is(err, services.ErrInvalidInput):
			return nil, huma.Error400BadRequest("Invalid input", err)
		default:
			return nil, huma.Error500InternalServerError("Failed to create client user", err)
		}
	}
	if mErr := h.clientUserService.MarkEmailVerified(ctx, created.UUID); mErr != nil {
		slog.WarnContext(ctx, "admin client-user: mark email verified failed",
			"userId", created.UUID, "error", mErr)
	}

	item, err := h.buildAdminItem(ctx, created.UUID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to load created client user", err)
	}
	return &CreateClientUserAdminResponse{Body: *item}, nil
}
