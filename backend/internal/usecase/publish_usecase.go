package usecase

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/publish"
	"mmo/internal/infrastructure/queue"
	apperr "mmo/pkg/errors"
	"mmo/pkg/util"
)

type PublishUsecase struct {
	publishRepo          *repository.PublishJobRepo
	videoRepo            *repository.VideoJobRepo
	channelRepo          *repository.ChannelRepoWithAll
	queue                *asynq.Client
	minScheduleBeforeNow time.Duration
}

func NewPublishUsecase(
	publishRepo *repository.PublishJobRepo,
	videoRepo *repository.VideoJobRepo,
	channelRepo *repository.ChannelRepoWithAll,
	queueClient *asynq.Client,
	minScheduleBeforeNow time.Duration,
) *PublishUsecase {
	return &PublishUsecase{
		publishRepo:          publishRepo,
		videoRepo:            videoRepo,
		channelRepo:          channelRepo,
		queue:                queueClient,
		minScheduleBeforeNow: minScheduleBeforeNow,
	}
}

type CreatePublishRequest struct {
	VideoJobID  uuid.UUID  `json:"video_job_id"`
	ChannelID   uuid.UUID  `json:"channel_id"`
	Caption     string     `json:"caption"`
	Hashtags    []string   `json:"hashtags"`
	ScheduledAt *time.Time `json:"scheduled_at"`
}

func (u *PublishUsecase) Create(ctx context.Context, userID uuid.UUID, req CreatePublishRequest) (*publish.Job, error) {
	videoJob, err := u.videoRepo.GetByID(ctx, req.VideoJobID)
	if err != nil {
		return nil, err
	}

	channel, err := u.channelRepo.GetByID(ctx, req.ChannelID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	scheduledAt := req.ScheduledAt
	if scheduledAt == nil {
		scheduledAt = &now
	}

	planID := videoJob.ContentPlanID
	job := &publish.Job{
		ID:            uuid.New(),
		VideoJobID:    req.VideoJobID,
		ChannelID:     req.ChannelID,
		ContentPlanID: &planID,
		Platform:      string(channel.Platform),
		Caption:       req.Caption,
		Hashtags:      req.Hashtags,
		ScheduledAt:   scheduledAt,
		Status:        publish.JobStatusScheduled,
	}
	if job.Hashtags == nil {
		job.Hashtags = []string{}
	}

	if err := u.publishRepo.Create(ctx, job); err != nil {
		return nil, err
	}

	// If scheduled for now, enqueue immediately
	if !scheduledAt.After(time.Now().Add(u.minScheduleBeforeNow)) {
		payload, _ := json.Marshal(map[string]string{"publish_job_id": job.ID.String()})
		t := asynq.NewTask(queue.TaskPublishNow, payload, asynq.Queue(queue.QueueCritical))
		_, _ = u.queue.EnqueueContext(ctx, t)
	}

	return job, nil
}

func (u *PublishUsecase) List(ctx context.Context, userID uuid.UUID, status string, page, perPage int) ([]*publish.Job, int, error) {
	p := util.Pagination{Page: page, PerPage: perPage}
	return u.publishRepo.List(ctx, &userID, publish.JobStatus(status), p)
}

func (u *PublishUsecase) GetByID(ctx context.Context, id uuid.UUID) (*publish.Job, error) {
	return u.publishRepo.GetByID(ctx, id)
}

func (u *PublishUsecase) Update(ctx context.Context, id uuid.UUID, caption string, hashtags []string, scheduledAt *time.Time) (*publish.Job, error) {
	job, err := u.publishRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if job.Status == publish.JobStatusPublished || job.Status == publish.JobStatusPublishing {
		return nil, apperr.New(http.StatusBadRequest, "cannot update a job that is already publishing or published")
	}
	job.Caption = caption
	job.Hashtags = hashtags
	if scheduledAt != nil {
		job.ScheduledAt = scheduledAt
	}
	if err := u.publishRepo.Update(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

func (u *PublishUsecase) Cancel(ctx context.Context, id uuid.UUID) error {
	job, err := u.publishRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if job.Status == publish.JobStatusPublished {
		return apperr.New(http.StatusBadRequest, "cannot cancel a published job")
	}
	return u.publishRepo.UpdateStatus(ctx, id, publish.JobStatusCancelled, "")
}

func (u *PublishUsecase) PublishNow(ctx context.Context, id uuid.UUID) error {
	job, err := u.publishRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if job.Status == publish.JobStatusPublished || job.Status == publish.JobStatusPublishing {
		return apperr.New(http.StatusBadRequest, "job is already publishing or published")
	}
	now := time.Now()
	job.ScheduledAt = &now
	job.Status = publish.JobStatusScheduled
	if err := u.publishRepo.Update(ctx, job); err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]string{"publish_job_id": id.String()})
	t := asynq.NewTask(queue.TaskPublishNow, payload, asynq.Queue(queue.QueueCritical))
	_, err = u.queue.EnqueueContext(ctx, t)
	return err
}

func (u *PublishUsecase) ListByDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*publish.Job, error) {
	return u.publishRepo.ListByDateRange(ctx, userID, start, end)
}
