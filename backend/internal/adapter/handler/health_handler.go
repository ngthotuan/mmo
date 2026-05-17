package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type HealthHandler struct {
	db *sqlx.DB
}

func NewHealthHandler(db *sqlx.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Check(c *gin.Context) {
	status := "ok"
	dbStatus := "ok"

	if err := h.db.PingContext(c.Request.Context()); err != nil {
		status = "degraded"
		dbStatus = "error"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    status,
		"db":        dbStatus,
		"timestamp": time.Now().UTC(),
	})
}
