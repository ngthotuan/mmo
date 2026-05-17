package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"mmo/internal/usecase"
	apperr "mmo/pkg/errors"
	"mmo/pkg/util"
)

type PublishHandler struct {
	uc *usecase.PublishUsecase
}

func NewPublishHandler(uc *usecase.PublishUsecase) *PublishHandler {
	return &PublishHandler{uc: uc}
}

func (h *PublishHandler) List(c *gin.Context) {
	userID := mustParseUserID(c)
	status := c.Query("status")
	p := util.ParsePagination(c)

	jobs, total, err := h.uc.List(c.Request.Context(), userID, status, p.Page, p.PerPage)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": jobs, "total": total})
}

func (h *PublishHandler) Create(c *gin.Context) {
	userID := mustParseUserID(c)
	var body struct {
		VideoJobID  string     `json:"video_job_id" binding:"required"`
		ChannelID   string     `json:"channel_id" binding:"required"`
		Caption     string     `json:"caption"`
		Hashtags    []string   `json:"hashtags"`
		ScheduledAt *time.Time `json:"scheduled_at"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}

	videoJobID, err := uuid.Parse(body.VideoJobID)
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	channelID, err := uuid.Parse(body.ChannelID)
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}

	job, err := h.uc.Create(c.Request.Context(), userID, usecase.CreatePublishRequest{
		VideoJobID:  videoJobID,
		ChannelID:   channelID,
		Caption:     body.Caption,
		Hashtags:    body.Hashtags,
		ScheduledAt: body.ScheduledAt,
	})
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": job})
}

func (h *PublishHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	job, err := h.uc.GetByID(c.Request.Context(), id)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": job})
}

func (h *PublishHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	var body struct {
		Caption     string     `json:"caption"`
		Hashtags    []string   `json:"hashtags"`
		ScheduledAt *time.Time `json:"scheduled_at"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	job, err := h.uc.Update(c.Request.Context(), id, body.Caption, body.Hashtags, body.ScheduledAt)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": job})
}

func (h *PublishHandler) Cancel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	if err := h.uc.Cancel(c.Request.Context(), id); err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "cancelled"})
}

func (h *PublishHandler) PublishNow(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	if err := h.uc.PublishNow(c.Request.Context(), id); err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "publish queued"})
}

func (h *PublishHandler) Calendar(c *gin.Context) {
	userID := mustParseUserID(c)
	startStr := c.Query("start")
	endStr := c.Query("end")

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid start date, use RFC3339"})
		return
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid end date, use RFC3339"})
		return
	}

	jobs, err := h.uc.ListByDateRange(c.Request.Context(), userID, start, end)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": jobs})
}
