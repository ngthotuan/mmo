package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

func (h *VideoHandler) List(c *gin.Context) {
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
	c.JSON(http.StatusOK, gin.H{"data": job})
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

func (h *VideoHandler) ListTemplates(c *gin.Context) {
	userID := mustParseUserID(c)
	templates, err := h.uc.ListTemplates(c.Request.Context(), userID)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": templates})
}
