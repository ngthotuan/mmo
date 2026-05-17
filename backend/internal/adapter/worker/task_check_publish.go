package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
	"mmo/internal/adapter/repository"
	"mmo/internal/infrastructure/queue"
	"mmo/pkg/logger"
	"go.uber.org/zap"
)

// CheckPublishHandler enqueues any scheduled publish jobs that are due.
type CheckPublishHandler struct {
	publishRepo *repository.PublishJobRepo
	queueClient *asynq.Client
}

func NewCheckPublishHandler(publishRepo *repository.PublishJobRepo, queueClient *asynq.Client) *CheckPublishHandler {
	return &CheckPublishHandler{publishRepo: publishRepo, queueClient: queueClient}
}

func (h *CheckPublishHandler) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	due, err := h.publishRepo.ListDue(ctx, time.Now())
	if err != nil {
		return err
	}
	for _, job := range due {
		payload, _ := json.Marshal(map[string]string{"publish_job_id": job.ID.String()})
		t := asynq.NewTask(queue.TaskPublishNow, payload, asynq.Queue(queue.QueueCritical))
		if _, err := h.queueClient.EnqueueContext(ctx, t); err != nil {
			logger.Warn("failed to enqueue publish job",
				zap.String("id", job.ID.String()), zap.Error(err))
		}
	}
	if len(due) > 0 {
		logger.Info("enqueued due publish jobs", zap.Int("count", len(due)))
	}
	return nil
}
