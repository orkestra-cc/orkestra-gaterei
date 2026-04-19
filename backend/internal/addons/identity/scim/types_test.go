package scim

import (
	"encoding/json"
	"testing"
)

// TestEmptyList shape-checks the SCIM ListResponse envelope we return
// on /Users and /Groups. SCIM clients require both `schemas` and
// `Resources` to be present even when the list is empty — Okta's
// provisioning agent, for example, crashes when Resources is omitted
// rather than rendered as [].
func TestEmptyList(t *testing.T) {
	t.Parallel()

	body, err := json.Marshal(EmptyList())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := out["Resources"]; !ok {
		t.Fatalf("Resources key missing — SCIM clients require it on empty lists. body=%s", body)
	}
	if resources, _ := out["Resources"].([]any); resources == nil {
		t.Fatalf("Resources must be an empty array not null. body=%s", body)
	}
	schemas, _ := out["schemas"].([]any)
	if len(schemas) != 1 || schemas[0] != SchemaListResponse {
		t.Fatalf("schemas must be [%q]; got %v", SchemaListResponse, schemas)
	}
}

// TestNewError verifies the RFC-7644 error envelope. `status` is a
// string (not an int) and the SCIM schema URI is mandatory — clients
// pattern-match on both.
func TestNewError(t *testing.T) {
	t.Parallel()

	e := NewError("501", "not implemented")
	body, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s, _ := out["status"].(string); s != "501" {
		t.Fatalf("status must be a string; got %v (%T)", out["status"], out["status"])
	}
	schemas, _ := out["schemas"].([]any)
	if len(schemas) != 1 || schemas[0] != SchemaError {
		t.Fatalf("schemas must be [%q]; got %v", SchemaError, schemas)
	}
	if out["detail"] != "not implemented" {
		t.Fatalf("detail mismatch: %v", out["detail"])
	}
}
