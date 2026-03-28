package sales

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/orkestra/backend/internal/sales/handlers"
)

func op(id, method, path, summary string) huma.Operation {
	return huma.Operation{
		OperationID: id,
		Method:      method,
		Path:        path,
		Summary:     summary,
		Tags:        []string{"Sales Intelligence"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}
}

// RegisterSkillRoutes registers individual skill endpoints
func RegisterSkillRoutes(api huma.API, h *handlers.SkillHandler) {
	huma.Register(api, op("sales-research", http.MethodPost, "/v1/sales/research", "Company research & firmographics"), h.Research)
	huma.Register(api, op("sales-qualify", http.MethodPost, "/v1/sales/qualify", "BANT + MEDDIC lead qualification"), h.Qualify)
	huma.Register(api, op("sales-contacts", http.MethodPost, "/v1/sales/contacts", "Decision maker identification"), h.Contacts)
	huma.Register(api, op("sales-outreach", http.MethodPost, "/v1/sales/outreach", "Generate outreach email sequence"), h.Outreach)
	huma.Register(api, op("sales-followup", http.MethodPost, "/v1/sales/followup", "Generate follow-up sequence"), h.Followup)
	huma.Register(api, op("sales-prep", http.MethodPost, "/v1/sales/prep", "Meeting preparation brief"), h.Prep)
	huma.Register(api, op("sales-proposal", http.MethodPost, "/v1/sales/proposal", "Client proposal generation"), h.Proposal)
	huma.Register(api, op("sales-objections", http.MethodPost, "/v1/sales/objections", "Objection handling playbook"), h.Objections)
	huma.Register(api, op("sales-icp", http.MethodPost, "/v1/sales/icp", "Ideal Customer Profile builder"), h.ICP)
	huma.Register(api, op("sales-competitors", http.MethodPost, "/v1/sales/competitors", "Competitive intelligence"), h.Competitors)
}

// RegisterProspectRoutes registers prospect analysis endpoints
func RegisterProspectRoutes(api huma.API, h *handlers.ProspectHandler) {
	huma.Register(api, op("sales-prospect", http.MethodPost, "/v1/sales/prospect", "Full async prospect analysis"), h.Prospect)
	huma.Register(api, op("sales-prospect-quick", http.MethodPost, "/v1/sales/prospect/quick", "Quick 60-second prospect snapshot"), h.QuickProspect)
}

// RegisterJobRoutes registers job management endpoints
func RegisterJobRoutes(api huma.API, h *handlers.JobHandler) {
	huma.Register(api, op("sales-list-jobs", http.MethodGet, "/v1/sales/jobs", "List prospect jobs"), h.ListJobs)
	huma.Register(api, op("sales-get-job", http.MethodGet, "/v1/sales/jobs/{uuid}", "Get job status and results"), h.GetJob)
	huma.Register(api, op("sales-cancel-job", http.MethodDelete, "/v1/sales/jobs/{uuid}", "Cancel a running job"), h.DeleteJob)
	huma.Register(api, op("sales-retry-job", http.MethodPost, "/v1/sales/jobs/{uuid}/retry", "Retry a failed job"), h.RetryJob)
}

// RegisterReportRoutes registers report endpoints via Huma
func RegisterReportRoutes(api huma.API, h *handlers.ReportHandler) {
	huma.Register(api, op("sales-list-reports", http.MethodGet, "/v1/sales/reports", "List prospect reports"), h.ListReports)
	huma.Register(api, op("sales-get-report", http.MethodGet, "/v1/sales/reports/{uuid}", "Get report with Markdown content"), h.GetReport)
	huma.Register(api, op("sales-generate-report", http.MethodPost, "/v1/sales/reports/generate/{jobUuid}", "Generate report for a completed job"), h.GenerateReport)
}

// RegisterReportDownloadRoute registers the raw Markdown download route on chi directly
func RegisterReportDownloadRoute(r interface{ Get(string, http.HandlerFunc) }, h *handlers.ReportHandler) {
	r.Get("/v1/sales/reports/{uuid}/md", h.DownloadMarkdown)
}

// RegisterSettingsRoutes registers per-user settings endpoints
func RegisterSettingsRoutes(api huma.API, h *handlers.SettingsHandler) {
	huma.Register(api, op("sales-get-settings", http.MethodGet, "/v1/sales/settings", "Get sales intelligence settings"), h.GetSettings)
	huma.Register(api, op("sales-update-settings", http.MethodPatch, "/v1/sales/settings", "Update sales intelligence settings"), h.UpdateSettings)
}
