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
	idTrasmittente := models.IdFiscale{
		IdPaese:  b.config.FiscalID[:2], // First 2 chars are country code (IT)
		IdCodice: b.config.FiscalID[2:], // Rest is the VAT number
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

	cp := models.CedentePrestatore{
		DatiAnagrafici: models.DatiAnagraficiCedente{
			IdFiscaleIVA: models.IdFiscale{
				IdPaese:  party.FiscalIDCountry,
				IdCodice: party.FiscalIDCode,
			},
			CodiceFiscale: party.CodiceFiscale,
			Anagrafica:    anagrafica,
			RegimeFiscale: party.RegimeFiscale,
		},
		Sede: models.Indirizzo{
			Indirizzo: party.Address,
			CAP:       party.PostalCode,
			Comune:    party.City,
			Provincia: party.Province,
			Nazione:   party.Country,
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

	cc := models.CessionarioCommittente{
		DatiAnagrafici: models.DatiAnagraficiCessionario{
			Anagrafica: anagrafica,
		},
		Sede: models.Indirizzo{
			Indirizzo: party.Address,
			CAP:       party.PostalCode,
			Comune:    party.City,
			Provincia: party.Province,
			Nazione:   party.Country,
		},
	}

	// Add fiscal ID if available
	if party.FiscalIDCode != "" {
		cc.DatiAnagrafici.IdFiscaleIVA = &models.IdFiscale{
			IdPaese:  party.FiscalIDCountry,
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
		}

		// Add esigibilità if present
		if vs.VATExigibility != "" {
			dr.EsigibilitaIVA = vs.VATExigibility
		}

		// Add normative reference if present
		if vs.NormativeRef != "" {
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
	// Format quantity, removing trailing zeros
	s := strconv.FormatFloat(qty, 'f', 8, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// GenerateProgressivoInvio generates a unique progressivo invio
func GenerateProgressivoInvio() string {
	// Format: YYYYMMDD + 5 char random string
	now := time.Now()
	return fmt.Sprintf("%s%05d", now.Format("20060102"), now.UnixNano()%100000)
}
