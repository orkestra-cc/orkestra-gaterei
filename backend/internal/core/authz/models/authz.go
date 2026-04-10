package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Permission is the unit of authorization. Modules declare permissions at
// boot via Module.Permissions() and the registry upserts them here. Admins
// bind permissions to roles; users inherit them through role bindings.
type Permission struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	Key         string             `bson:"key" json:"key"`
	Module      string             `bson:"module" json:"module"`
	Description string             `bson:"description" json:"description"`
	System      bool               `bson:"system" json:"system"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
}

// Role is a named bag of permissions. System roles (IsSystem=true) are
// seeded on first boot of the authz module; custom roles are created by
// org administrators and scoped to a specific orgId. System roles have
// an empty orgId.
type Role struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	UUID        string             `bson:"uuid" json:"id"`
	OrgID       string             `bson:"orgId" json:"orgId"` // empty for system roles
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Permissions []string           `bson:"permissions" json:"permissions"`
	IsSystem    bool               `bson:"isSystem" json:"isSystem"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// Binding grants a role to a user in a specific org (or globally when
// OrgID is empty — used for system role bindings derived from the user's
// system role). Optional ExpiresAt supports contractor/trial grants and is
// auto-reaped by a TTL index.
type Binding struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	UUID      string             `bson:"uuid" json:"id"`
	UserUUID  string             `bson:"userUUID" json:"userUUID"`
	OrgID     string             `bson:"orgId" json:"orgId"` // empty for system-level grants
	RoleUUID  string             `bson:"roleId" json:"roleId"`
	RoleName  string             `bson:"roleName" json:"roleName"`
	GrantedBy string             `bson:"grantedBy,omitempty" json:"grantedBy,omitempty"`
	GrantedAt time.Time          `bson:"grantedAt" json:"grantedAt"`
	ExpiresAt *time.Time         `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
}

// --- DTOs ---

type CreateRoleInput struct {
	Name        string   `json:"name" validate:"required,min=1,max=80"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions" validate:"required,min=1"`
}

type UpdateRoleInput struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

type CreateBindingInput struct {
	UserUUID  string     `json:"userUUID" validate:"required"`
	RoleUUID  string     `json:"roleId" validate:"required"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type PermissionCatalogResponse struct {
	Permissions []Permission `json:"permissions"`
}

type RoleListResponse struct {
	Roles []Role `json:"roles"`
}

type BindingListResponse struct {
	Bindings []Binding `json:"bindings"`
}

type EffectivePermissionsResponse struct {
	OrgID       string   `json:"orgId"`
	Permissions []string `json:"permissions"`
	SystemRole  string   `json:"systemRole"`
}
