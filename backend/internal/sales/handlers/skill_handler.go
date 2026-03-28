package handlers

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/orkestra/backend/internal/sales/models"
	"github.com/orkestra/backend/internal/sales/services"
)

// SkillHandler handles HTTP requests for individual sales intelligence skills
type SkillHandler struct {
	orchestrator services.OrchestratorService
}

// NewSkillHandler creates a new SkillHandler
func NewSkillHandler(orchestrator services.OrchestratorService) *SkillHandler {
	return &SkillHandler{orchestrator: orchestrator}
}

func (h *SkillHandler) executeSkill(ctx context.Context, skillName models.SkillName, req *models.SkillRequest) (*models.SkillResponse, error) {
	userUUID, _ := ctx.Value("userUUID").(string)

	result, err := h.orchestrator.RunSkill(ctx, skillName, req.Body.URL, req.Body.Locale, req.Body.Context, userUUID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Skill execution failed", err)
	}

	resp := &models.SkillResponse{}
	resp.Body.Skill = result.Skill
	resp.Body.Result = result.Result
	resp.Body.InputTokens = result.InputTokens
	resp.Body.OutputTokens = result.OutputTokens
	resp.Body.LatencyMs = result.LatencyMs
	resp.Body.ModelUsed = result.ModelUsed
	return resp, nil
}

// Research handles POST /v1/sales/research
func (h *SkillHandler) Research(ctx context.Context, req *models.SkillRequest) (*models.SkillResponse, error) {
	return h.executeSkill(ctx, models.SkillResearch, req)
}

// Qualify handles POST /v1/sales/qualify
func (h *SkillHandler) Qualify(ctx context.Context, req *models.SkillRequest) (*models.SkillResponse, error) {
	return h.executeSkill(ctx, models.SkillQualify, req)
}

// Contacts handles POST /v1/sales/contacts
func (h *SkillHandler) Contacts(ctx context.Context, req *models.SkillRequest) (*models.SkillResponse, error) {
	return h.executeSkill(ctx, models.SkillContacts, req)
}

// Outreach handles POST /v1/sales/outreach
func (h *SkillHandler) Outreach(ctx context.Context, req *models.SkillRequest) (*models.SkillResponse, error) {
	return h.executeSkill(ctx, models.SkillOutreach, req)
}

// Followup handles POST /v1/sales/followup
func (h *SkillHandler) Followup(ctx context.Context, req *models.SkillRequest) (*models.SkillResponse, error) {
	return h.executeSkill(ctx, models.SkillFollowup, req)
}

// Prep handles POST /v1/sales/prep
func (h *SkillHandler) Prep(ctx context.Context, req *models.SkillRequest) (*models.SkillResponse, error) {
	return h.executeSkill(ctx, models.SkillPrep, req)
}

// Proposal handles POST /v1/sales/proposal
func (h *SkillHandler) Proposal(ctx context.Context, req *models.SkillRequest) (*models.SkillResponse, error) {
	return h.executeSkill(ctx, models.SkillProposal, req)
}

// Objections handles POST /v1/sales/objections
func (h *SkillHandler) Objections(ctx context.Context, req *models.SkillRequest) (*models.SkillResponse, error) {
	return h.executeSkill(ctx, models.SkillObjections, req)
}

// ICP handles POST /v1/sales/icp
func (h *SkillHandler) ICP(ctx context.Context, req *models.SkillRequest) (*models.SkillResponse, error) {
	return h.executeSkill(ctx, models.SkillICP, req)
}

// Competitors handles POST /v1/sales/competitors
func (h *SkillHandler) Competitors(ctx context.Context, req *models.SkillRequest) (*models.SkillResponse, error) {
	return h.executeSkill(ctx, models.SkillCompetitors, req)
}
