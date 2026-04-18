// tenant_roles.cedar — placeholder for the tenant-level role catalog.
//
// Phase 2 introduces proper tenant-scoped roles (org_owner, org_admin,
// org_member, org_viewer, org_billing). Until then, tenant-level grants
// flow through the Mongo role table that authz.Service reads. This file
// encodes only the minimal shape so policy unit tests have something to
// exercise — refine in Phase 2.
//
// Principal attribute used here: tenant_roles — a Set<String> listing
// the role names the user holds in the acting tenant. The Cedar engine
// layer populates it from iface.TenantMembership.Roles.

// A tenant member with the "manager" role gets read + create + update
// actions on their tenant. Delete and admin actions are not granted.
// Matches the manager filter in authz/services/service.go::SeedSystemRoles.
@id("tenant_roles.manager.read_write_noop_delete")
permit (
    principal,
    action,
    resource
) when {
    principal has tenant_roles &&
    principal.tenant_roles.contains("manager") &&
    resource is Orkestra::Tenant &&
    context has action_suffix &&
    context.action_suffix != "delete" &&
    context.action_suffix != "admin"
};

// Operator: read-only + self-service actions on their tenant.
@id("tenant_roles.operator.read_self")
permit (
    principal,
    action,
    resource
) when {
    principal has tenant_roles &&
    principal.tenant_roles.contains("operator") &&
    resource is Orkestra::Tenant &&
    context has action_suffix &&
    (context.action_suffix == "read" ||
     context.action_suffix == "view" ||
     context.action_suffix == "self")
};

// Guest: read-only on their tenant.
@id("tenant_roles.guest.read_only")
permit (
    principal,
    action,
    resource
) when {
    principal has tenant_roles &&
    principal.tenant_roles.contains("guest") &&
    resource is Orkestra::Tenant &&
    context has action_suffix &&
    (context.action_suffix == "read" || context.action_suffix == "view")
};

// Holding "administrator" as a tenant-level role (legacy shape where
// bindings name "administrator" inside a tenant) grants every action
// on that tenant — still subject to tenant_scope.cedar forbid rules.
@id("tenant_roles.administrator.all_in_tenant")
permit (
    principal,
    action,
    resource
) when {
    principal has tenant_roles &&
    principal.tenant_roles.contains("administrator") &&
    resource is Orkestra::Tenant
};
