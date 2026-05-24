package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"mmo/internal/domain/video"
	"mmo/internal/usecase"
	apperr "mmo/pkg/errors"
	"mmo/pkg/util"
)

type VideoHandler struct {
	uc *usecase.VideoUsecase
}

func NewVideoHandler(uc *usecase.VideoUsecase) *VideoHandler {
	return &VideoHandler{uc: uc}
}

type videoJobDTO struct {
	ID              uuid.UUID  `json:"id"`
	ContentPlanID   uuid.UUID  `json:"content_plan_id"`
	Status          string     `json:"status"`
	OutputVideoURL  string     `json:"output_video_url"`
	DurationSeconds float64    `json:"duration_seconds"`
	FileSizeBytes   int64      `json:"file_size_bytes"`
	RetryCount      int        `json:"retry_count"`
	ErrorMessage    string     `json:"error_message"`
	CreatedAt       string     `json:"created_at"`
	UpdatedAt       string     `json:"updated_at"`
}

func toVideoJobDTO(j *video.Job) videoJobDTO {
	return videoJobDTO{
		ID:              j.ID,
		ContentPlanID:   j.ContentPlanID,
		Status:          string(j.Status),
		OutputVideoURL:  j.OutputVideoURL,
		DurationSeconds: j.DurationSeconds,
		FileSizeBytes:   j.FileSizeBytes,
		RetryCount:      j.RetryCount,
		ErrorMessage:    j.ErrorMessage,
		CreatedAt:       j.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       j.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func (h *VideoHandler) List(c *gin.Context) {
	userID := mustParseUserID(c)
	status := c.Query("status")
	p := util.ParsePagination(c)

	jobs, total, err := h.uc.List(c.Request.Context(), userID, status, p.Page, p.PerPage)
	if err != nil {
		respondErr(c, err)
		return
	}
	dtos := make([]videoJobDTO, len(jobs))
	for i, j := range jobs {
		dtos[i] = toVideoJobDTO(j)
	}
	c.JSON(http.StatusOK, gin.H{"data": dtos, "total": total})
}

func (h *VideoHandler) Get(c *gin.Context) {
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
	c.JSON(http.StatusOK, gin.H{"data": toVideoJobDTO(job)})
}

func (h *VideoHandler) GetDownloadURL(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	url, err := h.uc.GetDownloadURL(c.Request.Context(), id)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

func (h *VideoHandler) Retry(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	if err := h.uc.RetryJob(c.Request.Context(), id); err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "retry queued"})
}

func (h *VideoHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	if err := h.uc.Delete(c.Request.Context(), id); err != nil {
		respondErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type videoTemplateDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	IsDefault bool      `json:"is_default"`
}

func (h *VideoHandler) ListTemplates(c *gin.Context) {
	userID := mustParseUserID(c)
	templates, err := h.uc.ListTemplates(c.Request.Context(), userID)
	if err != nil {
		respondErr(c, err)
		return
	}
	dtos := make([]videoTemplateDTO, len(templates))
	for i, t := range templates {
		dtos[i] = videoTemplateDTO{
			ID:        t.ID,
			Name:      t.Name,
			Type:      string(t.Type),
			IsDefault: t.IsDefault,
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": dtos})
}
