package queue

import (
	"context"

	"github.com/hibiken/asynq"
	"mmo/pkg/config"
)

func NewClient(redisURL string) *asynq.Client {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		panic("invalid redis URL for asynq: " + err.Error())
	}
	return asynq.NewClient(opt)
}

func NewServer(redisURL string, videoOnly bool, queueCfg config.QueueConfig) *asynq.Server {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		panic("invalid redis URL for asynq: " + err.Error())
	}

	queues := map[string]int{
		QueueCritical: queueCfg.Weights.Critical,
		QueueDefault:  queueCfg.Weights.Default,
		QueueLow:      queueCfg.Weights.Low,
	}

	concurrency := queueCfg.GeneralConcurrency
	if videoOnly {
		queues = map[string]int{QueueVideo: queueCfg.VideoConcurrency}
		concurrency = queueCfg.VideoConcurrency
	}

	return asynq.NewServer(opt, asynq.Config{
		Concurrency: concurrency,
		Queues:      queues,
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
		}),
	})
}

func NewScheduler(redisURL string) *asynq.Scheduler {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		panic("invalid redis URL for asynq: " + err.Error())
	}
	return asynq.NewScheduler(opt, nil)
}
