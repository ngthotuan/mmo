package worker

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"mmo/internal/adapter/repository"
	"mmo/internal/infrastructure/storage"
	"mmo/pkg/logger"
)

// cleanupBatchSize bounds how many video jobs a single cron run purges, so a
// large backlog is drained over several runs rather than in one long task.
const cleanupBatchSize = 500

// DeleteR2VideosHandler periodically deletes the R2 artifacts of videos that
// have been published to all their target channels and whose retention window
// has elapsed, reclaiming storage. A video is only purged once nothing still
// needs the file (no scheduled/publishing job, no failed job with retry budget).
type DeleteR2VideosHandler struct {
	videoRepo   *repository.VideoJobRepo
	r2          *storage.R2Client
	retention   time.Duration
	maxAttempts int
}

func NewDeleteR2VideosHandler(videoRepo *repository.VideoJobRepo, r2 *storage.R2Client, retention time.Duration, maxAttempts int) *DeleteR2VideosHandler {
	if retention <= 0 {
		retention = 48 * time.Hour
	}
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	return &DeleteR2VideosHandler{videoRepo: videoRepo, r2: r2, retention: retention, maxAttempts: maxAttempts}
}

func (h *DeleteR2VideosHandler) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	cutoff := time.Now().Add(-h.retention)
	jobs, err := h.videoRepo.ListR2CleanupTargets(ctx, cutoff, h.maxAttempts, cleanupBatchSize)
	if err != nil {
		return err
	}

	purged := 0
	for _, j := range jobs {
		// Delete the output video plus its intermediate artifacts (TTS audio,
		// subtitles); none are needed once the video is published and retained.
		keys := []string{j.OutputVideoKey, j.TTSAudioKey, j.SubtitleKey}
		failed := false
		for _, key := range keys {
			if key == "" {
				continue
			}
			if err := h.r2.Delete(ctx, key); err != nil {
				logger.Warn("delete r2 object failed",
					zap.String("video_job_id", j.ID.String()), zap.String("key", key), zap.Error(err))
				failed = true
				break
			}
		}
		if failed {
			// Leave r2_deleted_at unset so the next run retries this job.
			continue
		}
		if err := h.videoRepo.MarkR2Deleted(ctx, j.ID, time.Now()); err != nil {
			logger.Warn("mark r2 deleted failed", zap.String("video_job_id", j.ID.String()), zap.Error(err))
			continue
		}
		purged++
	}

	if purged > 0 {
		logger.Info("purged published video R2 artifacts", zap.Int("count", purged))
	}
	return nil
}
