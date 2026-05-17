package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/publish"
	"mmo/internal/integration/facebook"
	"mmo/internal/integration/tiktok"
	"mmo/pkg/crypto"
	"mmo/pkg/logger"
	"go.uber.org/zap"
)

type PublishHandler struct {
	publishRepo *repository.PublishJobRepo
	channelRepo *repository.ChannelRepoWithAll
	videoRepo   *repository.VideoJobRepo
	productRepo *repository.ProductRepo
	tiktok      *tiktok.Client
	facebook    *facebook.Client
	encKey      string
}

func NewPublishHandler(
	publishRepo *repository.PublishJobRepo,
	channelRepo *repository.ChannelRepoWithAll,
	videoRepo *repository.VideoJobRepo,
	productRepo *repository.ProductRepo,
	tiktokClient *tiktok.Client,
	fbClient *facebook.Client,
	encKey string,
) *PublishHandler {
	return &PublishHandler{
		publishRepo: publishRepo,
		channelRepo: channelRepo,
		videoRepo:   videoRepo,
		productRepo: productRepo,
		tiktok:      tiktokClient,
		facebook:    fbClient,
		encKey:      encKey,
	}
}

func (h *PublishHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var p struct {
		PublishJobID string `json:"publish_job_id"`
	}
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		return err
	}

	jobID, err := uuid.Parse(p.PublishJobID)
	if err != nil {
		return err
	}

	job, err := h.publishRepo.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("publish job not found: %w", err)
	}
	if job.Status != publish.JobStatusScheduled {
		logger.Warn("publish job already processed", zap.String("id", jobID.String()), zap.String("status", string(job.Status)))
		return nil
	}

	videoJob, err := h.videoRepo.GetByID(ctx, job.VideoJobID)
	if err != nil {
		return fmt.Errorf("video job not found: %w", err)
	}
	if videoJob.OutputVideoURL == "" {
		_ = h.publishRepo.UpdateStatus(ctx, jobID, publish.JobStatusFailed, "video not ready: no output URL")
		return fmt.Errorf("video not ready")
	}

	channel, err := h.channelRepo.GetByID(ctx, job.ChannelID)
	if err != nil {
		return fmt.Errorf("channel not found: %w", err)
	}

	// Decrypt access token
	accessToken, err := crypto.Decrypt([]byte(h.encKey), channel.AccessToken)
	if err != nil {
		return fmt.Errorf("decrypt token: %w", err)
	}

	_ = h.publishRepo.UpdateStatus(ctx, jobID, publish.JobStatusPublishing, "")

	logger.Info("publishing video",
		zap.String("publish_job_id", jobID.String()),
		zap.String("platform", job.Platform),
	)

	// Load tagged products for this publish job
	products, _ := h.productRepo.ListByPublishJob(ctx, jobID)

	var postID string
	caption := buildCaption(job.Caption, job.Hashtags)

	switch job.Platform {
	case "tiktok":
		productLinks := make([]string, 0, len(products))
		for _, p := range products {
			productLinks = append(productLinks, p.PlatformProductID)
		}
		postID, err = h.tiktok.PostVideo(ctx, accessToken, tiktok.PostVideoRequest{
			VideoURL:     videoJob.OutputVideoURL,
			Caption:      caption,
			ProductLinks: productLinks,
		})
	case "facebook":
		productURLs := make([]string, 0, len(products))
		for _, p := range products {
			if p.ProductURL != "" {
				productURLs = append(productURLs, p.ProductURL)
			}
		}
		postID, err = h.facebook.PostVideo(ctx, accessToken, facebook.PostVideoRequest{
			PageID:      channel.PlatformUserID,
			VideoURL:    videoJob.OutputVideoURL,
			Description: caption,
			ProductURLs: productURLs,
		})
	default:
		err = fmt.Errorf("unsupported platform: %s", job.Platform)
	}

	if err != nil {
		job.RetryCount++
		_ = h.publishRepo.UpdateStatus(ctx, jobID, publish.JobStatusFailed, err.Error())
		return fmt.Errorf("publish failed: %w", err)
	}

	job.PlatformPostID = postID
	job.Status = publish.JobStatusPublished
	if err := h.publishRepo.Update(ctx, job); err != nil {
		return err
	}

	logger.Info("published successfully",
		zap.String("publish_job_id", jobID.String()),
		zap.String("platform_post_id", postID),
	)
	return nil
}

func buildCaption(caption string, hashtags []string) string {
	if len(hashtags) == 0 {
		return caption
	}
	tags := make([]string, len(hashtags))
	for i, h := range hashtags {
		if !strings.HasPrefix(h, "#") {
			tags[i] = "#" + h
		} else {
			tags[i] = h
		}
	}
	return caption + "\n\n" + strings.Join(tags, " ")
}
