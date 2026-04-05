package module

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ModuleConfig is the MongoDB document storing a module's runtime configuration.
// Created automatically by the registry from each module's ConfigSchema() on first boot.
type ModuleConfig struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	ModuleName      string             `bson:"moduleName" json:"moduleName"`
	DisplayName     string             `bson:"displayName" json:"displayName"`
	Description     string             `bson:"description" json:"description"`
	Category        ModuleCategory     `bson:"category" json:"category"`
	Enabled         bool               `bson:"enabled" json:"enabled"`
	ConfigValues    map[string]string  `bson:"configValues" json:"configValues"`
	EncryptedValues map[string]string  `bson:"encryptedValues" json:"-"`          // never exposed in API
	ConfigSchema    []ConfigField      `bson:"configSchema" json:"configSchema"`
	DependsOn       []string           `bson:"dependsOn,omitempty" json:"dependsOn,omitempty"`
	NeedsRestart    bool               `bson:"needsRestart" json:"needsRestart"` // true when config changed post-init
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt" json:"updatedAt"`
}
