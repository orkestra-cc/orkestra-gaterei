package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/orkestra/backend/internal/documents/models"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Errors for template engine
var (
	ErrTemplateParseError  = errors.New("failed to parse template")
	ErrTemplateRenderError = errors.New("failed to render template")
	ErrInvalidTemplate     = errors.New("invalid template content")
)

// TemplateEngine defines the interface for template rendering
type TemplateEngine interface {
	// Render renders a template with the given data
	Render(ctx context.Context, tmpl *models.Template, data map[string]interface{}) (string, error)
	// RenderString renders a raw HTML template string with the given data
	RenderString(ctx context.Context, htmlContent string, cssContent string, data map[string]interface{}) (string, error)
	// Validate validates a template's HTML content
	Validate(ctx context.Context, htmlContent string) error
}

type templateEngine struct {
	funcMap template.FuncMap
}

// NewTemplateEngine creates a new template engine with built-in functions
func NewTemplateEngine() TemplateEngine {
	return &templateEngine{
		funcMap: createTemplateFuncMap(),
	}
}

// createTemplateFuncMap creates the template function map with all built-in functions
func createTemplateFuncMap() template.FuncMap {
	// Italian locale printer for number formatting
	italianPrinter := message.NewPrinter(language.Italian)

	return template.FuncMap{
		// Date formatting
		"formatDate": func(t time.Time, layout string) string {
			return t.Format(layout)
		},
		"formatDateIT": func(t time.Time) string {
			return t.Format("02/01/2006")
		},
		"formatDateISO": func(t time.Time) string {
			return t.Format("2006-01-02")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("02/01/2006 15:04")
		},
		"now": func() time.Time {
			return time.Now()
		},

		// Money formatting
		"formatMoney": func(amount float64, decimals int, symbol string) string {
			format := fmt.Sprintf("%%.%df %s", decimals, symbol)
			return fmt.Sprintf(format, amount)
		},
		"formatMoneyEUR": func(amount float64) string {
			// Format with Italian locale (1.234,56 EUR)
			return fmt.Sprintf("%s EUR", italianPrinter.Sprintf("%.2f", amount))
		},
		"formatMoneySymbol": func(amount float64, symbol string) string {
			return fmt.Sprintf("%s %.2f", symbol, amount)
		},

		// Number formatting
		"formatNumber": func(value float64, decimals int) string {
			format := fmt.Sprintf("%%.%df", decimals)
			return fmt.Sprintf(format, value)
		},
		"formatNumberIT": func(value float64, decimals int) string {
			return italianPrinter.Sprintf(fmt.Sprintf("%%.%df", decimals), value)
		},
		"formatPercent": func(value float64, decimals int) string {
			format := fmt.Sprintf("%%.%df%%", decimals)
			return fmt.Sprintf(format, value)
		},

		// String manipulation
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"trim":  strings.TrimSpace,
		"title": strings.Title,
		"replace": func(s, old, new string) string {
			return strings.ReplaceAll(s, old, new)
		},
		"truncate": func(s string, maxLen int) string {
			if len(s) <= maxLen {
				return s
			}
			return s[:maxLen] + "..."
		},
		"split": strings.Split,
		"join":  strings.Join,

		// Line numbering (for tables)
		"lineNumber": func(i int) int {
			return i + 1
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"div": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},

		// Conditional helpers
		"default": func(defaultVal, val interface{}) interface{} {
			if val == nil || val == "" || val == 0 {
				return defaultVal
			}
			return val
		},
		"coalesce": func(values ...interface{}) interface{} {
			for _, v := range values {
				if v != nil && v != "" {
					return v
				}
			}
			return nil
		},
		"ifelse": func(cond bool, trueVal, falseVal interface{}) interface{} {
			if cond {
				return trueVal
			}
			return falseVal
		},

		// Safe HTML output
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeCSS": func(s string) template.CSS {
			return template.CSS(s)
		},

		// Array/slice helpers
		"first": func(items []interface{}) interface{} {
			if len(items) > 0 {
				return items[0]
			}
			return nil
		},
		"last": func(items []interface{}) interface{} {
			if len(items) > 0 {
				return items[len(items)-1]
			}
			return nil
		},
		"len": func(items interface{}) int {
			switch v := items.(type) {
			case []interface{}:
				return len(v)
			case string:
				return len(v)
			case map[string]interface{}:
				return len(v)
			default:
				return 0
			}
		},

		// Comparison
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"ne": func(a, b interface{}) bool {
			return a != b
		},
		"lt": func(a, b float64) bool {
			return a < b
		},
		"le": func(a, b float64) bool {
			return a <= b
		},
		"gt": func(a, b float64) bool {
			return a > b
		},
		"ge": func(a, b float64) bool {
			return a >= b
		},
	}
}

// Render renders a template with the given data
func (e *templateEngine) Render(ctx context.Context, tmpl *models.Template, data map[string]interface{}) (string, error) {
	return e.RenderString(ctx, tmpl.HTMLContent, tmpl.CSSContent, data)
}

// RenderString renders a raw HTML template string with the given data
func (e *templateEngine) RenderString(ctx context.Context, htmlContent string, cssContent string, data map[string]interface{}) (string, error) {
	// If CSS is provided, inject it into the HTML
	fullHTML := htmlContent
	if cssContent != "" {
		// Check if there's a <head> tag to inject CSS
		if strings.Contains(strings.ToLower(htmlContent), "<head>") {
			styleTag := fmt.Sprintf("<style>%s</style>", cssContent)
			fullHTML = strings.Replace(htmlContent, "</head>", styleTag+"</head>", 1)
		} else if strings.Contains(strings.ToLower(htmlContent), "<html>") {
			// Inject after <html> tag
			styleTag := fmt.Sprintf("<head><style>%s</style></head>", cssContent)
			fullHTML = strings.Replace(htmlContent, "<html>", "<html>"+styleTag, 1)
		} else {
			// Wrap with basic HTML structure
			fullHTML = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<style>%s</style>
</head>
<body>
%s
</body>
</html>`, cssContent, htmlContent)
		}
	}

	// Parse template
	t, err := template.New("document").Funcs(e.funcMap).Parse(fullHTML)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrTemplateParseError, err.Error())
	}

	// Render template
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("%w: %s", ErrTemplateRenderError, err.Error())
	}

	return buf.String(), nil
}

// Validate validates a template's HTML content
func (e *templateEngine) Validate(ctx context.Context, htmlContent string) error {
	if htmlContent == "" {
		return ErrInvalidTemplate
	}

	// Try to parse the template
	_, err := template.New("validate").Funcs(e.funcMap).Parse(htmlContent)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrTemplateParseError, err.Error())
	}

	return nil
}

// GetTemplateVariables returns available variables for a template type
func GetTemplateVariables(templateType models.TemplateType) models.TemplateVariablesResponse {
	var groups []models.TemplateVariableGroup

	switch templateType {
	case models.TemplateTypeInvoice:
		groups = getInvoiceVariables()
	case models.TemplateTypeOffer:
		groups = getOfferVariables()
	case models.TemplateTypeReceipt:
		groups = getReceiptVariables()
	default:
		groups = getCommonVariables()
	}

	return models.TemplateVariablesResponse{
		TemplateType: templateType,
		Groups:       groups,
	}
}

func getInvoiceVariables() []models.TemplateVariableGroup {
	return []models.TemplateVariableGroup{
		{
			Name:        "Document Info",
			Description: "Invoice document information",
			Variables: []models.TemplateVariableInfo{
				{Name: ".number", Description: "Invoice number", Type: "string", Required: true},
				{Name: ".date", Description: "Invoice date", Type: "date", Required: true},
				{Name: ".dueDate", Description: "Payment due date", Type: "date", Required: false},
				{Name: ".currency", Description: "Currency code (EUR, USD)", Type: "string", Required: false},
				{Name: ".notes", Description: "Invoice notes", Type: "string", Required: false},
				{Name: ".paymentTerms", Description: "Payment terms text", Type: "string", Required: false},
			},
		},
		{
			Name:        "Seller",
			Description: "Seller/company information",
			Variables: []models.TemplateVariableInfo{
				{Name: ".seller.name", Description: "Company name", Type: "string", Required: true},
				{Name: ".seller.address", Description: "Full address", Type: "string", Required: true},
				{Name: ".seller.vatNumber", Description: "VAT number (P.IVA)", Type: "string", Required: true},
				{Name: ".seller.fiscalCode", Description: "Fiscal code", Type: "string", Required: false},
				{Name: ".seller.email", Description: "Email address", Type: "string", Required: false},
				{Name: ".seller.phone", Description: "Phone number", Type: "string", Required: false},
				{Name: ".seller.pec", Description: "PEC email", Type: "string", Required: false},
			},
		},
		{
			Name:        "Buyer",
			Description: "Customer/buyer information",
			Variables: []models.TemplateVariableInfo{
				{Name: ".buyer.name", Description: "Customer name", Type: "string", Required: true},
				{Name: ".buyer.address", Description: "Full address", Type: "string", Required: true},
				{Name: ".buyer.vatNumber", Description: "VAT number (if applicable)", Type: "string", Required: false},
				{Name: ".buyer.fiscalCode", Description: "Fiscal code", Type: "string", Required: false},
				{Name: ".buyer.email", Description: "Email address", Type: "string", Required: false},
			},
		},
		{
			Name:        "Line Items",
			Description: "Invoice line items (use with range)",
			Variables: []models.TemplateVariableInfo{
				{Name: ".lines", Description: "Array of line items", Type: "array", Required: true},
				{Name: "$line.Description", Description: "Item description", Type: "string", Required: true},
				{Name: "$line.Quantity", Description: "Quantity", Type: "number", Required: true},
				{Name: "$line.UnitPrice", Description: "Unit price", Type: "number", Required: true},
				{Name: "$line.VATRate", Description: "VAT rate percentage", Type: "number", Required: true},
				{Name: "$line.TotalPrice", Description: "Line total (before VAT)", Type: "number", Required: true},
			},
		},
		{
			Name:        "Totals",
			Description: "Invoice totals",
			Variables: []models.TemplateVariableInfo{
				{Name: ".totalTaxable", Description: "Total taxable amount", Type: "number", Required: true},
				{Name: ".totalVAT", Description: "Total VAT amount", Type: "number", Required: true},
				{Name: ".totalAmount", Description: "Grand total", Type: "number", Required: true},
				{Name: ".discount", Description: "Discount amount", Type: "number", Required: false},
			},
		},
	}
}

func getOfferVariables() []models.TemplateVariableGroup {
	return []models.TemplateVariableGroup{
		{
			Name:        "Document Info",
			Description: "Offer document information",
			Variables: []models.TemplateVariableInfo{
				{Name: ".number", Description: "Offer number", Type: "string", Required: true},
				{Name: ".date", Description: "Offer date", Type: "date", Required: true},
				{Name: ".validUntil", Description: "Offer validity date", Type: "date", Required: false},
				{Name: ".subject", Description: "Offer subject/title", Type: "string", Required: false},
				{Name: ".notes", Description: "Offer notes", Type: "string", Required: false},
			},
		},
		{
			Name:        "Company",
			Description: "Company information",
			Variables: []models.TemplateVariableInfo{
				{Name: ".company.name", Description: "Company name", Type: "string", Required: true},
				{Name: ".company.address", Description: "Full address", Type: "string", Required: true},
				{Name: ".company.vatNumber", Description: "VAT number", Type: "string", Required: false},
				{Name: ".company.email", Description: "Email address", Type: "string", Required: false},
				{Name: ".company.phone", Description: "Phone number", Type: "string", Required: false},
			},
		},
		{
			Name:        "Customer",
			Description: "Customer information",
			Variables: []models.TemplateVariableInfo{
				{Name: ".customer.name", Description: "Customer name", Type: "string", Required: true},
				{Name: ".customer.address", Description: "Full address", Type: "string", Required: false},
				{Name: ".customer.email", Description: "Email address", Type: "string", Required: false},
			},
		},
		{
			Name:        "Items",
			Description: "Offer line items",
			Variables: []models.TemplateVariableInfo{
				{Name: ".items", Description: "Array of items", Type: "array", Required: true},
				{Name: "$item.Description", Description: "Item description", Type: "string", Required: true},
				{Name: "$item.Quantity", Description: "Quantity", Type: "number", Required: true},
				{Name: "$item.UnitPrice", Description: "Unit price", Type: "number", Required: true},
				{Name: "$item.Total", Description: "Item total", Type: "number", Required: true},
			},
		},
		{
			Name:        "Totals",
			Description: "Offer totals",
			Variables: []models.TemplateVariableInfo{
				{Name: ".subtotal", Description: "Subtotal before tax", Type: "number", Required: true},
				{Name: ".tax", Description: "Tax amount", Type: "number", Required: false},
				{Name: ".total", Description: "Grand total", Type: "number", Required: true},
			},
		},
	}
}

func getReceiptVariables() []models.TemplateVariableGroup {
	return []models.TemplateVariableGroup{
		{
			Name:        "Receipt Info",
			Description: "Receipt document information",
			Variables: []models.TemplateVariableInfo{
				{Name: ".number", Description: "Receipt number", Type: "string", Required: true},
				{Name: ".date", Description: "Receipt date", Type: "date", Required: true},
				{Name: ".paymentMethod", Description: "Payment method used", Type: "string", Required: false},
			},
		},
		{
			Name:        "Merchant",
			Description: "Merchant information",
			Variables: []models.TemplateVariableInfo{
				{Name: ".merchant.name", Description: "Business name", Type: "string", Required: true},
				{Name: ".merchant.address", Description: "Address", Type: "string", Required: false},
			},
		},
		{
			Name:        "Items",
			Description: "Receipt items",
			Variables: []models.TemplateVariableInfo{
				{Name: ".items", Description: "Array of items", Type: "array", Required: true},
				{Name: "$item.Name", Description: "Item name", Type: "string", Required: true},
				{Name: "$item.Price", Description: "Item price", Type: "number", Required: true},
				{Name: "$item.Quantity", Description: "Quantity", Type: "number", Required: false},
			},
		},
		{
			Name:        "Totals",
			Description: "Receipt totals",
			Variables: []models.TemplateVariableInfo{
				{Name: ".total", Description: "Total amount", Type: "number", Required: true},
			},
		},
	}
}

func getCommonVariables() []models.TemplateVariableGroup {
	return []models.TemplateVariableGroup{
		{
			Name:        "Custom Data",
			Description: "Custom data can be passed with any keys",
			Variables: []models.TemplateVariableInfo{
				{Name: ".<key>", Description: "Access any custom key from the data object", Type: "any", Required: false},
			},
		},
	}
}
