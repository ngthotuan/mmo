package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/channel"
	"mmo/internal/integration/tiktok"
	"mmo/pkg/crypto"
	"mmo/pkg/logger"
	"go.uber.org/zap"
)

type RefreshTokensHandler struct {
	repo      *repository.ChannelRepoWithAll
	tiktok    *tiktok.Client
	cryptoKey []byte
}

func NewRefreshTokensHandler(
	repo *repository.ChannelRepoWithAll,
	tiktokClient *tiktok.Client,
	encryptionKey string,
) *RefreshTokensHandler {
	return &RefreshTokensHandler{
		repo:      repo,
		tiktok:    tiktokClient,
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
