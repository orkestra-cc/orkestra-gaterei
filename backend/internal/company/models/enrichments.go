package models

import "encoding/json"

// AdvancedData holds data from the IT-advanced endpoint
type AdvancedData struct {
	REACode             string               `bson:"reaCode,omitempty" json:"reaCode,omitempty"`
	CCIAA               string               `bson:"cciaa,omitempty" json:"cciaa,omitempty"`
	AtecoClassification *AtecoClassification `bson:"atecoClassification,omitempty" json:"atecoClassification,omitempty"`
	DetailedLegalForm   *LegalFormDetail     `bson:"detailedLegalForm,omitempty" json:"detailedLegalForm,omitempty"`
	PEC                 string               `bson:"pec,omitempty" json:"pec,omitempty"`
	StartDate           string               `bson:"startDate,omitempty" json:"startDate,omitempty"`
	EndDate             string               `bson:"endDate,omitempty" json:"endDate,omitempty"`
	TaxCodeCeased       *bool                `bson:"taxCodeCeased,omitempty" json:"taxCodeCeased,omitempty"`
}

// AtecoClassification represents ATECO industry classification codes
type AtecoClassification struct {
	Ateco2007    *CodeDescription   `bson:"ateco2007,omitempty" json:"ateco2007,omitempty"`
	Nace         *CodeDescription   `bson:"nace,omitempty" json:"nace,omitempty"`
	Sector       *CodeDescription   `bson:"sector,omitempty" json:"sector,omitempty"`
	Category     *CodeDescription   `bson:"category,omitempty" json:"category,omitempty"`
	SubCategory  *CodeDescription   `bson:"subCategory,omitempty" json:"subCategory,omitempty"`
}

// CodeDescription is a reusable code + description pair
type CodeDescription struct {
	Code        string `bson:"code,omitempty" json:"code,omitempty"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`
}

// LegalFormDetail represents detailed legal form information
type LegalFormDetail struct {
	Code        string `bson:"code,omitempty" json:"code,omitempty"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`
}

// MarketingData holds data from the IT-marketing endpoint
type MarketingData struct {
	Contacts    json.RawMessage `bson:"contacts,omitempty" json:"contacts,omitempty"`
	WebAndSocial json.RawMessage `bson:"webAndSocial,omitempty" json:"webAndSocial,omitempty"`
	Mail        json.RawMessage `bson:"mail,omitempty" json:"mail,omitempty"`
	PEC         string          `bson:"pec,omitempty" json:"pec,omitempty"`
	Employees   json.RawMessage `bson:"employees,omitempty" json:"employees,omitempty"`
	Ecofin      json.RawMessage `bson:"ecofin,omitempty" json:"ecofin,omitempty"`
	Branches    json.RawMessage `bson:"branches,omitempty" json:"branches,omitempty"`
	AllOffices  json.RawMessage `bson:"allOffices,omitempty" json:"allOffices,omitempty"`
}

// StakeholdersData holds data from the IT-stakeholders endpoint
type StakeholdersData struct {
	Managers           json.RawMessage    `bson:"managers,omitempty" json:"managers,omitempty"`
	Shareholders       json.RawMessage    `bson:"shareholders,omitempty" json:"shareholders,omitempty"`
	CorporateGroups    json.RawMessage    `bson:"corporateGroups,omitempty" json:"corporateGroups,omitempty"`
	Subsidiaries       json.RawMessage    `bson:"subsidiaries,omitempty" json:"subsidiaries,omitempty"`
	AffiliateCompanies json.RawMessage    `bson:"affiliateCompanies,omitempty" json:"affiliateCompanies,omitempty"`
}

// AMLData holds data from the IT-aml endpoint (anti-money laundering)
type AMLData struct {
	Managers         json.RawMessage `bson:"managers,omitempty" json:"managers,omitempty"`
	Shareholders     json.RawMessage `bson:"shareholders,omitempty" json:"shareholders,omitempty"`
	CorporateGroups  json.RawMessage `bson:"corporateGroups,omitempty" json:"corporateGroups,omitempty"`
	ForeignTrade     json.RawMessage `bson:"foreignTrade,omitempty" json:"foreignTrade,omitempty"`
	PublicTenders    json.RawMessage `bson:"publicTenders,omitempty" json:"publicTenders,omitempty"`
	OperatingResults json.RawMessage `bson:"operatingResults,omitempty" json:"operatingResults,omitempty"`
	Debts            json.RawMessage `bson:"debts,omitempty" json:"debts,omitempty"`
	RAE              *CodeDescription `bson:"rae,omitempty" json:"rae,omitempty"`
	SAE              *CodeDescription `bson:"sae,omitempty" json:"sae,omitempty"`
}
