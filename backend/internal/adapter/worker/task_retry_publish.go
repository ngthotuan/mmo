package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"mmo/internal/adapter/repository"
	"mmo/internal/infrastructure/queue"
	"mmo/pkg/logger"
)

// RetryPublishHandler periodically requeues failed publish jobs whose backoff
// window (next_retry_at) has elapsed and that still have retry budget left.
type RetryPublishHandler struct {
	publishRepo *repository.PublishJobRepo
	queueClient *asynq.Client
	maxAttempts int
}

func NewRetryPublishHandler(publishRepo *repository.PublishJobRepo, queueClient *asynq.Client, maxAttempts int) *RetryPublishHandler {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	return &RetryPublishHandler{publishRepo: publishRepo, queueClient: queueClient, maxAttempts: maxAttempts}
}

func (h *RetryPublishHandler) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	jobs, err := h.publishRepo.ListRetryable(ctx, time.Now(), h.maxAttempts)
	if err != nil {
		return err
	}
	requeued := 0
	for _, job := range jobs {
		if err := h.publishRepo.Requeue(ctx, job.ID); err != nil {
			logger.Warn("requeue publish job failed", zap.String("id", job.ID.String()), zap.Error(err))
			continue
		}
		payload, _ := json.Marshal(map[string]string{"publish_job_id": job.ID.String()})
		t := asynq.NewTask(queue.TaskPublishNow, payload, asynq.Queue(queue.QueueCritical))
		if _, err := h.queueClient.EnqueueContext(ctx, t); err != nil {
			logger.Warn("enqueue retry publish failed", zap.String("id", job.ID.String()), zap.Error(err))
			continue
		}
		requeued++
	}
	if requeued > 0 {
		logger.Info("requeued failed publish jobs for retry", zap.Int("count", requeued))
	}
	return nil
}
