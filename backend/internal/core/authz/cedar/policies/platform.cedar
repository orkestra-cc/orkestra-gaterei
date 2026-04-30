// platform.cedar — system-role shortcuts.
//
// These policies grant broad permissions based on the user's platform-level
// system role (stored on User.system_role). They are the Cedar equivalent
// of the role-table shortcuts in authz/services/service.go: super_admin
// gets the wildcard, administrator inherits every registered action,
// developer is environment-gated.
//
// Every policy is annotated with @id so structured shadow-mode logs can
// report the matching policy id. Policy ids follow the scheme
//   <file>.<role_or_rule>.<variant>

// super_admin: wildcard. The canonical "god mode" role; no further checks.
@id("platform.super_admin.wildcard")
permit (
    principal,
    action,
    resource
) when {
    principal has system_role && principal.system_role == "super_admin"
};

// administrator: full access to every declared action. Tier scoping is
// enforced separately by tenant_scope.cedar forbid rules.
@id("platform.administrator.all")
permit (
    principal,
    action,
    resource
) when {
    principal has system_role && principal.system_role == "administrator"
};

// developer: full access in non-production environments (dev, staging).
// Environment is carried on the request Context so one deployed binary
// per environment can produce different developer privilege in a way that
// matches the existing role-seeder logic.
@id("platform.developer.nonprod")
permit (
    principal,
    action,
    resource
) when {
    principal has system_role && principal.system_role == "developer" &&
    context has env && context.env != "production"
};

// developer: production downgrade — read-only system actions only.
// Mirrors authz.Service.GetEffectivePermissions's developer-in-prod
// filter: keys ending in .read, .view, or .self are permitted; all
// other actions fall through to forbid.
@id("platform.developer.prod_readonly")
permit (
    principal,
    action,
    resource
) when {
    principal has system_role && principal.system_role == "developer" &&
    context has env && context.env == "production" &&
    context has action_suffix && (
        context.action_suffix == "read" ||
        context.action_suffix == "view" ||
        context.action_suffix == "self"
    )
};
