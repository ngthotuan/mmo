package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"mmo/internal/domain/autopilot"
	"mmo/internal/usecase"
	apperr "mmo/pkg/errors"
)

type AutoPilotHandler struct {
	uc *usecase.AutoPilotUsecase
}

func NewAutoPilotHandler(uc *usecase.AutoPilotUsecase) *AutoPilotHandler {
	return &AutoPilotHandler{uc: uc}
}

type autoPilotDTO struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Niche           string    `json:"niche"`
	Voice           string    `json:"voice"`
	TargetPlatforms []string  `json:"target_platforms"`
	TrendFilter     string    `json:"trend_filter"`
	TrendSources    []string  `json:"trend_sources"`
	DailyCount      int       `json:"daily_count"`
	ScheduleTimes   []string  `json:"schedule_times"`
	AutoApprove     bool      `json:"auto_approve"`
	AutoPublish     bool      `json:"auto_publish"`
	Enabled         bool      `json:"enabled"`
	LastRunAt       *string   `json:"last_run_at"`
	LastRunCount    int       `json:"last_run_count"`
	TotalVideos     int       `json:"total_videos"`
	CreatedAt       string    `json:"created_at"`
}

func toAutoPilotDTO(p *autopilot.Profile) autoPilotDTO {
	dto := autoPilotDTO{
		ID:              p.ID,
		Name:            p.Name,
		Niche:           p.Niche,
		Voice:           p.Voice,
		TargetPlatforms: p.TargetPlatforms,
		TrendFilter:     p.TrendFilter,
		TrendSources:    p.TrendSources,
		DailyCount:      p.DailyCount,
		ScheduleTimes:   p.ScheduleTimes,
		AutoApprove:     p.AutoApprove,
		AutoPublish:     p.AutoPublish,
		Enabled:         p.Enabled,
		LastRunCount:    p.LastRunCount,
		TotalVideos:     p.TotalVideos,
		CreatedAt:       p.CreatedAt.Format(time.RFC3339),
	}
	if dto.TargetPlatforms == nil {
		dto.TargetPlatforms = []string{}
	}
	if dto.TrendSources == nil {
		dto.TrendSources = []string{}
	}
	if dto.ScheduleTimes == nil {
		dto.ScheduleTimes = []string{}
	}
	if p.LastRunAt != nil {
		s := p.LastRunAt.Format(time.RFC3339)
		dto.LastRunAt = &s
	}
	return dto
}

type autoPilotBody struct {
	Name            string   `json:"name"`
	Niche           string   `json:"niche"`
	Voice           string   `json:"voice"`
	TargetPlatforms []string `json:"target_platforms"`
	TrendFilter     string   `json:"trend_filter"`
	TrendSources    []string `json:"trend_sources"`
	DailyCount      int      `json:"daily_count"`
	ScheduleTimes   []string `json:"schedule_times"`
	AutoApprove     *bool    `json:"auto_approve"`
	AutoPublish     *bool    `json:"auto_publish"`
	Enabled         *bool    `json:"enabled"`
}

func (b autoPilotBody) toEntity(userID uuid.UUID, id uuid.UUID) *autopilot.Profile {
	p := &autopilot.Profile{
		ID:              id,
		UserID:          userID,
		Name:            b.Name,
		Niche:           b.Niche,
		Voice:           b.Voice,
		TargetPlatforms: b.TargetPlatforms,
		TrendFilter:     b.TrendFilter,
		TrendSources:    b.TrendSources,
		DailyCount:      b.DailyCount,
		ScheduleTimes:   b.ScheduleTimes,
		AutoApprove:     true,
		AutoPublish:     true,
		Enabled:         true,
	}
	if b.AutoApprove != nil {
		p.AutoApprove = *b.AutoApprove
	}
	if b.AutoPublish != nil {
		p.AutoPublish = *b.AutoPublish
	}
	if b.Enabled != nil {
		p.Enabled = *b.Enabled
	}
	return p
}

// GET /auto-pilot
func (h *AutoPilotHandler) List(c *gin.Context) {
	userID := mustParseUserID(c)
	profiles, err := h.uc.List(c.Request.Context(), userID)
	if err != nil {
		respondErr(c, err)
		return
	}
	dtos := make([]autoPilotDTO, len(profiles))
	for i, p := range profiles {
		dtos[i] = toAutoPilotDTO(p)
	}
	c.JSON(http.StatusOK, gin.H{"data": dtos, "total": len(dtos)})
}

// GET /auto-pilot/:id
func (h *AutoPilotHandler) Get(c *gin.Context) {
	userID := mustParseUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	p, err := h.uc.Get(c.Request.Context(), userID, id)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": toAutoPilotDTO(p)})
}

// POST /auto-pilot
func (h *AutoPilotHandler) Create(c *gin.Context) {
	userID := mustParseUserID(c)
	var body autoPilotBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	p, err := h.uc.Create(c.Request.Context(), body.toEntity(userID, uuid.Nil))
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": toAutoPilotDTO(p)})
}

// POST /auto-pilot/quick-setup — one-click MMO channel: provisions dry-run
// channels + an enabled Vietnamese-MMO auto-pilot profile.
func (h *AutoPilotHandler) QuickSetup(c *gin.Context) {
	userID := mustParseUserID(c)
	var body struct {
		Name          string   `json:"name"`
		Niche         string   `json:"niche"`
		Voice         string   `json:"voice"`
		Platforms     []string `json:"platforms"`
		ScheduleTimes []string `json:"schedule_times"`
		DailyCount    int      `json:"daily_count"`
	}
	_ = c.ShouldBindJSON(&body) // all fields optional; defaults applied in usecase

	p, err := h.uc.QuickSetup(c.Request.Context(), userID, usecase.QuickSetupInput{
		Name:          body.Name,
		Niche:         body.Niche,
		Voice:         body.Voice,
		Platforms:     body.Platforms,
		ScheduleTimes: body.ScheduleTimes,
		DailyCount:    body.DailyCount,
	})
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": toAutoPilotDTO(p)})
}

// PUT /auto-pilot/:id
func (h *AutoPilotHandler) Update(c *gin.Context) {
	userID := mustParseUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	var body autoPilotBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	p, err := h.uc.Update(c.Request.Context(), userID, body.toEntity(userID, id))
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": toAutoPilotDTO(p)})
}

// PUT /auto-pilot/:id/toggle
func (h *AutoPilotHandler) Toggle(c *gin.Context) {
	userID := mustParseUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	if err := h.uc.Toggle(c.Request.Context(), userID, id, body.Enabled); err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "toggled"})
}

// DELETE /auto-pilot/:id
func (h *AutoPilotHandler) Delete(c *gin.Context) {
	userID := mustParseUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	if err := h.uc.Delete(c.Request.Context(), userID, id); err != nil {
		respondErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// POST /auto-pilot/:id/run — manual trigger for testing
func (h *AutoPilotHandler) RunNow(c *gin.Context) {
	userID := mustParseUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	p, err := h.uc.Get(c.Request.Context(), userID, id)
	if err != nil {
		respondErr(c, err)
		return
	}
	count, err := h.uc.RunProfile(c.Request.Context(), p)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"plans_created": count})
}
