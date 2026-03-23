package models

import "time"

// Project represents a project with an associated Hindsight memory bank
type Project struct {
	UUID            string `bson:"uuid" json:"uuid"`
	Name            string `bson:"name" json:"name"`
	Description     string `bson:"description" json:"description"`
	HindsightBankID string `bson:"hindsightBankId" json:"hindsightBankId"`

	// Document scoping - individual documents
	DocumentUUIDs []string `bson:"documentUuids" json:"documentUuids"`

	// Document scoping - category/standard filters
	ISOStandards []string `bson:"isoStandards,omitempty" json:"isoStandards,omitempty"`
	Categories   []string `bson:"categories,omitempty" json:"categories,omitempty"` // iso, law, regulation, generic

	Status    string    `bson:"status" json:"status"` // active, archived
	CreatedBy string    `bson:"createdBy" json:"createdBy"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

const (
	ProjectStatusActive   = "active"
	ProjectStatusArchived = "archived"
)
