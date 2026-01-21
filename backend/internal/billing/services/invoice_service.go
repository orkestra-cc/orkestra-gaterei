package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/orkestra/backend/internal/billing/models"
	"github.com/orkestra/backend/internal/billing/repository"
	documentsSvc "github.com/orkestra/backend/internal/documents/services"
)

// Common errors
var (
	ErrInvoiceNotFound       = errors.New("invoice not found")
	ErrInvoiceCannotEdit     = errors.New("invoice cannot be edited in current status")
	ErrInvoiceCannotSend     = errors.New("invoice cannot be sent in current status")
	ErrInvoiceCannotDelete   = errors.New("invoice cannot be deleted in current status")
	ErrInvoiceHTMLNotReady   = errors.New("HTML view not available: invoice has not been sent to SDI")
	ErrCustomerNotFound      = errors.New("customer not found")
	ErrSupplierNotFound      = errors.New("supplier not found")
	ErrInvalidInvoiceData    = errors.New("invalid invoice data")
)

// InvoiceService defines the interface for invoice business logic
type InvoiceService interface {
	// Issued invoices (fatture attive)
	CreateInvoice(ctx context.Context, input *models.CreateInvoiceInput, createdBy string) (*models.Invoice, error)
	GetInvoice(ctx context.Context, uuid string) (*models.Invoice, error)
	ListInvoices(ctx context.Context, filters *models.InvoiceFilters, pagination models.PaginationParams) (*models.InvoiceListResponse, error)
	UpdateInvoice(ctx context.Context, uuid string, input *models.UpdateInvoiceInput) (*models.Invoice, error)
	DeleteInvoice(ctx context.Context, uuid string) error
	SendInvoice(ctx context.Context, uuid string, sentBy string) (*models.SendInvoiceResponse, error)
	GetInvoicePDF(ctx context.Context, uuid string) ([]byte, error)
	GetInvoiceXML(ctx context.Context, uuid string) (string, error)
	GetInvoiceHTML(ctx context.Context, uuid string) ([]byte, error)

	// Received invoices (fatture passive)
	ListReceivedInvoices(ctx context.Context, filters *models.InvoiceFilters, pagination models.PaginationParams) (*models.InvoiceListResponse, error)
	AcceptReceivedInvoice(ctx context.Context, uuid string, acceptedBy string) error
	RejectReceivedInvoice(ctx context.Context, uuid string, reason string, rejectedBy string) error
	ImportInvoice(ctx context.Context, input *models.ImportInvoiceInput) (*models.ImportInvoiceResponse, error)

	// Legal storage / preserved documents
	GetPreservedDocument(ctx context.Context, uuid string) (*models.PreservedDocument, error)

	// Statistics
	GetStats(ctx context.Context, fromDate, toDate time.Time) (*models.BillingStats, error)
}

type invoiceService struct {
	invoiceRepo   repository.InvoiceRepository
	customerRepo  repository.CustomerRepository
	supplierRepo  repository.SupplierRepository
	companyRepo   repository.CompanyRepository
	openAPIClient OpenAPIClient
	xmlBuilder    XMLBuilder
	pdfService    documentsSvc.PDFService // Optional: nil if documents module disabled
	logger        *slog.Logger
}

// NewInvoiceService creates a new InvoiceService
func NewInvoiceService(
	invoiceRepo repository.InvoiceRepository,
	customerRepo repository.CustomerRepository,
	supplierRepo repository.SupplierRepository,
	companyRepo repository.CompanyRepository,
	openAPIClient OpenAPIClient,
	xmlBuilder XMLBuilder,
	pdfService documentsSvc.PDFService, // Can be nil if documents module is disabled
	logger *slog.Logger,
) InvoiceService {
	return &invoiceService{
		invoiceRepo:   invoiceRepo,
		customerRepo:  customerRepo,
		supplierRepo:  supplierRepo,
		companyRepo:   companyRepo,
		openAPIClient: openAPIClient,
		xmlBuilder:    xmlBuilder,
		pdfService:    pdfService,
		logger:        logger,
	}
}

func (s *invoiceService) CreateInvoice(ctx context.Context, input *models.CreateInvoiceInput, createdBy string) (*models.Invoice, error) {
	// Validate input
	if err := s.validateCreateInput(input); err != nil {
		return nil, err
	}

	// Get company (seller/provider) data
	var companyData *models.PartyData
	if input.CompanyID != "" {
		// Use specified company
		company, err := s.companyRepo.GetByUUID(ctx, input.CompanyID)
		if err != nil {
			if errors.Is(err, repository.ErrCompanyNotFound) {
				return nil, errors.New("specified company not found")
			}
			return nil, err
		}
		companyData = company.ToPartyData()
	} else {
		// Use default company
		company, err := s.companyRepo.GetDefault(ctx)
		if err != nil {
			if errors.Is(err, repository.ErrNoDefaultCompany) {
				return nil, errors.New("no company configured: please add a company in settings before creating invoices")
			}
			return nil, err
		}
		companyData = company.ToPartyData()
	}

	// Get customer data
	var customerData *models.PartyData
	if input.CustomerID != "" {
		customer, err := s.customerRepo.GetByUUID(ctx, input.CustomerID)
		if err != nil {
			if errors.Is(err, repository.ErrCustomerNotFound) {
				return nil, ErrCustomerNotFound
			}
			return nil, err
		}
		customerData = customer.ToPartyData()
	}

	// Build invoice
	invoice := &models.Invoice{
		UUID:               uuid.New().String(),
		Direction:          models.DirectionIssued,
		DocumentType:       input.DocumentType,
		Number:             input.Number,
		Date:               input.Date,
		Currency:           "EUR",
		CompanyID:          input.CompanyID,
		CedentePrestatore:  companyData,
		CustomerID:         input.CustomerID,
		CessionarioCommittente: customerData,
		Status:             models.StatusDraft,
		LegalStorageEnabled: input.LegalStorageEnabled,
		SignatureEnabled:    input.SignatureEnabled,
		Causale:      input.Causale,
		InternalNotes: input.InternalNotes,
		RelatedDocuments: input.RelatedDocuments,
		CreatedBy:    createdBy,
		ProgressivoInvio: GenerateProgressivoInvio(),
	}

	if input.Currency != "" {
		invoice.Currency = input.Currency
	}

	// Convert lines
	invoice.Lines = make([]models.InvoiceLine, 0, len(input.Lines))
	for i, line := range input.Lines {
		invoiceLine := models.InvoiceLine{
			LineNumber:          i + 1,
			Description:         line.Description,
			Quantity:            line.Quantity,
			UnitOfMeasure:       line.UnitOfMeasure,
			UnitPrice:           line.UnitPrice,
			VATRate:             line.VATRate,
			VATNature:           line.VATNature,
			Discounts:           line.Discounts,
			ProductCode:         line.ProductCode,
			StartDate:           line.StartDate,
			EndDate:             line.EndDate,
			AltriDatiGestionali: line.AltriDatiGestionali,
		}
		invoice.Lines = append(invoice.Lines, invoiceLine)
	}

	// Calculate totals
	invoice.CalculateTotals()

	// Convert payment terms
	if input.PaymentTerms != nil {
		invoice.PaymentTerms = &models.PaymentTerms{
			Condition:           input.PaymentTerms.Condition,
			PaymentMethod:       input.PaymentTerms.PaymentMethod,
			IBAN:                input.PaymentTerms.IBAN,
			BIC:                 input.PaymentTerms.BIC,
			ABI:                 input.PaymentTerms.ABI,
			CAB:                 input.PaymentTerms.CAB,
			Beneficiario:        input.PaymentTerms.Beneficiario,
			IstitutoFinanziario: input.PaymentTerms.IstitutoFinanziario,
			DueDate:             input.PaymentTerms.DueDate,
		}
	}

	// Copy stamp duty data (bollo virtuale)
	if input.DatiBollo != nil {
		invoice.DatiBollo = input.DatiBollo
	}

	// Save to database
	if err := s.invoiceRepo.Create(ctx, invoice); err != nil {
		return nil, err
	}

	s.logger.Info("invoice created",
		"invoiceUUID", invoice.UUID,
		"number", invoice.Number,
		"createdBy", createdBy,
	)

	return invoice, nil
}

func (s *invoiceService) GetInvoice(ctx context.Context, uuid string) (*models.Invoice, error) {
	invoice, err := s.invoiceRepo.GetByUUID(ctx, uuid)
	if err != nil {
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}
	return invoice, nil
}

func (s *invoiceService) ListInvoices(ctx context.Context, filters *models.InvoiceFilters, pagination models.PaginationParams) (*models.InvoiceListResponse, error) {
	// Default to issued invoices
	direction := models.DirectionIssued
	if filters != nil && filters.Direction != nil {
		direction = *filters.Direction
	} else if filters == nil {
		filters = &models.InvoiceFilters{}
	}
	filters.Direction = &direction

	invoices, total, err := s.invoiceRepo.List(ctx, filters, pagination)
	if err != nil {
		return nil, err
	}

	// Convert to summaries
	summaries := make([]models.InvoiceSummary, 0, len(invoices))
	for _, inv := range invoices {
		summary := models.InvoiceSummary{
			UUID:         inv.UUID,
			Direction:    inv.Direction,
			DocumentType: inv.DocumentType,
			Number:       inv.Number,
			Date:         inv.Date,
			TotalAmount:  inv.TotalAmount,
			Status:       inv.Status,
			SDIStatus:    inv.SDIStatus,
			CreatedAt:    inv.CreatedAt,
		}

		// Get party name based on direction
		if inv.Direction == models.DirectionIssued && inv.CessionarioCommittente != nil {
			summary.PartyName = inv.CessionarioCommittente.GetDisplayName()
		} else if inv.Direction == models.DirectionReceived && inv.CedentePrestatore != nil {
			summary.PartyName = inv.CedentePrestatore.GetDisplayName()
		}

		summaries = append(summaries, summary)
	}

	totalPages := int(total) / pagination.PageSize
	if int(total)%pagination.PageSize > 0 {
		totalPages++
	}

	return &models.InvoiceListResponse{
		Invoices:   summaries,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *invoiceService) UpdateInvoice(ctx context.Context, uuid string, input *models.UpdateInvoiceInput) (*models.Invoice, error) {
	invoice, err := s.invoiceRepo.GetByUUID(ctx, uuid)
	if err != nil {
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}

	// Check if invoice can be edited
	if !invoice.CanBeEdited() {
		return nil, ErrInvoiceCannotEdit
	}

	// Apply updates
	if input.Number != nil {
		invoice.Number = *input.Number
	}
	if input.Date != nil {
		invoice.Date = *input.Date
	}
	if input.InternalNotes != nil {
		invoice.InternalNotes = *input.InternalNotes
	}
	if input.Causale != nil {
		invoice.Causale = input.Causale
	}
	if input.RelatedDocuments != nil {
		invoice.RelatedDocuments = input.RelatedDocuments
	}

	// Update lines if provided
	if input.Lines != nil {
		invoice.Lines = make([]models.InvoiceLine, 0, len(input.Lines))
		for i, line := range input.Lines {
			invoiceLine := models.InvoiceLine{
				LineNumber:          i + 1,
				Description:         line.Description,
				Quantity:            line.Quantity,
				UnitOfMeasure:       line.UnitOfMeasure,
				UnitPrice:           line.UnitPrice,
				VATRate:             line.VATRate,
				VATNature:           line.VATNature,
				Discounts:           line.Discounts,
				ProductCode:         line.ProductCode,
				StartDate:           line.StartDate,
				EndDate:             line.EndDate,
				AltriDatiGestionali: line.AltriDatiGestionali,
			}
			invoice.Lines = append(invoice.Lines, invoiceLine)
		}
		// Recalculate totals
		invoice.CalculateTotals()
	}

	// Update payment terms if provided
	if input.PaymentTerms != nil {
		invoice.PaymentTerms = &models.PaymentTerms{
			Condition:           input.PaymentTerms.Condition,
			PaymentMethod:       input.PaymentTerms.PaymentMethod,
			IBAN:                input.PaymentTerms.IBAN,
			BIC:                 input.PaymentTerms.BIC,
			ABI:                 input.PaymentTerms.ABI,
			CAB:                 input.PaymentTerms.CAB,
			Beneficiario:        input.PaymentTerms.Beneficiario,
			IstitutoFinanziario: input.PaymentTerms.IstitutoFinanziario,
			DueDate:             input.PaymentTerms.DueDate,
		}
	}

	// Update stamp duty (bollo virtuale) if provided
	if input.DatiBollo != nil {
		invoice.DatiBollo = input.DatiBollo
	}

	// Save updates
	if err := s.invoiceRepo.Update(ctx, invoice); err != nil {
		return nil, err
	}

	return invoice, nil
}

func (s *invoiceService) DeleteInvoice(ctx context.Context, uuid string) error {
	invoice, err := s.invoiceRepo.GetByUUID(ctx, uuid)
	if err != nil {
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			return ErrInvoiceNotFound
		}
		return err
	}

	// Check if invoice can be deleted
	if !invoice.CanBeDeleted() {
		return ErrInvoiceCannotDelete
	}

	return s.invoiceRepo.SoftDelete(ctx, uuid)
}

func (s *invoiceService) SendInvoice(ctx context.Context, uuid string, sentBy string) (*models.SendInvoiceResponse, error) {
	invoice, err := s.invoiceRepo.GetByUUID(ctx, uuid)
	if err != nil {
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}

	// Check if invoice can be sent
	if !invoice.CanBeSent() {
		return nil, ErrInvoiceCannotSend
	}

	// Build XML
	xmlContent, err := s.xmlBuilder.Build(invoice)
	if err != nil {
		s.logger.Error("failed to build invoice XML",
			"invoiceUUID", uuid,
			"error", err,
		)
		return nil, err
	}

	// Store XML content
	invoice.XMLContent = xmlContent

	// Send to OpenAPI SDI
	response, err := s.openAPIClient.SendInvoice(ctx, invoice, xmlContent)
	if err != nil {
		s.logger.Error("failed to send invoice to SDI",
			"invoiceUUID", uuid,
			"error", err,
		)
		return nil, err
	}

	// Update invoice with OpenAPI data
	now := time.Now()
	invoice.OpenAPIUUID = response.UUID
	invoice.SDIIdentifier = response.SDIIdentifier
	invoice.Status = models.StatusSent
	invoice.SentAt = &now
	invoice.SentBy = sentBy

	if err := s.invoiceRepo.Update(ctx, invoice); err != nil {
		s.logger.Error("failed to update invoice after sending",
			"invoiceUUID", uuid,
			"error", err,
		)
		return nil, err
	}

	s.logger.Info("invoice sent to SDI",
		"invoiceUUID", uuid,
		"openAPIUUID", response.UUID,
		"sentBy", sentBy,
	)

	return &models.SendInvoiceResponse{
		InvoiceUUID:   uuid,
		OpenAPIUUID:   response.UUID,
		SDIIdentifier: response.SDIIdentifier,
		Status:        models.StatusSent,
		Message:       "Invoice sent successfully to SDI",
	}, nil
}

func (s *invoiceService) GetInvoicePDF(ctx context.Context, uuid string) ([]byte, error) {
	invoice, err := s.invoiceRepo.GetByUUID(ctx, uuid)
	if err != nil {
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}

	// If invoice has been sent, get PDF from OpenAPI
	if invoice.OpenAPIUUID != "" {
		return s.openAPIClient.DownloadInvoicePDF(ctx, invoice.OpenAPIUUID)
	}

	// For draft invoices, generate PDF locally using documents module
	if s.pdfService == nil {
		return nil, errors.New("PDF generation not available: documents module not configured")
	}

	// Convert invoice to template data
	data := s.invoiceToTemplateData(invoice)

	// Generate PDF (uses default invoice template if none specified)
	doc, err := s.pdfService.GenerateInvoicePDF(ctx, data, "", "system")
	if err != nil {
		s.logger.Error("Failed to generate invoice PDF",
			slog.String("invoiceUUID", uuid),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	// Get the PDF content
	content, _, err := s.pdfService.GetDocumentContent(ctx, doc.UUID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve PDF content: %w", err)
	}

	return content, nil
}

func (s *invoiceService) GetInvoiceXML(ctx context.Context, uuid string) (string, error) {
	invoice, err := s.invoiceRepo.GetByUUID(ctx, uuid)
	if err != nil {
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			return "", ErrInvoiceNotFound
		}
		return "", err
	}

	// For draft/pending invoices, always regenerate XML to include latest fixes
	// For sent/delivered invoices, use cached XML (it's what was actually sent)
	if invoice.Status == models.StatusDraft || invoice.Status == models.StatusPending {
		xmlContent, err := s.xmlBuilder.Build(invoice)
		if err != nil {
			return "", err
		}
		return ensureXMLDeclaration(xmlContent), nil
	}

	// For sent invoices, return cached XML if available
	if invoice.XMLContent != "" {
		return ensureXMLDeclaration(invoice.XMLContent), nil
	}

	// Fallback: generate XML on the fly
	xmlContent, err := s.xmlBuilder.Build(invoice)
	if err != nil {
		return "", err
	}

	// Ensure XML declaration is properly formatted
	return ensureXMLDeclaration(xmlContent), nil
}

// ensureXMLDeclaration ensures the XML has the proper declaration prolog
// FatturaPA requires: <?xml version="1.0" encoding="UTF-8"?>
// The prolog MUST be at byte 0 with no leading whitespace or BOM
func ensureXMLDeclaration(xmlContent string) string {
	const xmlDeclaration = `<?xml version="1.0" encoding="UTF-8"?>`

	// Remove any BOM (Byte Order Mark) that might be present
	// UTF-8 BOM is EF BB BF (239, 187, 191)
	content := strings.TrimPrefix(xmlContent, "\xef\xbb\xbf")

	// Trim all leading/trailing whitespace - prolog must be at byte 0
	trimmed := strings.TrimSpace(content)

	// Check if already has declaration
	if strings.HasPrefix(trimmed, "<?xml") {
		// Verify it has the correct format, replace if not
		if strings.HasPrefix(trimmed, xmlDeclaration) {
			// Return trimmed to ensure no leading whitespace
			return trimmed
		}
		// Has a declaration but might be malformed, find the end and replace
		endIdx := strings.Index(trimmed, "?>")
		if endIdx != -1 {
			rest := strings.TrimSpace(trimmed[endIdx+2:])
			return xmlDeclaration + "\n" + rest
		}
	}

	return xmlDeclaration + "\n" + trimmed
}

func (s *invoiceService) ListReceivedInvoices(ctx context.Context, filters *models.InvoiceFilters, pagination models.PaginationParams) (*models.InvoiceListResponse, error) {
	// Force direction to received
	direction := models.DirectionReceived
	if filters == nil {
		filters = &models.InvoiceFilters{}
	}
	filters.Direction = &direction

	return s.ListInvoices(ctx, filters, pagination)
}

func (s *invoiceService) AcceptReceivedInvoice(ctx context.Context, uuid string, acceptedBy string) error {
	invoice, err := s.invoiceRepo.GetByUUID(ctx, uuid)
	if err != nil {
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			return ErrInvoiceNotFound
		}
		return err
	}

	// Verify it's a received invoice
	if invoice.Direction != models.DirectionReceived {
		return errors.New("can only accept received invoices")
	}

	// Update status
	invoice.Status = models.StatusAccepted
	return s.invoiceRepo.Update(ctx, invoice)
}

func (s *invoiceService) RejectReceivedInvoice(ctx context.Context, uuid string, reason string, rejectedBy string) error {
	invoice, err := s.invoiceRepo.GetByUUID(ctx, uuid)
	if err != nil {
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			return ErrInvoiceNotFound
		}
		return err
	}

	// Verify it's a received invoice
	if invoice.Direction != models.DirectionReceived {
		return errors.New("can only reject received invoices")
	}

	// Update status
	invoice.Status = models.StatusRejected
	invoice.InternalNotes = reason
	return s.invoiceRepo.Update(ctx, invoice)
}

func (s *invoiceService) GetStats(ctx context.Context, fromDate, toDate time.Time) (*models.BillingStats, error) {
	return s.invoiceRepo.GetStats(ctx, fromDate, toDate)
}

// GetInvoiceHTML returns the HTML representation of an invoice from OpenAPI SDI
// Per OpenAPI SDI spec: GET /invoices/{uuid}/html
func (s *invoiceService) GetInvoiceHTML(ctx context.Context, uuid string) ([]byte, error) {
	invoice, err := s.invoiceRepo.GetByUUID(ctx, uuid)
	if err != nil {
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}

	// HTML is only available for invoices that have been sent to SDI
	if invoice.OpenAPIUUID == "" {
		return nil, ErrInvoiceHTMLNotReady
	}

	return s.openAPIClient.DownloadInvoiceHTML(ctx, invoice.OpenAPIUUID)
}

// ImportInvoice imports a supplier invoice via base64-encoded XML
// Per OpenAPI SDI spec: POST /invoices/import
func (s *invoiceService) ImportInvoice(ctx context.Context, input *models.ImportInvoiceInput) (*models.ImportInvoiceResponse, error) {
	// Call OpenAPI SDI to import the invoice
	openAPIInput := &ImportInvoiceInput{
		Invoice:         input.Invoice,
		InvoiceFileName: input.InvoiceFileName,
		SDIID:           input.SDIID,
		Metadata:        input.Metadata,
	}

	result, err := s.openAPIClient.ImportInvoice(ctx, openAPIInput)
	if err != nil {
		s.logger.Error("failed to import invoice",
			"error", err,
		)
		return nil, err
	}

	s.logger.Info("invoice imported successfully",
		"uuids", result.UUIDs,
		"count", result.Count,
	)

	return &models.ImportInvoiceResponse{
		UUIDs:   result.UUIDs,
		Count:   result.Count,
		Message: "Invoice(s) imported successfully",
	}, nil
}

// GetPreservedDocument retrieves the preservation status of a document
// Per OpenAPI SDI spec: GET /preserved_documents/{uuid}
func (s *invoiceService) GetPreservedDocument(ctx context.Context, uuid string) (*models.PreservedDocument, error) {
	result, err := s.openAPIClient.GetPreservedDocument(ctx, uuid)
	if err != nil {
		if errors.Is(err, ErrOpenAPINotFound) {
			return nil, errors.New("preserved document not found")
		}
		return nil, err
	}

	return &models.PreservedDocument{
		UUID:             result.UUID,
		Status:           result.Status,
		ReceiptTimestamp: result.ReceiptTimestamp,
		Weight:           result.Weight,
		ObjectID:         result.ObjectID,
		ObjectType:       result.ObjectType,
	}, nil
}

func (s *invoiceService) validateCreateInput(input *models.CreateInvoiceInput) error {
	if input.Number == "" {
		return ErrInvalidInvoiceData
	}
	if input.Date.IsZero() {
		return ErrInvalidInvoiceData
	}
	if len(input.Lines) == 0 {
		return ErrInvalidInvoiceData
	}
	return nil
}

// invoiceToTemplateData converts an Invoice model to template-compatible map[string]interface{}
// Includes all FatturaPA-specific fields for comprehensive PDF generation
func (s *invoiceService) invoiceToTemplateData(invoice *models.Invoice) map[string]interface{} {
	data := map[string]interface{}{
		"uuid":         invoice.UUID,
		"number":       invoice.Number,
		"date":         invoice.Date,
		"currency":     invoice.Currency,
		"documentType": string(invoice.DocumentType),

		// Totals
		"totalTaxable": invoice.TotalTaxableAmount,
		"totalVAT":     invoice.TotalVATAmount,
		"totalAmount":  invoice.TotalAmount,
		"rounding":     invoice.Rounding,
	}

	// Seller (CedentePrestatore)
	if invoice.CedentePrestatore != nil {
		seller := map[string]interface{}{
			"name":          getPartyName(invoice.CedentePrestatore),
			"address":       formatAddress(invoice.CedentePrestatore),
			"vatNumber":     invoice.CedentePrestatore.FiscalIDCode,
			"fiscalCode":    invoice.CedentePrestatore.CodiceFiscale,
			"email":         invoice.CedentePrestatore.Email,
			"phone":         invoice.CedentePrestatore.Phone,
			"pec":           invoice.CedentePrestatore.PEC,
			"regimeFiscale": string(invoice.CedentePrestatore.RegimeFiscale),
		}
		// REA registration (Italian company register)
		if invoice.CedentePrestatore.IscrizioneREA != nil {
			seller["rea"] = map[string]interface{}{
				"office":           invoice.CedentePrestatore.IscrizioneREA.Ufficio,
				"number":           invoice.CedentePrestatore.IscrizioneREA.NumeroREA,
				"capitale":         invoice.CedentePrestatore.IscrizioneREA.CapitaleSociale,
				"socioUnico":       invoice.CedentePrestatore.IscrizioneREA.SocioUnico,
				"statoLiquidazione": invoice.CedentePrestatore.IscrizioneREA.StatoLiquidazione,
			}
		}
		data["seller"] = seller
	}

	// Buyer (CessionarioCommittente)
	if invoice.CessionarioCommittente != nil {
		data["buyer"] = map[string]interface{}{
			"name":       getPartyName(invoice.CessionarioCommittente),
			"address":    formatAddress(invoice.CessionarioCommittente),
			"vatNumber":  invoice.CessionarioCommittente.FiscalIDCode,
			"fiscalCode": invoice.CessionarioCommittente.CodiceFiscale,
			"email":      invoice.CessionarioCommittente.Email,
		}
	}

	// Lines with full details
	lines := make([]map[string]interface{}, len(invoice.Lines))
	for i, line := range invoice.Lines {
		lineData := map[string]interface{}{
			"LineNumber":  line.LineNumber,
			"Description": line.Description,
			"Quantity":    line.Quantity,
			"UnitPrice":   line.UnitPrice,
			"VATRate":     line.VATRate,
			"VATNature":   string(line.VATNature),
			"TotalPrice":  line.TotalPrice,
			"VATAmount":   line.VATAmount,
			"Unit":        string(line.UnitOfMeasure),
			"Ritenuta":    line.Ritenuta,
		}
		// Discounts
		if len(line.Discounts) > 0 {
			discounts := make([]map[string]interface{}, len(line.Discounts))
			for j, d := range line.Discounts {
				discounts[j] = map[string]interface{}{
					"Type":       d.Type,
					"Percentage": d.Percentage,
					"Amount":     d.Amount,
				}
			}
			lineData["Discounts"] = discounts
		}
		lines[i] = lineData
	}
	data["lines"] = lines

	// VAT Summary by rate
	if len(invoice.VATSummary) > 0 {
		vatSummary := make([]map[string]interface{}, len(invoice.VATSummary))
		for i, vs := range invoice.VATSummary {
			vatSummary[i] = map[string]interface{}{
				"Rate":           vs.VATRate,
				"Nature":         string(vs.VATNature),
				"Taxable":        vs.TaxableAmount,
				"VAT":            vs.VATAmount,
				"Deductible":     vs.VATExigibility,
				"RifNormativo":   vs.NormativeRef,
			}
		}
		data["vatSummary"] = vatSummary
	}

	// Payment terms with installments
	if invoice.PaymentTerms != nil {
		payment := map[string]interface{}{
			"condition": string(invoice.PaymentTerms.Condition),
			"method":    string(invoice.PaymentTerms.PaymentMethod),
			"iban":      invoice.PaymentTerms.IBAN,
			"bic":       invoice.PaymentTerms.BIC,
		}
		if len(invoice.PaymentTerms.Installments) > 0 {
			installments := make([]map[string]interface{}, len(invoice.PaymentTerms.Installments))
			for i, inst := range invoice.PaymentTerms.Installments {
				installments[i] = map[string]interface{}{
					"DueDate": inst.DueDate,
					"Amount":  inst.Amount,
					"Paid":    inst.Paid,
				}
			}
			payment["installments"] = installments
			data["dueDate"] = invoice.PaymentTerms.Installments[0].DueDate
		} else if invoice.PaymentTerms.DueDate != nil {
			data["dueDate"] = *invoice.PaymentTerms.DueDate
		}
		data["payment"] = payment
		// Backward compatibility
		data["paymentTerms"] = getPaymentMethodDescription(invoice.PaymentTerms.PaymentMethod)
	}

	// Withholding tax (Ritenuta d'acconto)
	if len(invoice.DatiRitenuta) > 0 {
		ritenute := make([]map[string]interface{}, len(invoice.DatiRitenuta))
		var totalRitenuta float64
		for i, r := range invoice.DatiRitenuta {
			ritenute[i] = map[string]interface{}{
				"Type":       r.TipoRitenuta,
				"Rate":       r.AliquotaRitenuta,
				"Amount":     r.ImportoRitenuta,
				"CausalePag": r.CausalePagamento,
			}
			totalRitenuta += r.ImportoRitenuta
		}
		data["ritenuta"] = ritenute
		data["totalRitenuta"] = totalRitenuta
		// Calculate net payable (totale - ritenuta)
		data["netPayable"] = invoice.TotalAmount - totalRitenuta
	} else {
		data["netPayable"] = invoice.TotalAmount
	}

	// Stamp duty (Bollo)
	if invoice.DatiBollo != nil {
		data["bollo"] = map[string]interface{}{
			"virtual": true,
			"amount":  invoice.DatiBollo.ImportoBollo,
		}
	}

	// Social security fund (Cassa previdenziale)
	if len(invoice.DatiCassaPrevidenziale) > 0 {
		casse := make([]map[string]interface{}, len(invoice.DatiCassaPrevidenziale))
		var totalCassa float64
		for i, c := range invoice.DatiCassaPrevidenziale {
			casse[i] = map[string]interface{}{
				"Type":     c.TipoCassa,
				"Rate":     c.AlCassa,
				"Amount":   c.ImportoContributoCassa,
				"Taxable":  c.ImponibileCassa,
				"VATRate":  c.AliquotaIVA,
				"Ritenuta": c.Ritenuta,
				"Nature":   c.Natura,
			}
			totalCassa += c.ImportoContributoCassa
		}
		data["cassaPrevidenziale"] = casse
		data["totalCassa"] = totalCassa
	}

	// Causale (invoice description lines)
	if len(invoice.Causale) > 0 {
		data["causale"] = invoice.Causale
	}

	// Internal notes
	if invoice.InternalNotes != "" {
		data["notes"] = invoice.InternalNotes
	}

	return data
}

// getPartyName returns the display name for a party
func getPartyName(party *models.PartyData) string {
	if party.IsCompany && party.Denomination != "" {
		return party.Denomination
	}
	name := strings.TrimSpace(party.Name + " " + party.Surname)
	if name != "" {
		return name
	}
	return party.FiscalIDCode
}

// formatAddress formats a party's address for display
func formatAddress(party *models.PartyData) string {
	var parts []string

	// Street address with number
	address := party.Address
	if party.NumeroCivico != "" {
		address = party.Address + ", " + party.NumeroCivico
	}
	parts = append(parts, address)

	// City with postal code and province
	cityLine := party.PostalCode + " " + party.City
	if party.Province != "" {
		cityLine += " (" + party.Province + ")"
	}
	parts = append(parts, cityLine)

	return strings.Join(parts, " - ")
}

// getPaymentMethodDescription returns a human-readable description of the payment method
func getPaymentMethodDescription(method models.PaymentMethod) string {
	descriptions := map[models.PaymentMethod]string{
		"MP01": "Contanti",
		"MP02": "Assegno",
		"MP03": "Assegno circolare",
		"MP04": "Contanti presso Tesoreria",
		"MP05": "Bonifico",
		"MP06": "Vaglia cambiario",
		"MP07": "Bollettino bancario",
		"MP08": "Carta di pagamento",
		"MP09": "RID",
		"MP10": "RID utenze",
		"MP11": "RID veloce",
		"MP12": "RIBA",
		"MP13": "MAV",
		"MP14": "Quietanza erario",
		"MP15": "Giroconto su conti di contabilità speciale",
		"MP16": "Domiciliazione bancaria",
		"MP17": "Domiciliazione postale",
		"MP18": "Bollettino di c/c postale",
		"MP19": "SEPA Direct Debit",
		"MP20": "SEPA Direct Debit CORE",
		"MP21": "SEPA Direct Debit B2B",
		"MP22": "Trattenuta su somme già riscosse",
		"MP23": "PagoPA",
	}
	if desc, ok := descriptions[method]; ok {
		return desc
	}
	return string(method)
}
