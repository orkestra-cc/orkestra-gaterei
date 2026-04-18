// capability_grants.cedar — Phase 2 stub.
//
// Phase 2 introduces Capability entities and the event-sourced
// tenant_entitlements projection driven by Stripe webhooks. When that
// ships, this file will contain:
//
//   - permit rules that grant an external tenant access to specific
//     capabilities based on the presence of an active subscription
//     whose product.grants include that capability id;
//   - forbid rules that reject capability access when the tenant is
//     suspended, the entitlement has expired, or the usage quota is
//     depleted.
//
// Intentionally empty today so the shadow-mode evaluator never grants
// capability-based permits by accident. Leave this file in place (rather
// than deleting the marker) so Phase 2 can flip it on with a single PR.
