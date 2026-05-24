package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"mmo/internal/usecase"
	"mmo/pkg/util"
)

type AnalyticsHandler struct {
	uc *usecase.AnalyticsUsecase
}

func NewAnalyticsHandler(uc *usecase.AnalyticsUsecase) *AnalyticsHandler {
	return &AnalyticsHandler{uc: uc}
}

func (h *AnalyticsHandler) Overview(c *gin.Context) {
	userID := mustParseUserID(c)
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 || days > 365 {
		days = 30
	}
	stats, err := h.uc.Overview(c.Request.Context(), userID, days)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats, "days": days})
}

func (h *AnalyticsHandler) ListPosts(c *gin.Context) {
	userID := mustParseUserID(c)
	p := util.ParsePagination(c)
	rows, total, err := h.uc.ListPosts(c.Request.Context(), userID, p.Page, p.PerPage)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows, "total": total})
}

func (h *AnalyticsHandler) Timeseries(c *gin.Context) {
	userID := mustParseUserID(c)
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 || days > 365 {
		days = 30
	}
	rows, err := h.uc.Timeseries(c.Request.Context(), userID, days)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows, "days": days})
}
