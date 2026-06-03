package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apperr "mmo/pkg/errors"
	"mmo/pkg/util"

	"mmo/internal/usecase"
)

type AdminHandler struct {
	uc *usecase.AdminUsecase
}

func NewAdminHandler(uc *usecase.AdminUsecase) *AdminHandler {
	return &AdminHandler{uc: uc}
}

// GET /admin/users
func (h *AdminHandler) ListUsers(c *gin.Context) {
	pg := util.ParsePagination(c)
	users, total, err := h.uc.ListUsers(c.Request.Context(), pg)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users, "total": total})
}

// PUT /admin/users/:id/role
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.New(http.StatusBadRequest, "invalid user id"))
		return
	}
	var body struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.WithDetail(http.StatusBadRequest, "validation error", err.Error()))
		return
	}

	updated, err := h.uc.UpdateRole(c.Request.Context(), mustParseUserID(c), targetID, body.Role)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": updated})
}
