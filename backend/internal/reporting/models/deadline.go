package models

import (
	"time"
)

// EntityType rappresenta il tipo di entità con scadenza
type EntityType string

const (
	EntityTypeVehicle EntityType = "vehicle"
	EntityTypeUser    EntityType = "user"
	EntityTypeMedical EntityType = "medical"
)

// DeadlineType rappresenta il tipo di scadenza
type DeadlineType string

const (
	// Vehicle deadline types
	DeadlineTypeRevision          DeadlineType = "revision"
	DeadlineTypeScheduledRevision DeadlineType = "scheduled_revision"
	DeadlineTypeInsurance         DeadlineType = "insurance"
	DeadlineTypeCarTax            DeadlineType = "car_tax"

	// User certification deadline types
	DeadlineTypeLicense      DeadlineType = "license"
	DeadlineTypeDriverCard   DeadlineType = "driver_card"
	DeadlineTypeCQC          DeadlineType = "cqc"
	DeadlineTypeADR          DeadlineType = "adr"
	DeadlineTypeTachograph   DeadlineType = "tachograph"
	DeadlineTypeMedicalCheck DeadlineType = "medical_check"
)

// DeadlineStatus rappresenta lo stato della scadenza
type DeadlineStatus string

const (
	DeadlineStatusExpired DeadlineStatus = "expired"
	DeadlineStatusWarning DeadlineStatus = "warning" // < 30 giorni
	DeadlineStatusOk      DeadlineStatus = "ok"
)

// DeadlineItem rappresenta un singolo elemento con scadenza
type DeadlineItem struct {
	ID              string         `json:"id"`
	EntityType      EntityType     `json:"entityType"`
	EntityID        string         `json:"entityId"`
	EntityName      string         `json:"entityName"`
	DeadlineType    DeadlineType   `json:"deadlineType"`
	ExpiryDate      time.Time      `json:"expiryDate"`
	DaysUntilExpiry int            `json:"daysUntilExpiry"`
	Status          DeadlineStatus `json:"status"`
	Notes           string         `json:"notes,omitempty"`
	// Additional fields for medical checks
	Doctor string `json:"doctor,omitempty"`
	Where  string `json:"where,omitempty"`
}

// DeadlineFilters rappresenta i filtri per la ricerca scadenze
type DeadlineFilters struct {
	EntityType EntityType     `json:"entityType,omitempty" query:"entityType"`
	Status     DeadlineStatus `json:"status,omitempty" query:"status"`
	FromDate   *time.Time     `json:"fromDate,omitempty" query:"fromDate"`
	ToDate     *time.Time     `json:"toDate,omitempty" query:"toDate"`
	Search     string         `json:"search,omitempty" query:"search"`
}

// PaginationParams rappresenta i parametri di paginazione
type PaginationParams struct {
	Page     int `json:"page" query:"page" default:"1" minimum:"1"`
	PageSize int `json:"pageSize" query:"pageSize" default:"20" minimum:"1" maximum:"100"`
}

// DeadlineReportResponse rappresenta la risposta paginata per il report scadenze
type DeadlineReportResponse struct {
	Deadlines  []DeadlineItem `json:"deadlines"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"pageSize"`
	TotalPages int            `json:"totalPages"`
}

// CalculateDeadlineStatus calcola lo stato della scadenza basato sui giorni rimanenti
func CalculateDeadlineStatus(expiryDate time.Time) DeadlineStatus {
	now := time.Now()
	if expiryDate.Before(now) {
		return DeadlineStatusExpired
	}

	daysUntil := int(time.Until(expiryDate).Hours() / 24)
	if daysUntil <= 30 {
		return DeadlineStatusWarning
	}

	return DeadlineStatusOk
}

// CalculateDaysUntilExpiry calcola i giorni mancanti alla scadenza
func CalculateDaysUntilExpiry(expiryDate time.Time) int {
	now := time.Now()
	duration := expiryDate.Sub(now)
	days := int(duration.Hours() / 24)
	return days
}
