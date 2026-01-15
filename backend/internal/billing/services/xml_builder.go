package services

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/orkestra/backend/internal/billing/config"
	"github.com/orkestra/backend/internal/billing/models"
)

// XMLBuilder builds FatturaPA XML documents
type XMLBuilder interface {
	Build(invoice *models.Invoice) (string, error)
}

type xmlBuilder struct {
	config *config.OpenAPIConfig
}

// NewXMLBuilder creates a new XML builder
func NewXMLBuilder(cfg *config.OpenAPIConfig) XMLBuilder {
	return &xmlBuilder{
		config: cfg,
	}
}

func (b *xmlBuilder) Build(invoice *models.Invoice) (string, error) {
	// Determine format based on recipient type
	format := models.FormatFPR12 // Private sector
	if invoice.CessionarioCommittente != nil && len(invoice.CessionarioCommittente.CodiceDestinatario) == 6 {
		format = models.FormatFPA12 // Public Administration
	}

	fatturaPA := models.NewFatturaElettronica(format)

	// Build header
	fatturaPA.FatturaElettronicaHeader = b.buildHeader(invoice, format)

	// Build body
	fatturaPA.FatturaElettronicaBody = []models.FatturaElettronicaBody{
		b.buildBody(invoice),
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(fatturaPA, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal XML: %w", err)
	}

	// Add XML declaration
	xmlContent := xml.Header + string(output)

	return xmlContent, nil
}

func (b *xmlBuilder) buildHeader(invoice *models.Invoice, format models.TransmissionFormat) models.FatturaElettronicaHeader {
	header := models.FatturaElettronicaHeader{
		DatiTrasmissione:       b.buildDatiTrasmissione(invoice, format),
		CedentePrestatore:      b.buildCedentePrestatore(invoice.CedentePrestatore),
		CessionarioCommittente: b.buildCessionarioCommittente(invoice.CessionarioCommittente),
	}

	return header
}

func (b *xmlBuilder) buildDatiTrasmissione(invoice *models.Invoice, format models.TransmissionFormat) models.DatiTrasmissione {
	// Use transmitter's fiscal ID (our company)
	// FiscalID should be in format "ITXXXXXXX" where IT is the country code
	idPaese := "IT"
	idCodice := b.config.FiscalID
	if len(b.config.FiscalID) >= 2 {
		// Check if first 2 chars are letters (country code)
		if isAlpha(b.config.FiscalID[:2]) {
			idPaese = strings.ToUpper(b.config.FiscalID[:2])
			idCodice = b.config.FiscalID[2:]
		}
	}
	idTrasmittente := models.IdFiscale{
		IdPaese:  idPaese,
		IdCodice: idCodice,
	}

	// Determine recipient code
	codiceDestinatario := b.config.RecipientCode // Default to our OpenAPI recipient code
	pecDestinatario := ""

	if invoice.CessionarioCommittente != nil {
		if invoice.CessionarioCommittente.CodiceDestinatario != "" {
			codiceDestinatario = invoice.CessionarioCommittente.CodiceDestinatario
		} else if invoice.CessionarioCommittente.PECDestinatario != "" {
			codiceDestinatario = "0000000" // When using PEC, code is 7 zeros
			pecDestinatario = invoice.CessionarioCommittente.PECDestinatario
		}
	}

	dt := models.DatiTrasmissione{
		IdTrasmittente:      idTrasmittente,
		ProgressivoInvio:    invoice.ProgressivoInvio,
		FormatoTrasmissione: format,
		CodiceDestinatario:  codiceDestinatario,
	}

	if pecDestinatario != "" {
		dt.PECDestinatario = pecDestinatario
	}

	return dt
}

func (b *xmlBuilder) buildCedentePrestatore(party *models.PartyData) models.CedentePrestatore {
	if party == nil {
		return models.CedentePrestatore{}
	}

	anagrafica := models.Anagrafica{}
	if party.IsCompany {
		anagrafica.Denominazione = party.Denomination
	} else {
		anagrafica.Nome = party.Name
		anagrafica.Cognome = party.Surname
	}

	// Ensure IdPaese is uppercase 2-letter country code
	idPaese := ensureIdPaese(party.FiscalIDCountry)

	// Validate RegimeFiscale - must be a valid code (RF01-RF19, RF20)
	regimeFiscale := party.RegimeFiscale
	if regimeFiscale == "" {
		regimeFiscale = models.RegimeOrdinario // Default to RF01
	}

	// Ensure Nazione is uppercase 2-letter country code
	nazione := ensureIdPaese(party.Country)

	cp := models.CedentePrestatore{
		DatiAnagrafici: models.DatiAnagraficiCedente{
			IdFiscaleIVA: models.IdFiscale{
				IdPaese:  idPaese,
				IdCodice: party.FiscalIDCode,
			},
			CodiceFiscale: party.CodiceFiscale,
			Anagrafica:    anagrafica,
			RegimeFiscale: regimeFiscale,
		},
		Sede: models.Indirizzo{
			Indirizzo: party.Address,
			CAP:       party.PostalCode,
			Comune:    party.City,
			Provincia: party.Province,
			Nazione:   nazione,
		},
	}

	// Add contacts if available
	if party.Phone != "" || party.Email != "" {
		cp.Contatti = &models.Contatti{
			Telefono: party.Phone,
			Email:    party.Email,
		}
	}

	return cp
}

func (b *xmlBuilder) buildCessionarioCommittente(party *models.PartyData) models.CessionarioCommittente {
	if party == nil {
		return models.CessionarioCommittente{}
	}

	anagrafica := models.Anagrafica{}
	if party.IsCompany {
		anagrafica.Denominazione = party.Denomination
	} else {
		anagrafica.Nome = party.Name
		anagrafica.Cognome = party.Surname
	}

	// Ensure Nazione is uppercase 2-letter country code
	nazione := ensureIdPaese(party.Country)

	cc := models.CessionarioCommittente{
		DatiAnagrafici: models.DatiAnagraficiCessionario{
			Anagrafica: anagrafica,
		},
		Sede: models.Indirizzo{
			Indirizzo: party.Address,
			CAP:       party.PostalCode,
			Comune:    party.City,
			Provincia: party.Province,
			Nazione:   nazione,
		},
	}

	// Add fiscal ID if available
	if party.FiscalIDCode != "" {
		idPaese := ensureIdPaese(party.FiscalIDCountry)
		cc.DatiAnagrafici.IdFiscaleIVA = &models.IdFiscale{
			IdPaese:  idPaese,
			IdCodice: party.FiscalIDCode,
		}
	}

	// Add codice fiscale if different
	if party.CodiceFiscale != "" {
		cc.DatiAnagrafici.CodiceFiscale = party.CodiceFiscale
	}

	return cc
}

func (b *xmlBuilder) buildBody(invoice *models.Invoice) models.FatturaElettronicaBody {
	body := models.FatturaElettronicaBody{
		DatiGenerali:    b.buildDatiGenerali(invoice),
		DatiBeniServizi: b.buildDatiBeniServizi(invoice),
	}

	// Add payment data if present
	if invoice.PaymentTerms != nil {
		body.DatiPagamento = b.buildDatiPagamento(invoice)
	}

	// Add attachments if present
	if len(invoice.Attachments) > 0 {
		body.Allegati = b.buildAllegati(invoice.Attachments)
	}

	return body
}

func (b *xmlBuilder) buildDatiGenerali(invoice *models.Invoice) models.DatiGenerali {
	dg := models.DatiGenerali{
		DatiGeneraliDocumento: models.DatiGeneraliDocumento{
			TipoDocumento:          invoice.DocumentType,
			Divisa:                 invoice.Currency,
			Data:                   invoice.Date.Format("2006-01-02"),
			Numero:                 invoice.Number,
			ImportoTotaleDocumento: formatAmount(invoice.TotalAmount),
		},
	}

	// Add rounding if present
	if invoice.Rounding != 0 {
		dg.DatiGeneraliDocumento.Arrotondamento = formatAmount(invoice.Rounding)
	}

	// Add causale (description)
	if len(invoice.Causale) > 0 {
		dg.DatiGeneraliDocumento.Causale = invoice.Causale
	}

	// Add related documents
	for _, rd := range invoice.RelatedDocuments {
		docCorr := models.DatiDocumentoCorrelato{
			IdDocumento: rd.Number,
		}
		if rd.Date != nil {
			docCorr.Data = rd.Date.Format("2006-01-02")
		}
		if rd.CIG != "" {
			docCorr.CodiceCIG = rd.CIG
		}
		if rd.CUP != "" {
			docCorr.CodiceCUP = rd.CUP
		}

		switch rd.Type {
		case "ordine":
			dg.DatiOrdineAcquisto = append(dg.DatiOrdineAcquisto, docCorr)
		case "contratto":
			dg.DatiContratto = append(dg.DatiContratto, docCorr)
		case "convenzione":
			dg.DatiConvenzione = append(dg.DatiConvenzione, docCorr)
		case "ricezione":
			dg.DatiRicezione = append(dg.DatiRicezione, docCorr)
		case "fattura":
			dg.DatiFattureCollegate = append(dg.DatiFattureCollegate, docCorr)
		}
	}

	return dg
}

func (b *xmlBuilder) buildDatiBeniServizi(invoice *models.Invoice) models.DatiBeniServizi {
	dbs := models.DatiBeniServizi{
		DettaglioLinee: make([]models.DettaglioLinea, 0, len(invoice.Lines)),
		DatiRiepilogo:  make([]models.DatiRiepilogo, 0, len(invoice.VATSummary)),
	}

	// Build line details
	for i, line := range invoice.Lines {
		dl := models.DettaglioLinea{
			NumeroLinea:    i + 1,
			Descrizione:    truncateString(line.Description, 1000),
			PrezzoUnitario: formatAmount(line.UnitPrice),
			PrezzoTotale:   formatAmount(line.TotalPrice),
			AliquotaIVA:    formatAmount(line.VATRate),
		}

		// Add quantity if present
		if line.Quantity > 0 {
			dl.Quantita = formatQuantity(line.Quantity)
		}

		// Add unit of measure if present
		if line.UnitOfMeasure != "" {
			dl.UnitaMisura = string(line.UnitOfMeasure)
		}

		// Add natura IVA if rate is 0
		if line.VATRate == 0 && line.VATNature != "" {
			dl.Natura = string(line.VATNature)
		}

		// Add period dates if present
		if line.StartDate != nil {
			dl.DataInizioPeriodo = line.StartDate.Format("2006-01-02")
		}
		if line.EndDate != nil {
			dl.DataFinePeriodo = line.EndDate.Format("2006-01-02")
		}

		// Add product code if present
		if line.ProductCode != "" {
			dl.CodiceArticolo = []models.CodArticolo{
				{
					CodiceTipo:   "INTERNO",
					CodiceValore: line.ProductCode,
				},
			}
		}

		// Add discounts/markups
		for _, discount := range line.Discounts {
			sm := models.ScontoMagg{
				Tipo: discount.Type,
			}
			if discount.Percentage > 0 {
				sm.Percentuale = formatAmount(discount.Percentage)
			}
			if discount.Amount > 0 {
				sm.Importo = formatAmount(discount.Amount)
			}
			dl.ScontoMaggiorazione = append(dl.ScontoMaggiorazione, sm)
		}

		dbs.DettaglioLinee = append(dbs.DettaglioLinee, dl)
	}

	// Build VAT summary
	for _, vs := range invoice.VATSummary {
		dr := models.DatiRiepilogo{
			AliquotaIVA:       formatAmount(vs.VATRate),
			ImponibileImporto: formatAmount(vs.TaxableAmount),
			Imposta:           formatAmount(vs.VATAmount),
		}

		// Add natura if rate is 0
		if vs.VATRate == 0 && vs.VATNature != "" {
			dr.Natura = string(vs.VATNature)
			// RiferimentoNormativo is REQUIRED when AliquotaIVA is 0
			if vs.NormativeRef != "" {
				dr.RiferimentoNormativo = vs.NormativeRef
			} else {
				// Provide a default normative reference based on Natura
				dr.RiferimentoNormativo = getDefaultNormativeRef(string(vs.VATNature))
			}
		}

		// Add esigibilità if present
		if vs.VATExigibility != "" {
			dr.EsigibilitaIVA = vs.VATExigibility
		}

		// Add normative reference if present (for non-zero rates with specific references)
		if vs.VATRate != 0 && vs.NormativeRef != "" {
			dr.RiferimentoNormativo = vs.NormativeRef
		}

		dbs.DatiRiepilogo = append(dbs.DatiRiepilogo, dr)
	}

	return dbs
}

func (b *xmlBuilder) buildDatiPagamento(invoice *models.Invoice) *models.DatiPagamento {
	if invoice.PaymentTerms == nil {
		return nil
	}

	pt := invoice.PaymentTerms

	dp := &models.DatiPagamento{
		CondizioniPagamento: string(pt.Condition),
		DettaglioPagamento:  []models.DettaglioPagamento{},
	}

	// For installment payments
	if pt.Condition == models.PaymentConditionRata && len(pt.Installments) > 0 {
		for _, inst := range pt.Installments {
			detail := models.DettaglioPagamento{
				ModalitaPagamento:     string(pt.PaymentMethod),
				DataScadenzaPagamento: inst.DueDate.Format("2006-01-02"),
				ImportoPagamento:      formatAmount(inst.Amount),
			}
			if pt.IBAN != "" {
				detail.IBAN = pt.IBAN
			}
			if pt.BIC != "" {
				detail.BIC = pt.BIC
			}
			dp.DettaglioPagamento = append(dp.DettaglioPagamento, detail)
		}
	} else {
		// Single payment
		detail := models.DettaglioPagamento{
			ModalitaPagamento: string(pt.PaymentMethod),
			ImportoPagamento:  formatAmount(invoice.TotalAmount),
		}
		if pt.DueDate != nil {
			detail.DataScadenzaPagamento = pt.DueDate.Format("2006-01-02")
		}
		if pt.IBAN != "" {
			detail.IBAN = pt.IBAN
		}
		if pt.BIC != "" {
			detail.BIC = pt.BIC
		}
		dp.DettaglioPagamento = append(dp.DettaglioPagamento, detail)
	}

	return dp
}

func (b *xmlBuilder) buildAllegati(attachments []models.InvoiceAttachment) []models.Allegato {
	allegati := make([]models.Allegato, 0, len(attachments))

	for _, att := range attachments {
		allegato := models.Allegato{
			NomeAttachment:        att.Name,
			DescrizioneAttachment: att.Description,
			FormatoAttachment:     att.Format,
			Attachment:            att.Content, // Base64 encoded
		}
		allegati = append(allegati, allegato)
	}

	return allegati
}

// Helper functions

func formatAmount(amount float64) string {
	// Format with 2 decimal places, using dot as decimal separator
	return strconv.FormatFloat(amount, 'f', 2, 64)
}

func formatQuantity(qty float64) string {
	// FatturaPA requires minimum 2 decimals, maximum 8 decimals for Quantita
	s := strconv.FormatFloat(qty, 'f', 8, 64)
	// Remove trailing zeros but keep at least 2 decimal places
	s = strings.TrimRight(s, "0")
	// Ensure minimum 2 decimal places
	parts := strings.Split(s, ".")
	if len(parts) == 1 {
		// No decimal point, add .00
		return s + ".00"
	}
	if len(parts[1]) < 2 {
		// Less than 2 decimals, pad with zeros
		return s + strings.Repeat("0", 2-len(parts[1]))
	}
	return s
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// isAlpha checks if a string contains only alphabetic characters
func isAlpha(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return false
		}
	}
	return len(s) > 0
}

// ensureIdPaese ensures IdPaese is uppercase 2-letter country code
func ensureIdPaese(s string) string {
	if s == "" {
		return "IT" // Default to Italy
	}
	return strings.ToUpper(s)
}

// getDefaultNormativeRef returns a default normative reference based on VAT Nature code
func getDefaultNormativeRef(natura string) string {
	switch natura {
	case "N1":
		return "Art. 15 DPR 633/72"
	case "N2.1":
		return "Artt. 7-7 septies DPR 633/72"
	case "N2.2":
		return "Art. 7 DPR 633/72"
	case "N3.1":
		return "Art. 8 c.1 lett.a DPR 633/72"
	case "N3.2":
		return "Art. 8 c.1 lett.b DPR 633/72"
	case "N3.3":
		return "Art. 8-bis DPR 633/72"
	case "N3.4":
		return "Art. 41 DL 331/93"
	case "N3.5":
		return "Art. 8 c.1 lett.c DPR 633/72"
	case "N3.6":
		return "Art. 9 DPR 633/72"
	case "N4":
		return "Art. 10 DPR 633/72"
	case "N5":
		return "Art. 36 DL 41/95"
	case "N6.1":
		return "Art. 74-ter DPR 633/72"
	case "N6.2":
		return "Art. 17 c.6 DPR 633/72"
	case "N6.3":
		return "Art. 17 c.6 lett.a-bis DPR 633/72"
	case "N6.4":
		return "Art. 17 c.6 lett.b DPR 633/72"
	case "N6.5":
		return "Art. 17 c.6 lett.c DPR 633/72"
	case "N6.6":
		return "Art. 17 c.5 DPR 633/72"
	case "N6.7":
		return "Art. 17 c.6 lett.d-bis/d-ter/d-quater DPR 633/72"
	case "N6.8":
		return "Art. 17 c.6 lett.d DPR 633/72"
	case "N6.9":
		return "Art. 17 DPR 633/72"
	case "N7":
		return "Art. 7-bis DPR 633/72"
	default:
		return "Operazione non soggetta ad IVA"
	}
}

// GenerateProgressivoInvio generates a unique progressivo invio
// Max 10 characters alphanumeric as per FatturaPA spec
func GenerateProgressivoInvio() string {
	// Format: 5 chars from timestamp + 5 chars random = 10 chars max
	now := time.Now()
	// Use last 5 digits of unix timestamp + 5 random digits
	return fmt.Sprintf("%05d%05d", now.Unix()%100000, now.UnixNano()%100000)
}
