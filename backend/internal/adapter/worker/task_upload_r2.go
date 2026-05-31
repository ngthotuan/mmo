package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/content"
	"mmo/internal/domain/publish"
	"mmo/internal/domain/video"
	"mmo/internal/infrastructure/ffmpeg"
	"mmo/internal/infrastructure/storage"
	"mmo/pkg/logger"
	"go.uber.org/zap"
)

type R2UploadHandler struct {
	videoRepo     *repository.VideoJobRepo
	planRepo      *repository.ContentPlanRepo
	r2            *storage.R2Client
	assembler     *ffmpeg.Assembler
	autoPilotRepo *repository.AutoPilotRepo
	channelRepo   *repository.ChannelRepoWithAll
	publishRepo   *repository.PublishJobRepo
}

func NewR2UploadHandler(
	videoRepo *repository.VideoJobRepo,
	planRepo *repository.ContentPlanRepo,
	r2 *storage.R2Client,
	assembler *ffmpeg.Assembler,
	autoPilotRepo *repository.AutoPilotRepo,
	channelRepo *repository.ChannelRepoWithAll,
	publishRepo *repository.PublishJobRepo,
) *R2UploadHandler {
	return &R2UploadHandler{
		videoRepo:     videoRepo,
		planRepo:      planRepo,
		r2:            r2,
		assembler:     assembler,
		autoPilotRepo: autoPilotRepo,
		channelRepo:   channelRepo,
		publishRepo:   publishRepo,
	}
}

func (h *R2UploadHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var p struct {
		JobID     string `json:"job_id"`
		VideoPath string `json:"video_path"`
	}
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		return err
	}

	jobID, err := uuid.Parse(p.JobID)
	if err != nil {
		return err
	}

	job, err := h.videoRepo.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("video job not found: %w", err)
	}

	logger.Info("uploading video to R2", zap.String("job_id", p.JobID), zap.String("path", p.VideoPath))

	f, err := os.Open(p.VideoPath)
	if err != nil {
		_ = h.videoRepo.UpdateStatus(ctx, jobID, video.JobStatusFailed, "open video file: "+err.Error())
		return fmt.Errorf("open video: %w", err)
	}
	defer f.Close()

	r2Key := fmt.Sprintf("videos/%s/output.mp4", jobID)
	if err := h.r2.Upload(ctx, r2Key, f, "video/mp4"); err != nil {
		_ = h.videoRepo.UpdateStatus(ctx, jobID, video.JobStatusFailed, "R2 upload failed: "+err.Error())
		return fmt.Errorf("R2 upload: %w", err)
	}
	publicURL := h.r2.PublicURL(r2Key)

	now := time.Now()
	job.OutputVideoKey = r2Key
	job.OutputVideoURL = publicURL
	job.Status = video.JobStatusDone
	job.CompletedAt = &now
	if err := h.videoRepo.Update(ctx, job); err != nil {
		return err
	}

	// Update content plan status
	if job.ContentPlanID != uuid.Nil {
		_ = h.planRepo.UpdateStatus(ctx, job.ContentPlanID, "video_ready")

		// Auto-publish: if this plan was created by an auto-pilot profile with
		// AutoPublish=true, create scheduled publish_jobs for the user's active
		// channels matching the profile's target platforms. The check_publish
		// cron picks them up within 1 minute.
		if err := h.maybeAutoPublish(ctx, job.ContentPlanID, job.ID); err != nil {
			logger.Warn("auto-publish failed (continuing)",
				zap.String("plan_id", job.ContentPlanID.String()), zap.Error(err))
		}
	}

	// Clean up temp files
	h.assembler.CleanupTempDir(jobID.String())

	logger.Info("video uploaded, job done", zap.String("job_id", p.JobID), zap.String("url", publicURL))
	return nil
}

// maybeAutoPublish creates publish_jobs when the plan belongs to an auto-pilot
// profile with auto_publish enabled. Each (platform, channel) pair gets its own
// publish_job, scheduled +1 minute so the check_publish cron picks it up cleanly.
func (h *R2UploadHandler) maybeAutoPublish(ctx context.Context, planID, videoJobID uuid.UUID) error {
	plan, err := h.planRepo.GetByID(ctx, planID)
	if err != nil || plan.AutoPilotProfileID == nil {
		return nil
	}
	profile, err := h.autoPilotRepo.GetByID(ctx, *plan.AutoPilotProfileID)
	if err != nil || !profile.AutoPublish {
		return nil
	}

	channels, err := h.channelRepo.ListByUserID(ctx, profile.UserID)
	if err != nil {
		return err
	}

	caption, hashtags := extractCaptionAndHashtags(plan)
	scheduledAt := time.Now().Add(1 * time.Minute)

	created := 0
	for _, target := range profile.TargetPlatforms {
		for _, ch := range channels {
			if !ch.IsActive || !strings.EqualFold(string(ch.Platform), target) {
				continue
			}
			// Idempotency: skip if a publish_job already exists for this
			// (video_job, channel) pair — guards against R2-upload retries.
			if exists, _ := h.publishRepo.ExistsForVideoJobChannel(ctx, videoJobID, ch.ID); exists {
				continue
			}
			pcopy := planID
			pj := &publish.Job{
				ID:            uuid.New(),
				VideoJobID:    videoJobID,
				ChannelID:     ch.ID,
				ContentPlanID: &pcopy,
				Platform:      string(ch.Platform),
				Caption:       caption,
				Hashtags:      hashtags,
				ScheduledAt:   &scheduledAt,
				Status:        publish.JobStatusScheduled,
			}
			if err := h.publishRepo.Create(ctx, pj); err != nil {
				logger.Warn("auto-publish: create publish_job failed",
					zap.String("channel_id", ch.ID.String()), zap.Error(err))
				continue
			}
			created++
		}
	}
	if created > 0 {
		_ = h.planRepo.UpdateStatus(ctx, planID, content.StatusScheduled)
		logger.Info("auto-publish queued",
			zap.String("plan_id", planID.String()), zap.Int("jobs", created))
	}
	return nil
}

func extractCaptionAndHashtags(plan *content.ContentPlan) (string, []string) {
	var meta struct {
		Caption  string   `json:"caption"`
		Hashtags []string `json:"hashtags"`
	}
	_ = json.Unmarshal(plan.ScriptMetadata, &meta)
	caption := meta.Caption
	if caption == "" {
		caption = plan.Title
	}
	return caption, meta.Hashtags
}
