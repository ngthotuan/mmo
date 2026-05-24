package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/video"
	"mmo/internal/infrastructure/queue"
	"mmo/internal/infrastructure/storage"
	apperr "mmo/pkg/errors"
)

type VideoUsecase struct {
	videoRepo       *repository.VideoJobRepo
	templateRepo    *repository.VideoTemplateRepo
	r2              *storage.R2Client
	queue           *asynq.Client
	presignedURLTTL time.Duration
}

func NewVideoUsecase(
	videoRepo *repository.VideoJobRepo,
	templateRepo *repository.VideoTemplateRepo,
	r2 *storage.R2Client,
	queueClient *asynq.Client,
	presignedURLTTL time.Duration,
) *VideoUsecase {
	return &VideoUsecase{
		videoRepo:       videoRepo,
		templateRepo:    templateRepo,
		r2:              r2,
		queue:           queueClient,
		presignedURLTTL: presignedURLTTL,
	}
}

func (u *VideoUsecase) List(ctx context.Context, userID uuid.UUID, status string, page, perPage int) ([]*video.Job, int, error) {
	p := paginationOf(page, perPage)
	return u.videoRepo.List(ctx, &userID, video.JobStatus(status), p)
}

func (u *VideoUsecase) GetByID(ctx context.Context, id uuid.UUID) (*video.Job, error) {
	return u.videoRepo.GetByID(ctx, id)
}

func (u *VideoUsecase) GetDownloadURL(ctx context.Context, id uuid.UUID) (string, error) {
	job, err := u.videoRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if job.OutputVideoKey == "" {
		return "", apperr.ErrNotFound
	}
	url, err := u.r2.PresignGet(ctx, job.OutputVideoKey, u.presignedURLTTL)
	if err != nil {
		return "", fmt.Errorf("presign: %w", err)
	}
	return url, nil
}

func (u *VideoUsecase) RetryJob(ctx context.Context, id uuid.UUID) error {
	job, err := u.videoRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if job.Status != video.JobStatusFailed {
		return apperr.New(http.StatusBadRequest, "job is not in failed state")
	}

	job.Status = video.JobStatusMediaCollecting
	job.RetryCount++
	job.ErrorMessage = ""
	if err := u.videoRepo.Update(ctx, job); err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]string{
		"content_plan_id": job.ContentPlanID.String(),
	})
	t := asynq.NewTask(queue.TaskCollectMedia, payload, asynq.Queue(queue.QueueVideo))
	_, err = u.queue.EnqueueContext(ctx, t)
	return err
}

func (u *VideoUsecase) Delete(ctx context.Context, id uuid.UUID) error {
	job, err := u.videoRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if job.OutputVideoKey != "" {
		_ = u.r2.Delete(ctx, job.OutputVideoKey)
	}
	return u.videoRepo.Delete(ctx, id)
}

func (u *VideoUsecase) ListTemplates(ctx context.Context, userID uuid.UUID) ([]*video.Template, error) {
	return u.templateRepo.List(ctx, userID)
}
