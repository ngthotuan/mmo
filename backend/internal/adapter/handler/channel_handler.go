package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"mmo/internal/domain/channel"
	"mmo/internal/usecase"
	apperr "mmo/pkg/errors"
	"mmo/pkg/middleware"
)

type ChannelHandler struct {
	uc *usecase.ChannelUsecase
}

func NewChannelHandler(uc *usecase.ChannelUsecase) *ChannelHandler {
	return &ChannelHandler{uc: uc}
}

// GET /channels
func (h *ChannelHandler) List(c *gin.Context) {
	userID := mustParseUserID(c)

	channels, err := h.uc.List(c.Request.Context(), userID)
	if err != nil {
		respondErr(c, err)
		return
	}

	type channelDTO struct {
		ID             uuid.UUID        `json:"id"`
		Platform       channel.Platform `json:"platform"`
		PlatformUserID string           `json:"platform_user_id"`
		Username       string           `json:"username"`
		DisplayName    string           `json:"display_name"`
		AvatarURL      string           `json:"avatar_url"`
		IsActive       bool             `json:"is_active"`
		TokenExpiresAt interface{}      `json:"token_expires_at"`
	}

	dtos := make([]channelDTO, len(channels))
	for i, ch := range channels {
		dtos[i] = channelDTO{
			ID:             ch.ID,
			Platform:       ch.Platform,
			PlatformUserID: ch.PlatformUserID,
			Username:       ch.Username,
			DisplayName:    ch.DisplayName,
			AvatarURL:      ch.AvatarURL,
			IsActive:       ch.IsActive,
			TokenExpiresAt: ch.TokenExpiresAt,
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": dtos})
}

// GET /channels/connect/:platform — returns OAuth redirect URL
func (h *ChannelHandler) GetAuthURL(c *gin.Context) {
	platform := channel.Platform(c.Param("platform"))
	state := c.Query("state")
	if state == "" {
		// Use user ID as state for CSRF protection
		state = middleware.GetUserID(c)
	}

	authURL, err := h.uc.GetAuthURL(platform, state)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"auth_url": authURL})
}

// POST /channels/oauth/tiktok
func (h *ChannelHandler) ConnectTikTok(c *gin.Context) {
	userID := mustParseUserID(c)
	var body struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}

	ch, err := h.uc.ConnectTikTok(c.Request.Context(), userID, body.Code)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, channelToDTO(ch))
}

// POST /channels/oauth/facebook
func (h *ChannelHandler) ConnectFacebook(c *gin.Context) {
	userID := mustParseUserID(c)
	var body struct {
		Code   string `json:"code"    binding:"required"`
		PageID string `json:"page_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}

	ch, err := h.uc.ConnectFacebook(c.Request.Context(), userID, body.Code, body.PageID)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, channelToDTO(ch))
}

// GET /channels/facebook/pages?code=...
// Returns list of FB pages the user manages (before they pick one to connect).
func (h *ChannelHandler) ListFacebookPages(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	pages, err := h.uc.GetFacebookPages(c.Request.Context(), code)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": pages})
}

// DELETE /channels/:id
func (h *ChannelHandler) Delete(c *gin.Context) {
	userID := mustParseUserID(c)
	channelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	if err := h.uc.Delete(c.Request.Context(), userID, channelID); err != nil {
		respondErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// PUT /channels/:id/toggle
func (h *ChannelHandler) Toggle(c *gin.Context) {
	userID := mustParseUserID(c)
	channelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	var body struct {
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	if err := h.uc.SetActive(c.Request.Context(), userID, channelID, body.Active); err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"active": body.Active})
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func mustParseUserID(c *gin.Context) uuid.UUID {
	id, _ := uuid.Parse(middleware.GetUserID(c))
	return id
}

func respondErr(c *gin.Context, err error) {
	if appErr, ok := apperr.As(err); ok {
		c.JSON(appErr.Code, appErr)
		return
	}
	c.JSON(http.StatusInternalServerError, apperr.ErrInternalServer)
}

func channelToDTO(ch *channel.Channel) gin.H {
	var meta map[string]interface{}
	_ = json.Unmarshal(ch.Metadata, &meta)
	return gin.H{
		"id":               ch.ID,
		"platform":         ch.Platform,
		"platform_user_id": ch.PlatformUserID,
		"username":         ch.Username,
		"display_name":     ch.DisplayName,
		"avatar_url":       ch.AvatarURL,
		"is_active":        ch.IsActive,
		"token_expires_at": ch.TokenExpiresAt,
	}
}
