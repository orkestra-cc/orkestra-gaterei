package errcode

// Code constants. Every code is <module>.<situation> in snake_case;
// the module owns its namespace, the situation names the failure
// semantically (not the HTTP status). Adding a code is additive — the
// SPA falls back to `detail` when an unknown code arrives — but
// renaming or removing one is a wire-contract break. codes_test.go
// pins every const here against a golden snapshot so a silent rename
// fails CI loudly.
//
// Grouping below is by module to make it easy to scan ownership.
// Insert new entries inside the matching block; one-off codes that
// don't belong to a module (rare) go at the bottom.

// --- auth ---

// AuthEmailInUse signals that a sign-up or invite was rejected because
// the email already maps to a live user in this audience tier. 409.
const AuthEmailInUse = "auth.email_in_use"
