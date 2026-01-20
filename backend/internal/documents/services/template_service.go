package services

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/orkestra/backend/internal/documents/models"
	"github.com/orkestra/backend/internal/documents/repository"
)

// Errors for template service
var (
	ErrTemplateNameRequired    = errors.New("template name is required")
	ErrTemplateContentRequired = errors.New("template HTML content is required")
	ErrInvalidTemplateType     = errors.New("invalid template type")
	ErrInvalidPageSize         = errors.New("invalid page size")
	ErrInvalidOrientation      = errors.New("invalid page orientation")
)

// TemplateService defines the interface for template business logic
type TemplateService interface {
	// CRUD operations
	Create(ctx context.Context, input *models.CreateTemplateInput, createdBy string) (*models.Template, error)
	GetByUUID(ctx context.Context, uuid string) (*models.Template, error)
	GetDefault(ctx context.Context, templateType models.TemplateType) (*models.Template, error)
	List(ctx context.Context, filters *models.TemplateFilters, pagination models.PaginationParams) (*models.TemplateListResponse, error)
	Update(ctx context.Context, uuid string, input *models.UpdateTemplateInput, updatedBy string) (*models.Template, error)
	Delete(ctx context.Context, uuid string) error

	// Template management
	SetDefault(ctx context.Context, uuid string) error
	Duplicate(ctx context.Context, uuid string, newName string, createdBy string) (*models.Template, error)

	// Template variables
	GetVariables(ctx context.Context, templateType models.TemplateType) (*models.TemplateVariablesResponse, error)

	// Seeding
	SeedBuiltInTemplates(ctx context.Context) error
}

type templateService struct {
	templateRepo   repository.TemplateRepository
	templateEngine TemplateEngine
	logger         *slog.Logger
}

// NewTemplateService creates a new template service
func NewTemplateService(
	templateRepo repository.TemplateRepository,
	templateEngine TemplateEngine,
	logger *slog.Logger,
) TemplateService {
	return &templateService{
		templateRepo:   templateRepo,
		templateEngine: templateEngine,
		logger:         logger,
	}
}

// Create creates a new template
func (s *templateService) Create(ctx context.Context, input *models.CreateTemplateInput, createdBy string) (*models.Template, error) {
	// Validate input
	if err := s.validateCreateInput(input); err != nil {
		return nil, err
	}

	// Validate template content
	if err := s.templateEngine.Validate(ctx, input.HTMLContent); err != nil {
		return nil, err
	}

	// Set defaults
	pageSize := input.PageSize
	if pageSize == "" {
		pageSize = models.PageSizeA4
	}
	orientation := input.Orientation
	if orientation == "" {
		orientation = models.PageOrientationPortrait
	}
	margins := input.Margins
	if margins == nil {
		defaultMargins := models.DefaultMargins()
		margins = &defaultMargins
	}

	template := &models.Template{
		UUID:        uuid.New().String(),
		Name:        input.Name,
		Description: input.Description,
		Type:        input.Type,
		HTMLContent: input.HTMLContent,
		CSSContent:  input.CSSContent,
		PageSize:    pageSize,
		Orientation: orientation,
		Margins:     *margins,
		HeaderHTML:  input.HeaderHTML,
		FooterHTML:  input.FooterHTML,
		IsDefault:   false,
		IsBuiltIn:   false,
		IsActive:    true,
		Version:     1,
		CreatedBy:   createdBy,
	}

	if err := s.templateRepo.Create(ctx, template); err != nil {
		s.logger.Error("Failed to create template",
			slog.String("name", input.Name),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	s.logger.Info("Template created",
		slog.String("uuid", template.UUID),
		slog.String("name", template.Name),
		slog.String("type", string(template.Type)),
	)

	return template, nil
}

// GetByUUID retrieves a template by UUID
func (s *templateService) GetByUUID(ctx context.Context, uuid string) (*models.Template, error) {
	return s.templateRepo.GetByUUID(ctx, uuid)
}

// GetDefault retrieves the default template for a type
func (s *templateService) GetDefault(ctx context.Context, templateType models.TemplateType) (*models.Template, error) {
	return s.templateRepo.GetDefault(ctx, templateType)
}

// List retrieves templates with filters and pagination
func (s *templateService) List(ctx context.Context, filters *models.TemplateFilters, pagination models.PaginationParams) (*models.TemplateListResponse, error) {
	templates, total, err := s.templateRepo.List(ctx, filters, pagination)
	if err != nil {
		return nil, err
	}

	// Convert to list items
	items := make([]models.TemplateListItem, len(templates))
	for i, t := range templates {
		items[i] = t.ToListItem()
	}

	// Calculate total pages
	totalPages := int(total) / pagination.PageSize
	if int(total)%pagination.PageSize > 0 {
		totalPages++
	}

	return &models.TemplateListResponse{
		Templates:  items,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

// Update updates a template
func (s *templateService) Update(ctx context.Context, uuid string, input *models.UpdateTemplateInput, updatedBy string) (*models.Template, error) {
	// Get existing template
	template, err := s.templateRepo.GetByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if input.Name != nil {
		if *input.Name == "" {
			return nil, ErrTemplateNameRequired
		}
		template.Name = *input.Name
	}
	if input.Description != nil {
		template.Description = *input.Description
	}
	if input.HTMLContent != nil {
		if *input.HTMLContent == "" {
			return nil, ErrTemplateContentRequired
		}
		// Validate new content
		if err := s.templateEngine.Validate(ctx, *input.HTMLContent); err != nil {
			return nil, err
		}
		template.HTMLContent = *input.HTMLContent
	}
	if input.CSSContent != nil {
		template.CSSContent = *input.CSSContent
	}
	if input.PageSize != nil {
		if !input.PageSize.IsValid() {
			return nil, ErrInvalidPageSize
		}
		template.PageSize = *input.PageSize
	}
	if input.Orientation != nil {
		if !input.Orientation.IsValid() {
			return nil, ErrInvalidOrientation
		}
		template.Orientation = *input.Orientation
	}
	if input.Margins != nil {
		template.Margins = *input.Margins
	}
	if input.HeaderHTML != nil {
		template.HeaderHTML = *input.HeaderHTML
	}
	if input.FooterHTML != nil {
		template.FooterHTML = *input.FooterHTML
	}
	if input.IsActive != nil {
		template.IsActive = *input.IsActive
	}

	template.UpdatedBy = updatedBy

	if err := s.templateRepo.Update(ctx, template); err != nil {
		s.logger.Error("Failed to update template",
			slog.String("uuid", uuid),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	s.logger.Info("Template updated",
		slog.String("uuid", template.UUID),
		slog.String("name", template.Name),
	)

	return template, nil
}

// Delete soft-deletes a template
func (s *templateService) Delete(ctx context.Context, uuid string) error {
	if err := s.templateRepo.SoftDelete(ctx, uuid); err != nil {
		s.logger.Error("Failed to delete template",
			slog.String("uuid", uuid),
			slog.String("error", err.Error()),
		)
		return err
	}

	s.logger.Info("Template deleted", slog.String("uuid", uuid))
	return nil
}

// SetDefault sets a template as the default for its type
func (s *templateService) SetDefault(ctx context.Context, uuid string) error {
	// Get template to find its type
	template, err := s.templateRepo.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}

	if err := s.templateRepo.SetDefault(ctx, uuid, template.Type); err != nil {
		s.logger.Error("Failed to set default template",
			slog.String("uuid", uuid),
			slog.String("error", err.Error()),
		)
		return err
	}

	s.logger.Info("Template set as default",
		slog.String("uuid", uuid),
		slog.String("type", string(template.Type)),
	)
	return nil
}

// Duplicate creates a copy of an existing template
func (s *templateService) Duplicate(ctx context.Context, uuid string, newName string, createdBy string) (*models.Template, error) {
	// Get existing template
	original, err := s.templateRepo.GetByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}

	// Create new template with same content
	input := &models.CreateTemplateInput{
		Name:        newName,
		Description: original.Description,
		Type:        original.Type,
		HTMLContent: original.HTMLContent,
		CSSContent:  original.CSSContent,
		PageSize:    original.PageSize,
		Orientation: original.Orientation,
		Margins:     &original.Margins,
		HeaderHTML:  original.HeaderHTML,
		FooterHTML:  original.FooterHTML,
	}

	return s.Create(ctx, input, createdBy)
}

// GetVariables returns available variables for a template type
func (s *templateService) GetVariables(ctx context.Context, templateType models.TemplateType) (*models.TemplateVariablesResponse, error) {
	vars := GetTemplateVariables(templateType)
	return &vars, nil
}

// SeedBuiltInTemplates seeds the default built-in templates
func (s *templateService) SeedBuiltInTemplates(ctx context.Context) error {
	// Check if templates already exist
	count, err := s.templateRepo.Count(ctx)
	if err != nil {
		return err
	}

	if count > 0 {
		s.logger.Info("Built-in templates already seeded", slog.Int64("count", count))
		return nil
	}

	templates := getBuiltInTemplates()
	if err := s.templateRepo.CreateBatch(ctx, templates); err != nil {
		s.logger.Error("Failed to seed built-in templates", slog.String("error", err.Error()))
		return err
	}

	s.logger.Info("Built-in templates seeded", slog.Int("count", len(templates)))
	return nil
}

func (s *templateService) validateCreateInput(input *models.CreateTemplateInput) error {
	if input.Name == "" {
		return ErrTemplateNameRequired
	}
	if input.HTMLContent == "" {
		return ErrTemplateContentRequired
	}
	if !input.Type.IsValid() {
		return ErrInvalidTemplateType
	}
	if input.PageSize != "" && !input.PageSize.IsValid() {
		return ErrInvalidPageSize
	}
	if input.Orientation != "" && !input.Orientation.IsValid() {
		return ErrInvalidOrientation
	}
	return nil
}

// getBuiltInTemplates returns the default built-in templates
func getBuiltInTemplates() []*models.Template {
	defaultMargins := models.DefaultMargins()

	return []*models.Template{
		{
			UUID:        uuid.New().String(),
			Name:        "Invoice - Default",
			Description: "Default Italian invoice template (Fattura)",
			Type:        models.TemplateTypeInvoice,
			HTMLContent: getDefaultInvoiceHTML(),
			CSSContent:  getDefaultInvoiceCSS(),
			PageSize:    models.PageSizeA4,
			Orientation: models.PageOrientationPortrait,
			Margins:     defaultMargins,
			IsDefault:   true,
			IsBuiltIn:   true,
			IsActive:    true,
			Version:     1,
		},
		{
			UUID:        uuid.New().String(),
			Name:        "Offer - Default",
			Description: "Default offer/quote template (Preventivo)",
			Type:        models.TemplateTypeOffer,
			HTMLContent: getDefaultOfferHTML(),
			CSSContent:  getDefaultOfferCSS(),
			PageSize:    models.PageSizeA4,
			Orientation: models.PageOrientationPortrait,
			Margins:     defaultMargins,
			IsDefault:   true,
			IsBuiltIn:   true,
			IsActive:    true,
			Version:     1,
		},
	}
}

// Default template content (will be replaced with embedded files later)
func getDefaultInvoiceHTML() string {
	return `<!DOCTYPE html>
<html lang="it">
<head>
  <meta charset="UTF-8">
  <title>Fattura {{.number}}</title>
</head>
<body>
  <div class="header">
    <div class="title-section">
      <div class="invoice-title">FATTURA</div>
      <div class="invoice-meta">N. {{.number}} del {{formatDateIT .date}}</div>
    </div>
    <div class="company-info">
      <strong>{{.seller.name}}</strong><br>
      {{.seller.address}}<br>
      P.IVA: {{.seller.vatNumber}}
      {{if .seller.pec}}<br>PEC: {{.seller.pec}}{{end}}
    </div>
  </div>

  <div class="customer-box">
    <strong>Spett.le</strong><br>
    {{.buyer.name}}<br>
    {{.buyer.address}}<br>
    {{if .buyer.vatNumber}}P.IVA: {{.buyer.vatNumber}}<br>{{end}}
    {{if .buyer.fiscalCode}}C.F.: {{.buyer.fiscalCode}}{{end}}
  </div>

  <table class="items-table">
    <thead>
      <tr>
        <th class="col-num">#</th>
        <th class="col-desc">Descrizione</th>
        <th class="col-qty">Q.tà</th>
        <th class="col-price">Prezzo</th>
        <th class="col-vat">IVA</th>
        <th class="col-total">Totale</th>
      </tr>
    </thead>
    <tbody>
      {{range $i, $line := .lines}}
      <tr>
        <td class="col-num">{{lineNumber $i}}</td>
        <td class="col-desc">{{$line.Description}}</td>
        <td class="col-qty">{{formatNumber $line.Quantity 2}}</td>
        <td class="col-price">{{formatMoneyEUR $line.UnitPrice}}</td>
        <td class="col-vat">{{formatNumber $line.VATRate 0}}%</td>
        <td class="col-total">{{formatMoneyEUR $line.TotalPrice}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>

  <div class="totals">
    <div class="totals-row">
      <span class="label">Imponibile:</span>
      <span class="value">{{formatMoneyEUR .totalTaxable}}</span>
    </div>
    <div class="totals-row">
      <span class="label">IVA:</span>
      <span class="value">{{formatMoneyEUR .totalVAT}}</span>
    </div>
    <div class="totals-row total-final">
      <span class="label">TOTALE:</span>
      <span class="value">{{formatMoneyEUR .totalAmount}}</span>
    </div>
  </div>

  {{if .paymentTerms}}
  <div class="payment-info">
    <strong>Modalità di pagamento:</strong> {{.paymentTerms}}
  </div>
  {{end}}

  {{if .notes}}
  <div class="notes">
    <strong>Note:</strong><br>
    {{.notes}}
  </div>
  {{end}}
</body>
</html>`
}

func getDefaultInvoiceCSS() string {
	return `@page {
  size: A4;
  margin: 20mm;
}

body {
  font-family: 'Helvetica Neue', Arial, sans-serif;
  font-size: 10pt;
  line-height: 1.4;
  color: #333;
  margin: 0;
  padding: 0;
}

.header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 30px;
  padding-bottom: 20px;
  border-bottom: 2px solid #333;
}

.title-section {
  flex: 1;
}

.invoice-title {
  font-size: 28pt;
  font-weight: bold;
  color: #333;
  margin-bottom: 5px;
}

.invoice-meta {
  font-size: 11pt;
  color: #666;
}

.company-info {
  text-align: right;
  font-size: 9pt;
  line-height: 1.6;
}

.customer-box {
  background: #f8f8f8;
  padding: 15px 20px;
  margin-bottom: 25px;
  border-left: 4px solid #333;
  font-size: 10pt;
  line-height: 1.6;
}

.items-table {
  width: 100%;
  border-collapse: collapse;
  margin: 25px 0;
}

.items-table th {
  background: #333;
  color: white;
  padding: 12px 10px;
  text-align: left;
  font-weight: 600;
  font-size: 9pt;
  text-transform: uppercase;
}

.items-table td {
  padding: 12px 10px;
  border-bottom: 1px solid #ddd;
  font-size: 9pt;
}

.items-table tbody tr:hover {
  background: #fafafa;
}

.col-num { width: 5%; text-align: center; }
.col-desc { width: 40%; }
.col-qty { width: 10%; text-align: right; }
.col-price { width: 15%; text-align: right; }
.col-vat { width: 10%; text-align: center; }
.col-total { width: 20%; text-align: right; }

.totals {
  margin-top: 20px;
  margin-left: auto;
  width: 280px;
}

.totals-row {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid #eee;
  font-size: 10pt;
}

.totals-row .label {
  color: #666;
}

.totals-row .value {
  font-weight: 500;
}

.total-final {
  border-bottom: none;
  border-top: 2px solid #333;
  padding-top: 12px;
  margin-top: 5px;
  font-size: 14pt;
  font-weight: bold;
}

.total-final .label,
.total-final .value {
  color: #333;
}

.payment-info {
  margin-top: 30px;
  padding: 15px;
  background: #f8f8f8;
  font-size: 9pt;
}

.notes {
  margin-top: 20px;
  padding: 15px;
  background: #fffbf0;
  border-left: 4px solid #f0ad4e;
  font-size: 9pt;
}`
}

func getDefaultOfferHTML() string {
	return `<!DOCTYPE html>
<html lang="it">
<head>
  <meta charset="UTF-8">
  <title>Preventivo {{.number}}</title>
</head>
<body>
  <div class="header">
    <div class="title-section">
      <div class="offer-title">PREVENTIVO</div>
      <div class="offer-meta">N. {{.number}} del {{formatDateIT .date}}</div>
      {{if .validUntil}}<div class="validity">Valido fino al: {{formatDateIT .validUntil}}</div>{{end}}
    </div>
    <div class="company-info">
      <strong>{{.company.name}}</strong><br>
      {{.company.address}}<br>
      {{if .company.vatNumber}}P.IVA: {{.company.vatNumber}}<br>{{end}}
      {{if .company.email}}{{.company.email}}<br>{{end}}
      {{if .company.phone}}Tel: {{.company.phone}}{{end}}
    </div>
  </div>

  <div class="customer-box">
    <strong>Cliente</strong><br>
    {{.customer.name}}<br>
    {{if .customer.address}}{{.customer.address}}<br>{{end}}
    {{if .customer.email}}{{.customer.email}}{{end}}
  </div>

  {{if .subject}}
  <div class="subject">
    <strong>Oggetto:</strong> {{.subject}}
  </div>
  {{end}}

  <table class="items-table">
    <thead>
      <tr>
        <th class="col-num">#</th>
        <th class="col-desc">Descrizione</th>
        <th class="col-qty">Q.tà</th>
        <th class="col-price">Prezzo Unit.</th>
        <th class="col-total">Totale</th>
      </tr>
    </thead>
    <tbody>
      {{range $i, $item := .items}}
      <tr>
        <td class="col-num">{{lineNumber $i}}</td>
        <td class="col-desc">{{$item.Description}}</td>
        <td class="col-qty">{{formatNumber $item.Quantity 2}}</td>
        <td class="col-price">{{formatMoneyEUR $item.UnitPrice}}</td>
        <td class="col-total">{{formatMoneyEUR $item.Total}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>

  <div class="totals">
    <div class="totals-row">
      <span class="label">Subtotale:</span>
      <span class="value">{{formatMoneyEUR .subtotal}}</span>
    </div>
    {{if .tax}}
    <div class="totals-row">
      <span class="label">IVA:</span>
      <span class="value">{{formatMoneyEUR .tax}}</span>
    </div>
    {{end}}
    <div class="totals-row total-final">
      <span class="label">TOTALE:</span>
      <span class="value">{{formatMoneyEUR .total}}</span>
    </div>
  </div>

  {{if .notes}}
  <div class="notes">
    <strong>Note:</strong><br>
    {{.notes}}
  </div>
  {{end}}

  <div class="footer">
    <p>Questo preventivo è valido per 30 giorni dalla data di emissione, salvo diversamente specificato.</p>
  </div>
</body>
</html>`
}

func getDefaultOfferCSS() string {
	return `@page {
  size: A4;
  margin: 20mm;
}

body {
  font-family: 'Helvetica Neue', Arial, sans-serif;
  font-size: 10pt;
  line-height: 1.4;
  color: #333;
  margin: 0;
  padding: 0;
}

.header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 30px;
  padding-bottom: 20px;
  border-bottom: 2px solid #2563eb;
}

.title-section {
  flex: 1;
}

.offer-title {
  font-size: 28pt;
  font-weight: bold;
  color: #2563eb;
  margin-bottom: 5px;
}

.offer-meta {
  font-size: 11pt;
  color: #666;
}

.validity {
  font-size: 10pt;
  color: #dc2626;
  margin-top: 5px;
}

.company-info {
  text-align: right;
  font-size: 9pt;
  line-height: 1.6;
}

.customer-box {
  background: #eff6ff;
  padding: 15px 20px;
  margin-bottom: 25px;
  border-left: 4px solid #2563eb;
  font-size: 10pt;
  line-height: 1.6;
}

.subject {
  margin-bottom: 20px;
  font-size: 11pt;
}

.items-table {
  width: 100%;
  border-collapse: collapse;
  margin: 25px 0;
}

.items-table th {
  background: #2563eb;
  color: white;
  padding: 12px 10px;
  text-align: left;
  font-weight: 600;
  font-size: 9pt;
  text-transform: uppercase;
}

.items-table td {
  padding: 12px 10px;
  border-bottom: 1px solid #ddd;
  font-size: 9pt;
}

.items-table tbody tr:hover {
  background: #f8fafc;
}

.col-num { width: 5%; text-align: center; }
.col-desc { width: 50%; }
.col-qty { width: 10%; text-align: right; }
.col-price { width: 17%; text-align: right; }
.col-total { width: 18%; text-align: right; }

.totals {
  margin-top: 20px;
  margin-left: auto;
  width: 280px;
}

.totals-row {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid #eee;
  font-size: 10pt;
}

.totals-row .label {
  color: #666;
}

.totals-row .value {
  font-weight: 500;
}

.total-final {
  border-bottom: none;
  border-top: 2px solid #2563eb;
  padding-top: 12px;
  margin-top: 5px;
  font-size: 14pt;
  font-weight: bold;
  color: #2563eb;
}

.notes {
  margin-top: 30px;
  padding: 15px;
  background: #fffbf0;
  border-left: 4px solid #f0ad4e;
  font-size: 9pt;
}

.footer {
  margin-top: 40px;
  padding-top: 20px;
  border-top: 1px solid #eee;
  font-size: 8pt;
  color: #666;
  text-align: center;
}`
}
