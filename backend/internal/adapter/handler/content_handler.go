package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"mmo/internal/domain/content"
	"mmo/internal/usecase"
	apperr "mmo/pkg/errors"
	"mmo/pkg/util"
)

type ContentHandler struct {
	uc *usecase.ContentUsecase
}

func NewContentHandler(uc *usecase.ContentUsecase) *ContentHandler {
	return &ContentHandler{uc: uc}
}

type planDTO struct {
	ID              uuid.UUID       `json:"id"`
	Title           string          `json:"title"`
	Niche           string          `json:"niche"`
	TargetPlatforms []string        `json:"target_platforms"`
	Script          string          `json:"script"`
	ScriptMetadata  json.RawMessage `json:"script_metadata"`
	Status          string          `json:"status"`
	AutoApprove     bool            `json:"auto_approve"`
	Voice           string          `json:"voice"`
	Notes           string          `json:"notes"`
	CreatedAt       string          `json:"created_at"`
}

func toPlanDTO(p *content.ContentPlan) planDTO {
	platforms := p.TargetPlatforms
	if platforms == nil {
		platforms = []string{}
	}
	return planDTO{
		ID:              p.ID,
		Title:           p.Title,
		Niche:           p.Niche,
		TargetPlatforms: platforms,
		Script:          p.Script,
		ScriptMetadata:  p.ScriptMetadata,
		Status:          string(p.Status),
		AutoApprove:     p.AutoApprove,
		Voice:           p.Voice,
		Notes:           p.Notes,
		CreatedAt:       p.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// GET /trends
func (h *ContentHandler) ListTrends(c *gin.Context) {
	userID := mustParseUserID(c)
	p := util.ParsePagination(c)
	status := c.Query("status")

	trends, total, err := h.uc.ListTrends(c.Request.Context(), userID, status, p.Page, p.PerPage)
	if err != nil {
		respondErr(c, err)
		return
	}

	type trendDTO struct {
		ID            uuid.UUID `json:"id"`
		Source        string    `json:"source"`
		Title         string    `json:"title"`
		Description   string    `json:"description"`
		Keywords      []string  `json:"keywords"`
		TrendingScore float64   `json:"trending_score"`
		SourceURL     string    `json:"source_url"`
		Status        string    `json:"status"`
		DiscoveredAt  string    `json:"discovered_at"`
	}

	dtos := make([]trendDTO, len(trends))
	for i, t := range trends {
		dtos[i] = trendDTO{
			ID:            t.ID,
			Source:        t.Source,
			Title:         t.Title,
			Description:   t.Description,
			Keywords:      t.Keywords,
			TrendingScore: t.TrendingScore,
			SourceURL:     t.SourceURL,
			Status:        t.Status,
			DiscoveredAt:  t.DiscoveredAt.Format("2006-01-02T15:04:05Z"),
		}
		if dtos[i].Keywords == nil {
			dtos[i].Keywords = []string{}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       dtos,
		"pagination": gin.H{"total": total, "page": p.Page, "per_page": p.PerPage},
	})
}

// POST /trends/discover
func (h *ContentHandler) TriggerDiscover(c *gin.Context) {
	userID := mustParseUserID(c)
	if err := h.uc.TriggerDiscover(c.Request.Context(), userID); err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "trend discovery queued"})
}

// GET /content
func (h *ContentHandler) ListPlans(c *gin.Context) {
	userID := mustParseUserID(c)
	p := util.ParsePagination(c)
	status := c.Query("status")

	plans, total, err := h.uc.ListPlans(c.Request.Context(), userID, status, p.Page, p.PerPage)
	if err != nil {
		respondErr(c, err)
		return
	}

	dtos := make([]planDTO, len(plans))
	for i, plan := range plans {
		dtos[i] = toPlanDTO(plan)
	}
	c.JSON(http.StatusOK, gin.H{
		"data":       dtos,
		"pagination": gin.H{"total": total, "page": p.Page, "per_page": p.PerPage},
	})
}

// GET /content/:id
func (h *ContentHandler) GetPlan(c *gin.Context) {
	userID := mustParseUserID(c)
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	plan, err := h.uc.GetPlan(c.Request.Context(), userID, planID)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, toPlanDTO(plan))
}

// PUT /content/:id
func (h *ContentHandler) UpdatePlan(c *gin.Context) {
	userID := mustParseUserID(c)
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	var body struct {
		Title           string   `json:"title"`
		Niche           string   `json:"niche"`
		Script          string   `json:"script"`
		Notes           string   `json:"notes"`
		Voice           string   `json:"voice"`
		TargetPlatforms []string `json:"target_platforms"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	plan, err := h.uc.UpdatePlan(c.Request.Context(), userID, planID,
		body.Title, body.Niche, body.Script, body.Notes, body.Voice, body.TargetPlatforms)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, toPlanDTO(plan))
}

// POST /content/:id/approve
func (h *ContentHandler) ApprovePlan(c *gin.Context) {
	userID := mustParseUserID(c)
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	if err := h.uc.ApprovePlan(c.Request.Context(), userID, planID); err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "approved"})
}

// POST /content/:id/reject
func (h *ContentHandler) RejectPlan(c *gin.Context) {
	userID := mustParseUserID(c)
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	if err := h.uc.RejectPlan(c.Request.Context(), userID, planID); err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "rejected"})
}

// POST /content/:id/generate-script
func (h *ContentHandler) RegenerateScript(c *gin.Context) {
	userID := mustParseUserID(c)
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	plan, err := h.uc.RegenerateScript(c.Request.Context(), userID, planID)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, toPlanDTO(plan))
}

// POST /content/from-trend
func (h *ContentHandler) CreateFromTrend(c *gin.Context) {
	userID := mustParseUserID(c)
	var body struct {
		TopicID     string   `json:"topic_id"     binding:"required"`
		Niche       string   `json:"niche"`
		Platforms   []string `json:"platforms"`
		AutoApprove bool     `json:"auto_approve"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	topicID, err := uuid.Parse(body.TopicID)
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	if len(body.Platforms) == 0 {
		body.Platforms = []string{"tiktok"}
	}
	plan, err := h.uc.GenerateScriptForTrend(c.Request.Context(), userID, topicID,
		body.Niche, body.Platforms, body.AutoApprove)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, toPlanDTO(plan))
}

// POST /content/bulk-action
func (h *ContentHandler) BulkActionPlans(c *gin.Context) {
	userID := mustParseUserID(c)
	var body struct {
		Action string   `json:"action" binding:"required"`
		IDs    []string `json:"ids"    binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	planIDs := make([]uuid.UUID, 0, len(body.IDs))
	for _, raw := range body.IDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			continue
		}
		planIDs = append(planIDs, id)
	}
	done, err := h.uc.BulkActionPlans(c.Request.Context(), userID, body.Action, planIDs)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"processed": done})
}

// POST /trends/bulk-reject
func (h *ContentHandler) BulkRejectTrends(c *gin.Context) {
	userID := mustParseUserID(c)
	var body struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	topicIDs := make([]uuid.UUID, 0, len(body.IDs))
	for _, raw := range body.IDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			continue
		}
		topicIDs = append(topicIDs, id)
	}
	done, err := h.uc.BulkRejectTrends(c.Request.Context(), userID, topicIDs)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"processed": done})
}

// DELETE /content/:id
func (h *ContentHandler) DeletePlan(c *gin.Context) {
	userID := mustParseUserID(c)
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	if err := h.uc.DeletePlan(c.Request.Context(), userID, planID); err != nil {
		respondErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

