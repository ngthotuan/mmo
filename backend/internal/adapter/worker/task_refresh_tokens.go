package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/channel"
	"mmo/internal/integration/tiktok"
	"mmo/internal/integration/youtubepublish"
	"mmo/pkg/crypto"
	"mmo/pkg/logger"
	"go.uber.org/zap"
)

type RefreshTokensHandler struct {
	repo      *repository.ChannelRepoWithAll
	tiktok    *tiktok.Client
	youtube   *youtubepublish.Client
	cryptoKey []byte
}

func NewRefreshTokensHandler(
	repo *repository.ChannelRepoWithAll,
	tiktokClient *tiktok.Client,
	ytClient *youtubepublish.Client,
	encryptionKey string,
) *RefreshTokensHandler {
	return &RefreshTokensHandler{
		repo:      repo,
		tiktok:    tiktokClient,
		youtube:   ytClient,
		cryptoKey: []byte(encryptionKey),
	}
}

func (h *RefreshTokensHandler) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	channels, err := h.repo.ListAllActive(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	refreshed := 0

	for _, ch := range channels {
		// Only refresh if token expires within 24 hours
		if ch.TokenExpiresAt == nil || ch.TokenExpiresAt.After(now.Add(24*time.Hour)) {
			continue
		}

		switch ch.Platform {
		case channel.PlatformTikTok:
			if err := h.refreshTikTok(ctx, ch); err != nil {
				logger.Error("failed to refresh tiktok token",
					zap.String("channel_id", ch.ID.String()), zap.Error(err))
			} else {
				refreshed++
			}
		case channel.PlatformYouTube:
			if err := h.refreshYouTube(ctx, ch); err != nil {
				logger.Error("failed to refresh youtube token",
					zap.String("channel_id", ch.ID.String()), zap.Error(err))
			} else {
				refreshed++
			}
		case channel.PlatformFacebook:
			// Facebook long-lived page tokens cannot be refreshed via API —
			// mark as inactive so user reconnects via OAuth.
			if ch.TokenExpiresAt.Before(now) {
				_ = h.repo.SetActive(ctx, ch.ID, false)
				logger.Warn("facebook token expired, channel deactivated",
					zap.String("channel_id", ch.ID.String()))
			}
		}
	}

	logger.Info("token refresh complete", zap.Int("refreshed", refreshed))
	return nil
}

func (h *RefreshTokensHandler) refreshTikTok(ctx context.Context, ch *channel.Channel) error {
	encRefresh := ch.RefreshToken
	if encRefresh == "" {
		return nil
	}
	plainRefresh, err := crypto.Decrypt(h.cryptoKey, encRefresh)
	if err != nil {
		return err
	}

	tokens, err := h.tiktok.RefreshToken(ctx, plainRefresh)
	if err != nil {
		return err
	}

	encAccess, err := crypto.Encrypt(h.cryptoKey, tokens.AccessToken)
	if err != nil {
		return err
	}
	encNewRefresh, err := crypto.Encrypt(h.cryptoKey, tokens.RefreshToken)
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	ch.AccessToken = encAccess
	ch.RefreshToken = encNewRefresh
	ch.TokenExpiresAt = &expiresAt

	// Preserve existing metadata
	var meta map[string]interface{}
	_ = json.Unmarshal(ch.Metadata, &meta)

	return h.repo.Update(ctx, ch)
}

// refreshYouTube refreshes the access token. Google does NOT issue a new refresh
// token, so the existing one is preserved.
func (h *RefreshTokensHandler) refreshYouTube(ctx context.Context, ch *channel.Channel) error {
	if ch.RefreshToken == "" {
		return nil
	}
	plainRefresh, err := crypto.Decrypt(h.cryptoKey, ch.RefreshToken)
	if err != nil {
		return err
	}
	tokens, err := h.youtube.RefreshToken(ctx, plainRefresh)
	if err != nil {
		return err
	}
	encAccess, err := crypto.Encrypt(h.cryptoKey, tokens.AccessToken)
	if err != nil {
		return err
	}
	ch.AccessToken = encAccess
	if tokens.RefreshToken != "" { // Google usually omits it; keep the old one otherwise.
		if encNew, err := crypto.Encrypt(h.cryptoKey, tokens.RefreshToken); err == nil {
			ch.RefreshToken = encNew
		}
	}
	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	ch.TokenExpiresAt = &expiresAt
	return h.repo.Update(ctx, ch)
}
