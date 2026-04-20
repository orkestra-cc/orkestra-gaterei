// Package migrations holds one-shot data migrations for the subscriptions
// module. They are invoked from the module's Start() when a dedicated env
// flag is set; a completion row in subscriptions_migrations prevents
// re-execution on the next boot.
//
// Each migration is idempotent — re-running mid-flight (crash, restart)
// picks up where it left off because the filter predicate only selects
// rows that still need work.
package migrations

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	pmtmodels "github.com/orkestra/backend/internal/addons/payments/models"
	pmtrepo "github.com/orkestra/backend/internal/addons/payments/repository"
	"github.com/orkestra/backend/internal/addons/subscriptions/repository"
	"github.com/orkestra/backend/internal/shared/iface"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// BackfillName is the stable identifier used both as the _id of the
// completion row and as a log field, so operators can grep migration
// traces across reruns.
const BackfillName = "0002_backfill_tenant_uuid"

// MigrationsCollection holds completion rows for the subscriptions
// module's data migrations. Kept independent of module_configs so the
// module's admin config surface isn't polluted with migration plumbing.
const MigrationsCollection = "subscriptions_migrations"

// Default batch size. Tuned for typical Mongo deployments — small enough
// to avoid long transactions on the cold tail, large enough to amortize
// network round-trips.
const defaultBatch = 500

// Report is the aggregate outcome returned by Run, also persisted in
// the completion row so operators can audit the migration without
// re-reading logs.
type Report struct {
	StartedAt             time.Time `bson:"startedAt" json:"startedAt"`
	CompletedAt           time.Time `bson:"completedAt" json:"completedAt"`
	SubscriptionsScanned  int       `bson:"subscriptionsScanned" json:"subscriptionsScanned"`
	SubscriptionsUpdated  int       `bson:"subscriptionsUpdated" json:"subscriptionsUpdated"`
	TransactionsScanned   int       `bson:"transactionsScanned" json:"transactionsScanned"`
	TransactionsUpdated   int       `bson:"transactionsUpdated" json:"transactionsUpdated"`
	TenantsProvisioned    int       `bson:"tenantsProvisioned" json:"tenantsProvisioned"`
	ClientsBackStamped    int       `bson:"clientsBackStamped" json:"clientsBackStamped"`
}

// BackfillDeps groups the collaborators the migration needs. A nil
// TenantProvider aborts the run — the paired-tenant step cannot be
// skipped without leaving the data half-migrated.
type BackfillDeps struct {
	DB             *mongo.Database
	Subs           repository.SubscriptionRepository
	Clients        repository.ClientRepository
	Transactions   pmtrepo.TransactionRepository
	TenantProvider iface.TenantProvider
	// MigrationOwnerUUID is the user UUID recorded as the initial owner of
	// any paired external tenant lazily created by the backfill. Typically
	// a dedicated service account ("migration-bot") or the operator who
	// triggered the run.
	MigrationOwnerUUID string
	Logger             *slog.Logger
	// BatchSize overrides the default page size. 0 uses defaultBatch.
	BatchSize int64
}

// AlreadyRun reports whether BackfillName has a completion row in
// subscriptions_migrations. Callers should short-circuit in that case
// rather than paying the scan cost of a no-op run.
func AlreadyRun(ctx context.Context, db *mongo.Database) (bool, error) {
	err := db.Collection(MigrationsCollection).FindOne(ctx, bson.M{"_id": BackfillName}).Err()
	if err == nil {
		return true, nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return false, err
}

// Run executes the backfill. Safe to call repeatedly — rows already
// carrying TenantUUID are filtered out at the DB level, and paired-tenant
// provisioning is idempotent via the unique sparse index on
// tenant_orgs.metadata.legacyClientUUID.
func Run(ctx context.Context, d BackfillDeps) (*Report, error) {
	if d.TenantProvider == nil {
		return nil, errors.New("migrations: backfill requires TenantProvider")
	}
	if d.Subs == nil || d.Clients == nil || d.Transactions == nil || d.DB == nil {
		return nil, errors.New("migrations: backfill requires subs, clients, transactions and db")
	}
	if d.MigrationOwnerUUID == "" {
		return nil, errors.New("migrations: backfill requires MigrationOwnerUUID")
	}
	logger := d.Logger
	if logger == nil {
		logger = slog.Default()
	}
	batch := d.BatchSize
	if batch <= 0 {
		batch = defaultBatch
	}

	report := &Report{StartedAt: time.Now().UTC()}

	// Pass 1 — subscriptions. Resolve TenantUUID per row, back-stamping the
	// paired Client.OrgUUID when it is still empty so pass 2 and every
	// subsequent Create avoids the cross-module hop.
	tenantByClient := map[string]string{}
	for {
		rows, err := d.Subs.FindWithoutTenantUUID(ctx, batch)
		if err != nil {
			return report, fmt.Errorf("migrations: scan subscriptions: %w", err)
		}
		if len(rows) == 0 {
			break
		}
		report.SubscriptionsScanned += len(rows)

		progressed := 0
		for i := range rows {
			sub := rows[i]
			if sub.ClientUUID == "" {
				// Orphan row with neither tenantUUID nor clientUUID. Log and
				// skip so the operator can triage manually — we refuse to
				// guess at the owning tenant.
				logger.Warn("migrations: subscription has no ClientUUID, skipping",
					slog.String("migration", BackfillName),
					slog.String("subscriptionUUID", sub.UUID),
				)
				continue
			}
			tenantUUID, ok := tenantByClient[sub.ClientUUID]
			if !ok {
				resolved, provisioned, backStamped, err := resolveTenantForBackfill(ctx, d, sub.ClientUUID)
				if err != nil {
					logger.Warn("migrations: resolve tenant failed, leaving row untouched",
						slog.String("migration", BackfillName),
						slog.String("subscriptionUUID", sub.UUID),
						slog.String("clientUUID", sub.ClientUUID),
						slog.Any("error", err),
					)
					continue
				}
				if provisioned {
					report.TenantsProvisioned++
				}
				if backStamped {
					report.ClientsBackStamped++
				}
				tenantUUID = resolved
				tenantByClient[sub.ClientUUID] = tenantUUID
			}
			if tenantUUID == "" {
				continue
			}
			if err := d.Subs.SetTenantUUID(ctx, sub.UUID, tenantUUID); err != nil {
				logger.Warn("migrations: SetTenantUUID on subscription failed",
					slog.String("migration", BackfillName),
					slog.String("subscriptionUUID", sub.UUID),
					slog.Any("error", err),
				)
				continue
			}
			report.SubscriptionsUpdated++
			progressed++
		}
		// Safeguard against an unfixable tail (every row in the batch fails
		// to resolve). Without this we would loop forever because the filter
		// still matches them.
		if progressed == 0 {
			break
		}
	}

	// Pass 2 — transactions. Each transaction carries SubscriptionUUID; the
	// owning subscription now (after pass 1) carries TenantUUID, so a
	// single lookup pierces the legacy indirection.
	subCache := map[string]string{}
	for {
		rows, err := d.Transactions.FindWithoutTenantUUID(ctx, batch)
		if err != nil {
			return report, fmt.Errorf("migrations: scan transactions: %w", err)
		}
		if len(rows) == 0 {
			break
		}
		report.TransactionsScanned += len(rows)

		progressed := 0
		for i := range rows {
			tx := rows[i]
			tenantUUID, err := resolveTransactionTenant(ctx, d, &tx, subCache, tenantByClient)
			if err != nil {
				logger.Warn("migrations: resolve tenant for transaction failed",
					slog.String("migration", BackfillName),
					slog.String("transactionUUID", tx.UUID),
					slog.Any("error", err),
				)
				continue
			}
			if tenantUUID == "" {
				continue
			}
			if err := d.Transactions.SetTenantUUID(ctx, tx.UUID, tenantUUID); err != nil {
				logger.Warn("migrations: SetTenantUUID on transaction failed",
					slog.String("migration", BackfillName),
					slog.String("transactionUUID", tx.UUID),
					slog.Any("error", err),
				)
				continue
			}
			report.TransactionsUpdated++
			progressed++
		}
		if progressed == 0 {
			break
		}
	}

	report.CompletedAt = time.Now().UTC()

	if _, err := d.DB.Collection(MigrationsCollection).InsertOne(ctx, bson.M{
		"_id":                   BackfillName,
		"name":                  BackfillName,
		"startedAt":             report.StartedAt,
		"completedAt":           report.CompletedAt,
		"subscriptionsScanned":  report.SubscriptionsScanned,
		"subscriptionsUpdated":  report.SubscriptionsUpdated,
		"transactionsScanned":   report.TransactionsScanned,
		"transactionsUpdated":   report.TransactionsUpdated,
		"tenantsProvisioned":    report.TenantsProvisioned,
		"clientsBackStamped":    report.ClientsBackStamped,
	}); err != nil {
		// A duplicate _id means a concurrent runner beat us to it — that's
		// fine, treat as success. Any other write failure is logged but
		// non-fatal because the data changes above are already applied.
		if mongo.IsDuplicateKeyError(err) {
			logger.Info("migrations: completion row already present (concurrent run)",
				slog.String("migration", BackfillName))
		} else {
			logger.Warn("migrations: persist completion row failed",
				slog.String("migration", BackfillName),
				slog.Any("error", err),
			)
		}
	}

	logger.Info("migrations: backfill finished",
		slog.String("migration", BackfillName),
		slog.Int("subscriptionsUpdated", report.SubscriptionsUpdated),
		slog.Int("transactionsUpdated", report.TransactionsUpdated),
		slog.Int("tenantsProvisioned", report.TenantsProvisioned),
		slog.Int("clientsBackStamped", report.ClientsBackStamped),
	)
	return report, nil
}

// resolveTenantForBackfill returns the paired external tenant for a
// legacy Client. The provisioned/backStamped booleans drive the report
// counters so the operator sees real work separately from cached hits.
func resolveTenantForBackfill(ctx context.Context, d BackfillDeps, clientUUID string) (tenantUUID string, provisioned, backStamped bool, err error) {
	client, err := d.Clients.GetByUUID(ctx, clientUUID)
	if err != nil {
		return "", false, false, fmt.Errorf("load client: %w", err)
	}
	if client.OrgUUID != "" {
		return client.OrgUUID, false, false, nil
	}
	name := client.DisplayName
	if name == "" {
		name = client.LegalName
	}
	if name == "" {
		name = client.Email
	}
	if name == "" {
		return "", false, false, errors.New("client has no name to derive tenant from")
	}
	tenant, err := d.TenantProvider.FindOrProvisionLegacyClientTenant(ctx, iface.LegacyClientTenantSpec{
		LegacyClientUUID: client.UUID,
		OwnerUserUUID:    d.MigrationOwnerUUID,
		Name:             name,
		VATNumber:        client.VATNumber,
		FiscalCode:       client.FiscalCode,
		StripeCustomerID: client.StripeCustomerID,
	})
	if err != nil || tenant == nil {
		return "", false, false, fmt.Errorf("provision tenant: %w", err)
	}
	if err := d.Clients.SetOrgUUID(ctx, client.UUID, tenant.UUID); err != nil {
		return tenant.UUID, true, false, fmt.Errorf("back-stamp client.orgUUID: %w", err)
	}
	return tenant.UUID, true, true, nil
}

// resolveTransactionTenant derives the tenantUUID for a transaction by
// consulting the owning subscription first (fast path — post-pass-1 every
// subscription carries tenantUUID) and falling back to the legacy
// clientUUID → tenantUUID cache built in pass 1.
func resolveTransactionTenant(
	ctx context.Context,
	d BackfillDeps,
	tx *pmtmodels.Transaction,
	subCache map[string]string,
	tenantByClient map[string]string,
) (string, error) {
	if tx.SubscriptionUUID != "" {
		if cached, ok := subCache[tx.SubscriptionUUID]; ok {
			return cached, nil
		}
		sub, err := d.Subs.GetByUUID(ctx, tx.SubscriptionUUID)
		if err == nil && sub != nil {
			subCache[tx.SubscriptionUUID] = sub.TenantUUID
			if sub.TenantUUID != "" {
				return sub.TenantUUID, nil
			}
			if sub.ClientUUID != "" {
				if cached, ok := tenantByClient[sub.ClientUUID]; ok {
					return cached, nil
				}
			}
		}
	}
	if tx.ClientUUID != "" {
		if cached, ok := tenantByClient[tx.ClientUUID]; ok {
			return cached, nil
		}
	}
	return "", nil
}
