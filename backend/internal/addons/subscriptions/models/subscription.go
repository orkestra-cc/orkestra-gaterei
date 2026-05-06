package models

import (
	"time"

	"github.com/orkestra/backend/internal/shared/iface"
)

// Subscription links an Owner (a user OR an admin-attached tenant) to a
// Service tier with a recurring billing cycle. The renewal job scans
// NextBillingAt and transitions Status through the state machine defined in
// services/subscription_service.go.
//
// Post-onboarding refactor: the legacy TenantUUID-only ownership has been
// replaced by polymorphic OwnerKind + OwnerUUID. Self-registered clients
// carry Kind="user" and a userUUID; admin-attached businesses carry
// Kind="tenant" and a tenantUUID. The capability projection mirrors the
// shape so the entitlement syncer can grant against either kind without
// branching.
type Subscription struct {
	UUID               string          `bson:"uuid" json:"uuid"`
	OwnerKind          iface.OwnerKind `bson:"ownerKind" json:"ownerKind"`
	OwnerUUID          string          `bson:"ownerUUID" json:"ownerUUID"`
	ServiceUUID        string          `bson:"serviceUUID" json:"serviceUUID"`
	TierCode           string          `bson:"tierCode" json:"tierCode"`
	Status             SubStatus       `bson:"status" json:"status"`
	StartedAt          time.Time       `bson:"startedAt" json:"startedAt"`
	CurrentPeriodStart time.Time       `bson:"currentPeriodStart" json:"currentPeriodStart"`
	CurrentPeriodEnd   time.Time       `bson:"currentPeriodEnd" json:"currentPeriodEnd"`
	NextBillingAt      time.Time       `bson:"nextBillingAt" json:"nextBillingAt"`
	CancelledAt        *time.Time      `bson:"cancelledAt,omitempty" json:"cancelledAt,omitempty"`
	EndsAt             *time.Time      `bson:"endsAt,omitempty" json:"endsAt,omitempty"`
	CancelAtPeriodEnd  bool            `bson:"cancelAtPeriodEnd" json:"cancelAtPeriodEnd"`
	FailedChargeCount  int             `bson:"failedChargeCount" json:"failedChargeCount"`
	PaymentProvider    string          `bson:"paymentProvider,omitempty" json:"paymentProvider,omitempty"`
	PaymentMethodID    string          `bson:"paymentMethodID,omitempty" json:"paymentMethodID,omitempty"`
	Metadata           map[string]any  `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt          time.Time       `bson:"createdAt" json:"createdAt"`
	UpdatedAt          time.Time       `bson:"updatedAt" json:"updatedAt"`
}

// Owner returns the polymorphic principal that holds this subscription.
func (s Subscription) Owner() iface.Owner {
	return iface.Owner{Kind: s.OwnerKind, UUID: s.OwnerUUID}
}
