// Package marketing is the Orkestra marketing addon — contact base
// (Organizations, Persons, Memberships, Tags, Custom Field Schemas),
// importer pipeline, and (in future phases) immutable activity log,
// multi-profile scoring engine, and card/membership lifecycle.
//
// Phase 1 (Fondazione anagrafica MVP) ships the data layer + (in
// upcoming PRs) handlers and a CSV importer. The design lives at
// docs/plans/marketing-addon/Orkestra_marketing_addon.md and the
// per-phase implementation plan at
// docs/plans/marketing-addon/IMPLEMENTATION_PLAN.md in the orkestra
// monorepo.
package marketing

import (
	"log/slog"

	"github.com/orkestra-cc/orkestra-addon-marketing/models"
	"github.com/orkestra-cc/orkestra-sdk/module"
)

// MarketingModule implements the Orkestra SDK Module interface for the
// marketing addon. Phase 1 ships only the shell — subsequent PRs on
// feature/marketing-addon wire collections, handlers, services, and the
// CSV importer.
type MarketingModule struct {
	module.BaseModule

	logger *slog.Logger
}

// NewModule returns a new instance. The registry calls this once per
// boot from cmd/server/catalog_marketing.go (and from any external
// host that wires the addon via its public mirror).
func NewModule() *MarketingModule { return &MarketingModule{} }

// Name is the stable identifier used in module_configs, route gating,
// and ORKESTRA_PROFILE addon lists. Never rename without a coordinated
// data migration.
func (m *MarketingModule) Name() string { return "marketing" }

// DisplayName is shown in the /admin/modules UI. Italian-friendly
// cognate so it sits well next to the other localized module labels
// in the operator console (e.g. "Sottoscrizioni" for subscriptions).
func (m *MarketingModule) DisplayName() string { return "Marketing" }

// Description gives the operator a one-line summary on hover/expand
// in /admin/modules.
func (m *MarketingModule) Description() string {
	return "Anagrafica contatti, importer multi-sorgente, storico attività, scoring multi-profilo e card/membership"
}

// Category marks this module as toggleable — enabled/disabled from the
// admin UI without external infra. All optional addons in orkestra
// share this category.
func (m *MarketingModule) Category() module.ModuleCategory {
	return module.CategoryToggleable
}

// Enabled returns the first-boot default state when no module_configs
// document exists yet. False keeps marketing opt-in everywhere except
// the enterprise SKU, where the "*" entry in profileAddons
// (pkg/sdk/module/config_service.go) pre-enables every optional
// addon on first boot.
func (m *MarketingModule) Enabled() bool { return false }

// Dependencies is empty in Phase 1 — the contact base does not consume
// any other module. Phase 2 (scoring) and beyond may add optional deps
// (e.g. notification for campaign delivery) via ServiceRegistry
// lookups rather than hard Dependencies entries.
func (m *MarketingModule) Dependencies() []string { return nil }

// Collections returns the MongoDB collections this module owns,
// declaring the indexes the registry creates at boot. Index
// declarations follow the schemas at
// docs/plans/marketing-addon/schemas/ — adjust both together when
// fields evolve.
//
// Limitation: the SDK's IndexSpec does not expose
// PartialFilterExpression, so the schema's "unique partial" entries
// on marketing_memberships (one Active=true per (person,org) pair,
// one Active+Primary=true per person) cannot be enforced at the
// index level today. Service-layer helpers in
// repository/membership_repo.go (UnsetPrimaryForPerson, Close)
// preserve the invariant under the typical mutation flows; widening
// the SDK to support partial filters is tracked as a Phase-2+
// follow-up.
func (m *MarketingModule) Collections() []module.CollectionSpec {
	return []module.CollectionSpec{
		{Name: models.OrganizationsCollection, Indexes: []module.IndexSpec{
			{Keys: map[string]int{"uuid": 1}, Unique: true},
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "vat", Direction: 1},
			}, Unique: true, Sparse: true},
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "taxCode", Direction: 1},
			}, Unique: true, Sparse: true},
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "kind", Direction: 1},
			}},
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "tags", Direction: 1},
			}},
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "updatedAt", Direction: -1},
			}},
		}},
		{Name: models.PersonsCollection, Indexes: []module.IndexSpec{
			{Keys: map[string]int{"uuid": 1}, Unique: true},
			// Multikey unique sparse on emails.address — every email
			// is unique per tenant regardless of which entry carries
			// it. Sparse so persons without emails don't conflict.
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "emails.address", Direction: 1},
			}, Unique: true, Sparse: true},
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "tags", Direction: 1},
			}},
			// Index lands now even though Phase 4 populates the
			// field — avoids a write-blocking index build later.
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "activeCardUuids", Direction: 1},
			}, Sparse: true},
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "updatedAt", Direction: -1},
			}},
		}},
		{Name: models.MembershipsCollection, Indexes: []module.IndexSpec{
			{Keys: map[string]int{"uuid": 1}, Unique: true},
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "personUuid", Direction: 1},
			}},
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "orgUuid", Direction: 1},
			}},
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "personUuid", Direction: 1},
				{Field: "orgUuid", Direction: 1},
			}},
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "personUuid", Direction: 1},
				{Field: "primary", Direction: 1},
			}},
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "active", Direction: 1},
			}},
		}},
		{Name: models.TagsCollection, Indexes: []module.IndexSpec{
			{Keys: map[string]int{"uuid": 1}, Unique: true},
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "slug", Direction: 1},
			}, Unique: true},
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "parentUuid", Direction: 1},
			}},
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "path", Direction: 1},
			}},
		}},
		{Name: models.CustomFieldSchemasCollection, Indexes: []module.IndexSpec{
			{Keys: map[string]int{"uuid": 1}, Unique: true},
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "targetCollection", Direction: 1},
			}, Unique: true},
		}},
	}
}

// Init is the lifecycle hook the registry calls after all dependencies
// have been initialized. Phase 1 keeps it minimal — the logger is
// stashed so subsequent phase work can wire repositories, services,
// and handlers without changing this signature.
func (m *MarketingModule) Init(deps *module.Dependencies) error {
	m.logger = deps.Logger
	m.logger.Info("Marketing module initialized (phase 1 scaffold)")
	return nil
}
