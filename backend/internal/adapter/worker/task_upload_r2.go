package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/video"
	"mmo/internal/infrastructure/ffmpeg"
	"mmo/internal/infrastructure/storage"
	"mmo/pkg/logger"
	"go.uber.org/zap"
)

type R2UploadHandler struct {
	videoRepo *repository.VideoJobRepo
	planRepo  *repository.ContentPlanRepo
	r2        *storage.R2Client
	assembler *ffmpeg.Assembler
}

func NewR2UploadHandler(
	videoRepo *repository.VideoJobRepo,
	planRepo *repository.ContentPlanRepo,
	r2 *storage.R2Client,
	assembler *ffmpeg.Assembler,
) *R2UploadHandler {
	return &R2UploadHandler{
		videoRepo: videoRepo,
		planRepo:  planRepo,
		r2:        r2,
		assembler: assembler,
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
	}

	// Clean up temp files
	h.assembler.CleanupTempDir(jobID.String())

	logger.Info("video uploaded, job done", zap.String("job_id", p.JobID), zap.String("url", publicURL))
	return nil
}
