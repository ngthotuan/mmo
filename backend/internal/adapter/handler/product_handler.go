package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"mmo/internal/usecase"
	apperr "mmo/pkg/errors"
	"mmo/pkg/util"
)

type ProductHandler struct {
	uc *usecase.ProductUsecase
}

func NewProductHandler(uc *usecase.ProductUsecase) *ProductHandler {
	return &ProductHandler{uc: uc}
}

func (h *ProductHandler) List(c *gin.Context) {
	userID := mustParseUserID(c)
	platform := c.Query("platform")
	p := util.ParsePagination(c)

	products, total, err := h.uc.List(c.Request.Context(), userID, platform, p.Page, p.PerPage)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": products, "total": total})
}

func (h *ProductHandler) Create(c *gin.Context) {
	userID := mustParseUserID(c)
	var body usecase.CreateProductRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	p, err := h.uc.Create(c.Request.Context(), userID, body)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": p})
}

func (h *ProductHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	p, err := h.uc.GetByID(c.Request.Context(), id)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": p})
}

func (h *ProductHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	if err := h.uc.Delete(c.Request.Context(), id); err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// Sync fetches products from the given platform channel and upserts them locally.
func (h *ProductHandler) Sync(c *gin.Context) {
	userID := mustParseUserID(c)
	var body struct {
		Platform   string `json:"platform"    binding:"required"`
		ChannelID  string `json:"channel_id"  binding:"required"`
		CatalogID  string `json:"catalog_id"` // Facebook only
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	channelID, err := uuid.Parse(body.ChannelID)
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}

	var count int
	switch body.Platform {
	case "tiktok":
		count, err = h.uc.SyncFromTikTok(c.Request.Context(), userID, channelID)
	case "facebook":
		count, err = h.uc.SyncFromFacebook(c.Request.Context(), userID, channelID, body.CatalogID)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"message": "unsupported platform"})
		return
	}
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"synced": count})
}

// ListByPublishJob returns products tagged on a publish job.
func (h *ProductHandler) ListByPublishJob(c *gin.Context) {
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	products, err := h.uc.ListByPublishJob(c.Request.Context(), jobID)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": products})
}

// TagPublishJob sets product tags on a publish job.
func (h *ProductHandler) TagPublishJob(c *gin.Context) {
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	var body struct {
		ProductIDs []string `json:"product_ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	productIDs := make([]uuid.UUID, 0, len(body.ProductIDs))
	for _, s := range body.ProductIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
			return
		}
		productIDs = append(productIDs, id)
	}
	if err := h.uc.TagPublishJob(c.Request.Context(), jobID, productIDs); err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "products tagged"})
}
