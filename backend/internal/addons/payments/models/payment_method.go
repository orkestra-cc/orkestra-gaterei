package models

import (
	"time"

	"github.com/orkestra/backend/internal/shared/iface"
)

// PaymentMethod is a tokenized card/SEPA/etc. stored on a provider customer.
// We never store raw PANs — only the provider's reference token (pm_xxx).
//
// Owner is the polymorphic principal the saved card belongs to — a self-
// registered user (Kind="user") or an admin-attached tenant (Kind="tenant").
type PaymentMethod struct {
	UUID             string          `bson:"uuid" json:"uuid"`
	OwnerKind        iface.OwnerKind `bson:"ownerKind" json:"ownerKind"`
	OwnerUUID        string          `bson:"ownerUUID" json:"ownerUUID"`
	Provider         ProviderName    `bson:"provider" json:"provider"`
	ProviderMethodID string          `bson:"providerMethodID" json:"providerMethodID"` // pm_xxx
	Brand            string          `bson:"brand,omitempty" json:"brand,omitempty"`
	Last4            string          `bson:"last4,omitempty" json:"last4,omitempty"`
	ExpiryMonth      int             `bson:"expiryMonth,omitempty" json:"expiryMonth,omitempty"`
	ExpiryYear       int             `bson:"expiryYear,omitempty" json:"expiryYear,omitempty"`
	IsDefault        bool            `bson:"isDefault" json:"isDefault"`
	CreatedAt        time.Time       `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time       `bson:"updatedAt" json:"updatedAt"`
}

// Owner returns the polymorphic principal that holds this payment method.
func (p PaymentMethod) Owner() iface.Owner {
	return iface.Owner{Kind: p.OwnerKind, UUID: p.OwnerUUID}
}
