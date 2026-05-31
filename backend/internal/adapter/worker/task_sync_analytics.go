package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/publish"
	"mmo/internal/integration/facebook"
	"mmo/internal/integration/tiktok"
	"mmo/internal/integration/youtubepublish"
	"mmo/pkg/crypto"
	"mmo/pkg/logger"
	"mmo/pkg/util"
	"go.uber.org/zap"
)

type AnalyticsSyncHandler struct {
	publishRepo  *repository.PublishJobRepo
	channelRepo  *repository.ChannelRepoWithAll
	analyticsRepo *repository.AnalyticsRepo
	tiktok       *tiktok.Client
	facebook     *facebook.Client
	youtube      *youtubepublish.Client
	encKey       string
}

func NewAnalyticsSyncHandler(
	publishRepo *repository.PublishJobRepo,
	channelRepo *repository.ChannelRepoWithAll,
	analyticsRepo *repository.AnalyticsRepo,
	tiktokClient *tiktok.Client,
	fbClient *facebook.Client,
	ytClient *youtubepublish.Client,
	encKey string,
) *AnalyticsSyncHandler {
	return &AnalyticsSyncHandler{
		publishRepo:   publishRepo,
		channelRepo:   channelRepo,
		analyticsRepo: analyticsRepo,
		tiktok:        tiktokClient,
		facebook:      fbClient,
		youtube:       ytClient,
		encKey:        encKey,
	}
}

func (h *AnalyticsSyncHandler) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	// Sync analytics for all published jobs in the last 30 days
	since := time.Now().AddDate(0, 0, -30)
	jobs, _, err := h.publishRepo.List(ctx, nil, publish.JobStatusPublished, util.Pagination{Page: 1, PerPage: 1000})
	if err != nil {
		return err
	}

	synced := 0
	for _, job := range jobs {
		if job.PublishedAt != nil && job.PublishedAt.Before(since) {
			continue
		}
		if job.PlatformPostID == "" {
			continue
		}

		channel, err := h.channelRepo.GetByID(ctx, job.ChannelID)
		if err != nil {
			logger.Warn("channel not found for analytics sync", zap.String("channel_id", job.ChannelID.String()))
			continue
		}

		accessToken, err := crypto.Decrypt([]byte(h.encKey), channel.AccessToken)
		if err != nil {
			continue
		}

		var analytics publish.Analytics
		analytics.ID = uuid.New()
		analytics.PublishJobID = job.ID
		analytics.ChannelID = job.ChannelID
		analytics.Platform = job.Platform
		analytics.SyncedAt = time.Now()

		switch job.Platform {
		case "tiktok":
			stats, err := h.tiktok.GetVideoStats(ctx, accessToken, job.PlatformPostID)
			if err != nil {
				logger.Warn("tiktok stats fetch failed", zap.Error(err))
				continue
			}
			analytics.Views = stats.ViewCount
			analytics.Likes = stats.LikeCount
			analytics.Comments = stats.CommentCount
			analytics.Shares = stats.ShareCount
		case "facebook":
			stats, err := h.facebook.GetVideoStats(ctx, accessToken, job.PlatformPostID)
			if err != nil {
				logger.Warn("facebook stats fetch failed", zap.Error(err))
				continue
			}
			analytics.Views = stats.Views
			analytics.Likes = stats.Likes
			analytics.Comments = stats.Comments
			analytics.Shares = stats.Shares
			analytics.Reach = stats.Reach
		case "youtube":
			stats, err := h.youtube.GetVideoStats(ctx, accessToken, job.PlatformPostID)
			if err != nil {
				logger.Warn("youtube stats fetch failed", zap.Error(err))
				continue
			}
			analytics.Views = stats.ViewCount
			analytics.Likes = stats.LikeCount
			analytics.Comments = stats.CommentCount
		}

		raw, _ := json.Marshal(analytics)
		analytics.RawData = raw

		if err := h.analyticsRepo.Upsert(ctx, &analytics); err != nil {
			logger.Warn("analytics upsert failed", zap.Error(err))
		} else {
			synced++
		}
	}

	logger.Info("analytics sync complete", zap.Int("synced", synced))
	return nil
}
