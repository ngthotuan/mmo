package worker

import (
	"context"
	"encoding/json"
	"time"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/content"
	"mmo/internal/domain/publish"
	"mmo/internal/integration/facebook"
	"mmo/internal/integration/tiktok"
	"mmo/internal/integration/youtubepublish"
	"mmo/pkg/crypto"
	"mmo/pkg/logger"
	"go.uber.org/zap"
)

type PublishHandler struct {
	publishRepo *repository.PublishJobRepo
	channelRepo *repository.ChannelRepoWithAll
	videoRepo   *repository.VideoJobRepo
	planRepo    *repository.ContentPlanRepo
	productRepo *repository.ProductRepo
	tiktok      *tiktok.Client
	facebook    *facebook.Client
	youtube     *youtubepublish.Client
	encKey      string
	dryRun      bool
	maxRetry    int
}

func NewPublishHandler(
	publishRepo *repository.PublishJobRepo,
	channelRepo *repository.ChannelRepoWithAll,
	videoRepo *repository.VideoJobRepo,
	planRepo *repository.ContentPlanRepo,
	productRepo *repository.ProductRepo,
	tiktokClient *tiktok.Client,
	fbClient *facebook.Client,
	ytClient *youtubepublish.Client,
	encKey string,
	dryRun bool,
	maxRetry int,
) *PublishHandler {
	if maxRetry <= 0 {
		maxRetry = 5
	}
	return &PublishHandler{
		publishRepo: publishRepo,
		channelRepo: channelRepo,
		videoRepo:   videoRepo,
		planRepo:    planRepo,
		productRepo: productRepo,
		tiktok:      tiktokClient,
		facebook:    fbClient,
		youtube:     ytClient,
		encKey:      encKey,
		dryRun:      dryRun,
		maxRetry:    maxRetry,
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

	// Atomic claim (scheduled → publishing). Guards against double-publish when
	// asynq delivers the same task more than once.
	claimed, err := h.publishRepo.Claim(ctx, jobID)
	if err != nil {
		return fmt.Errorf("claim publish job: %w", err)
	}
	if !claimed {
		logger.Warn("publish job not claimable (already taken/processed)",
			zap.String("id", jobID.String()), zap.String("status", string(job.Status)))
		return nil
	}

	videoJob, err := h.videoRepo.GetByID(ctx, job.VideoJobID)
	if err != nil {
		return h.fail(ctx, job, fmt.Errorf("video job not found: %w", err))
	}
	if videoJob.OutputVideoURL == "" {
		return h.fail(ctx, job, fmt.Errorf("video not ready: no output URL"))
	}

	channel, err := h.channelRepo.GetByID(ctx, job.ChannelID)
	if err != nil {
		return h.fail(ctx, job, fmt.Errorf("channel not found: %w", err))
	}

	caption := buildCaption(job.Caption, job.Hashtags)

	var postID, postURL string
	dryRun := h.dryRun || channel.DryRun
	if dryRun {
		postID = "dryrun_" + job.Platform + "_" + uuid.NewString()
		postURL = "https://dry-run.local/" + job.Platform + "/" + postID
		logger.Info("DRY-RUN publish (no real API call)",
			zap.String("publish_job_id", jobID.String()), zap.String("platform", job.Platform))
	} else {
		postID, postURL, err = h.publishReal(ctx, job, videoJob.OutputVideoURL, channel.PlatformUserID, channel.AccessToken, caption)
		if err != nil {
			return h.fail(ctx, job, fmt.Errorf("publish failed: %w", err))
		}
	}

	now := time.Now()
	job.PlatformPostID = postID
	job.PlatformPostURL = postURL
	job.Status = publish.JobStatusPublished
	job.PublishedAt = &now
	if err := h.publishRepo.Update(ctx, job); err != nil {
		return err
	}
	h.maybeFinishPlan(ctx, job)

	logger.Info("published successfully",
		zap.String("publish_job_id", jobID.String()),
		zap.String("platform_post_id", postID),
	)
	return nil
}

// publishReal performs the actual platform API call. Token decryption only
// happens here, so dry-run channels need no valid encrypted token.
func (h *PublishHandler) publishReal(ctx context.Context, job *publish.Job, videoURL, pageID, encToken, caption string) (string, string, error) {
	accessToken, err := crypto.Decrypt([]byte(h.encKey), encToken)
	if err != nil {
		return "", "", fmt.Errorf("decrypt token: %w", err)
	}

	products, _ := h.productRepo.ListByPublishJob(ctx, job.ID)

	switch job.Platform {
	case "tiktok":
		productLinks := make([]string, 0, len(products))
		for _, p := range products {
			productLinks = append(productLinks, p.PlatformProductID)
		}
		postID, err := h.tiktok.PostVideo(ctx, accessToken, tiktok.PostVideoRequest{
			VideoURL:     videoURL,
			Caption:      caption,
			ProductLinks: productLinks,
		})
		return postID, "", err
	case "facebook":
		productURLs := make([]string, 0, len(products))
		for _, p := range products {
			if p.ProductURL != "" {
				productURLs = append(productURLs, p.ProductURL)
			}
		}
		postID, err := h.facebook.PostVideo(ctx, accessToken, facebook.PostVideoRequest{
			PageID:      pageID,
			VideoURL:    videoURL,
			Description: caption,
			ProductURLs: productURLs,
		})
		return postID, "", err
	case "youtube":
		postID, err := h.youtube.PostVideo(ctx, accessToken, youtubepublish.PostVideoRequest{
			VideoURL:    videoURL,
			Title:       job.Caption,
			Description: caption,
			Tags:        stripHashes(job.Hashtags),
		})
		if err != nil {
			return "", "", err
		}
		return postID, youtubepublish.WatchURL(postID), nil
	default:
		return "", "", fmt.Errorf("unsupported platform: %s", job.Platform)
	}
}

// stripHashes removes a leading '#' from each hashtag (YouTube tags must not include it).
func stripHashes(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		out = append(out, strings.TrimPrefix(t, "#"))
	}
	return out
}

// fail records the failure with an incremented retry count and an exponential
// backoff next_retry_at (nil once max attempts is reached). The auto-retry cron
// picks up jobs whose next_retry_at has passed.
func (h *PublishHandler) fail(ctx context.Context, job *publish.Job, cause error) error {
	job.RetryCount++
	var nextRetry *time.Time
	if job.RetryCount < h.maxRetry {
		shift := job.RetryCount
		if shift > 6 {
			shift = 6
		}
		t := time.Now().Add(time.Duration(1<<shift) * time.Minute)
		nextRetry = &t
	}
	if err := h.publishRepo.MarkFailed(ctx, job.ID, cause.Error(), job.RetryCount, nextRetry); err != nil {
		logger.Error("mark publish failed", zap.Error(err))
	}
	return cause
}

// maybeFinishPlan flips the content plan to "published" once every publish job
// for that plan has reached a terminal state.
func (h *PublishHandler) maybeFinishPlan(ctx context.Context, job *publish.Job) {
	if job.ContentPlanID == nil {
		return
	}
	n, err := h.publishRepo.CountUnfinishedForPlan(ctx, *job.ContentPlanID)
	if err != nil || n > 0 {
		return
	}
	if err := h.planRepo.UpdateStatus(ctx, *job.ContentPlanID, content.StatusPublished); err != nil {
		logger.Warn("flip plan to published failed", zap.Error(err))
	}
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
