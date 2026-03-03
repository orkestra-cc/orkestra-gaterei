package models

// OpenAPIBaseResponse represents the raw response from the OpenAPI Company API
// GET /IT-start/{vatCode_taxCode_or_id}
type OpenAPIBaseResponse struct {
	Success bool                `json:"success"`
	Data    *OpenAPICompanyData `json:"data"`
	Message string              `json:"message,omitempty"`
	Error   int                 `json:"error,omitempty"`
}

// OpenAPICompanyData represents the company data from the API response
type OpenAPICompanyData struct {
	ID                  string              `json:"id"`
	TaxCode             string              `json:"taxCode"`
	CompanyName         string              `json:"companyName"`
	VATCode             string              `json:"vatCode"`
	ActivityStatus      string              `json:"activityStatus"`
	SDICode             string              `json:"sdiCode"`
	SDICodeTimestamp     string              `json:"sdiCodeTimestamp"`
	RegistrationDate    string              `json:"registrationDate"`
	Address             OpenAPIAddressData  `json:"address"`
	CreationTimestamp    string              `json:"creationTimestamp"`
	LastUpdateTimestamp  string              `json:"lastUpdateTimestamp"`
}

// OpenAPIAddressData represents the address data from the API response
type OpenAPIAddressData struct {
	RegisteredOffice *OpenAPIRegisteredOffice `json:"registeredOffice"`
}

// OpenAPIRegisteredOffice represents the registered office address
type OpenAPIRegisteredOffice struct {
	StreetName   string  `json:"streetName"`
	StreetNumber string  `json:"streetNumber"`
	Town         string  `json:"town"`
	Province     string  `json:"province"`
	ZipCode      string  `json:"zipCode"`
	Region       string  `json:"region"`
	RegionCode   string  `json:"regionCode"`
	Longitude    float64 `json:"longitude"`
	Latitude     float64 `json:"latitude"`
}

// CompanyLookupListResponse represents a paginated list of company lookups
type CompanyLookupListResponse struct {
	Lookups    []CompanyLookup `json:"lookups" doc:"List of company lookups"`
	Total      int64           `json:"total" doc:"Total number of lookups"`
	Page       int             `json:"page" doc:"Current page"`
	PageSize   int             `json:"pageSize" doc:"Items per page"`
	TotalPages int             `json:"totalPages" doc:"Total number of pages"`
}
