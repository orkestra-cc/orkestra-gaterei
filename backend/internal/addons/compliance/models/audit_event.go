// Package models defines the audit-event record that powers SOC2 evidence
// collection, GDPR data-subject audit trails, and the platform admin's
// audit view. Phase 4.1 lands the append-only write path and the admin read
// surface; later Phase-4 commits build DSR exports and retention policy on
// top of this same collection.
package models

import "time"

// AuditEventsCollection is the MongoDB collection backing the append-only
// audit trail. One document per emitted event.
const AuditEventsCollection = "compliance_audit_events"

// ActorType classifies who caused the event. Platform-scoped events without
// an authenticated principal (cron jobs, webhooks) are stamped "system".
// Anonymous public endpoints (signup, OIDC start) are stamped "anonymous".
const (
	ActorTypeUser      = "user"
	ActorTypeSystem    = "system"
	ActorTypeAnonymous = "anonymous"
)

// Outcome is the coarse-grained result of the audited action. Distinct from
// HTTP status — a request that reached the handler and was denied by an
// authz gate is "denied", not "failure". "failure" is reserved for errors
// that happened after authorization passed.
const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"
	OutcomeDenied  = "denied"
)

// AuditEvent is the persistent shape of a single audit trail row. Fields
// are denormalized on purpose so the SOC2 evidence queries don't require
// joins — retention is enforced via a TTL index on Timestamp.
type AuditEvent struct {
	UUID         string         `bson:"uuid" json:"uuid"`
	TenantID     string         `bson:"tenantId,omitempty" json:"tenantId,omitempty"`
	TenantKind   string         `bson:"tenantKind,omitempty" json:"tenantKind,omitempty"`
	ActorUserID  string         `bson:"actorUserId,omitempty" json:"actorUserId,omitempty"`
	ActorEmail   string         `bson:"actorEmail,omitempty" json:"actorEmail,omitempty"`
	ActorType    string         `bson:"actorType" json:"actorType"`
	Action       string         `bson:"action" json:"action"`
	ResourceType string         `bson:"resourceType,omitempty" json:"resourceType,omitempty"`
	ResourceID   string         `bson:"resourceId,omitempty" json:"resourceId,omitempty"`
	Outcome      string         `bson:"outcome" json:"outcome"`
	IPAddress    string         `bson:"ipAddress,omitempty" json:"ipAddress,omitempty"`
	UserAgent    string         `bson:"userAgent,omitempty" json:"userAgent,omitempty"`
	Metadata     map[string]any `bson:"metadata,omitempty" json:"metadata,omitempty"`
	Timestamp    time.Time      `bson:"timestamp" json:"timestamp"`
}

// Action constants — the vocabulary the emitter and reader agree on. Kept
// dotted hierarchy-style so filtering by prefix picks up an action family.
const (
	// auth.*
	ActionAuthLoginSucceeded     = "auth.login.succeeded"
	ActionAuthLoginFailed        = "auth.login.failed"
	ActionAuthLogout             = "auth.logout"
	ActionAuthPasswordChanged    = "auth.password.changed"
	ActionAuthPasswordResetStart = "auth.password.reset_requested"
	ActionAuthPasswordResetDone  = "auth.password.reset_completed"
	ActionAuthEmailVerified      = "auth.email.verified"
	ActionAuthMFAEnrolled        = "auth.mfa.enrolled"
	ActionAuthMFAVerified        = "auth.mfa.verified"
	ActionAuthMFARemoved         = "auth.mfa.removed"
	ActionAuthMFAReset           = "auth.mfa.reset"
	// admin-on-user auth surfaces — pair with the four /v1/admin/users/{id}/*
	// endpoints owned by the auth module. authService.recordAuthEvent
	// maps its internal event-types onto these via
	// authEventComplianceAction.
	ActionAuthEmailVerifyResend = "auth.email.verify_resend"
	ActionAuthOAuthUnlinked     = "auth.oauth.unlinked"
	// self-service auth surfaces — same audit lane, actor==target.
	ActionAuthOAuthUnlinkedSelf  = "auth.oauth.unlinked.self"
	ActionAuthOAuthLinkedSelf    = "auth.oauth.linked.self"
	ActionAuthSessionRevokedSelf = "auth.session.revoked.self"
	ActionAuthSessionRevokedAll  = "auth.session.revoked_all.self"

	// tenant.*
	ActionTenantProvisioned = "tenant.lifecycle.provisioned"
	ActionTenantActivated   = "tenant.lifecycle.activated"
	ActionTenantSuspended   = "tenant.lifecycle.suspended"
	ActionTenantArchived    = "tenant.lifecycle.archived"
	ActionTenantPurged      = "tenant.lifecycle.purged"
	ActionTenantUpdated     = "tenant.updated"
	ActionTenantDeleted     = "tenant.deleted"
	ActionTenantPlanChanged = "tenant.plan.changed"

	// membership.*
	ActionMembershipInvited = "tenant.membership.invited"
	ActionMembershipJoined  = "tenant.membership.joined"
	ActionMembershipRemoved = "tenant.membership.removed"

	// identity.*
	ActionIdentityIdPCreated  = "identity.idp.created"
	ActionIdentityIdPUpdated  = "identity.idp.updated"
	ActionIdentityIdPDeleted  = "identity.idp.deleted"
	ActionIdentityOIDCLogin   = "identity.oidc.login"
	ActionIdentitySCIMRotated = "identity.scim.token_rotated"

	// onboarding.*
	ActionOnboardingRegistered = "onboarding.register.completed"

	// user.* — operator-tier admin lifecycle actions on the /admin/users
	// surface (and the symmetric tier-2 admin/clients surface, where the
	// resource type discriminates operator vs client). All emitted from
	// the user service; the actor is the admin performing the operation.
	// `*.refused` variants exist so the backend guards (self-delete,
	// last-administrator quorum) leave an audit row even when the
	// destructive call is rejected — SOC2 wants to see "an admin tried
	// to lock the platform out" as much as it wants to see the
	// successful changes.
	ActionUserDeleted       = "user.deleted"
	ActionUserDeleteRefused = "user.delete.refused"
	ActionUserActivated     = "user.activated"
	ActionUserDeactivated   = "user.deactivated"
	ActionUserRoleChanged   = "user.role.changed"
	ActionUserUpdateRefused = "user.update.refused"
	// user.create.refused covers the role-escalation guard on
	// POST /v1/users — an admin trying to seed a higher-tier role at
	// create time. Wire code in metadata is
	// errcode.UserRoleEscalationForbidden.
	ActionUserCreateRefused = "user.create.refused"
)
