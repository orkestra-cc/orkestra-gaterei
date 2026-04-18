package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MFAFactorType enumerates the kinds of second-factor authentication supported.
// Block A ships TOTP only; WebAuthn is reserved for a later phase.
type MFAFactorType string

const (
	MFAFactorTOTP     MFAFactorType = "totp"
	MFAFactorWebAuthn MFAFactorType = "webauthn" // reserved
)

// MFAStatus communicates the per-user enrollment state to the frontend.
// The "required" variants are populated by Block B once role-based MFA
// enforcement is wired in; Block A only reports enrolled vs not_required.
type MFAStatus string

const (
	MFAStatusNotRequired             MFAStatus = "not_required"
	MFAStatusRequiredPendingEnrollment MFAStatus = "required_pending_enrollment"
	MFAStatusEnrolled                MFAStatus = "enrolled"
	MFAStatusGrace                   MFAStatus = "grace"
)

// MFAFactorDoc represents a stored second-factor binding for a user.
// Secrets are AES-256-GCM encrypted and backup codes are argon2id hashed,
// so a database dump does not expose live credentials.
type MFAFactorDoc struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	UUID              string             `bson:"uuid" json:"id"`
	UserUUID          string             `bson:"userUuid" json:"userUuid"`
	Type              MFAFactorType      `bson:"type" json:"type"`
	SecretEnc         string             `bson:"secretEnc" json:"-"`
	VerifiedAt        *time.Time         `bson:"verifiedAt,omitempty" json:"verifiedAt,omitempty"`
	CreatedAt         time.Time          `bson:"createdAt" json:"createdAt"`
	LastUsedAt        *time.Time         `bson:"lastUsedAt,omitempty" json:"lastUsedAt,omitempty"`
	BackupCodesHashed []string           `bson:"backupCodesHashed" json:"-"`
	// LastUsedStep is the most recent TOTP step index (unix / period)
	// accepted for this factor. Verify refuses any candidate whose step
	// is ≤ LastUsedStep, preventing replay of a just-captured code within
	// its 30s window.
	LastUsedStep int64 `bson:"lastUsedStep,omitempty" json:"-"`
}
