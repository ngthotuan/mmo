package publish

import (
	"context"
	"time"

	"github.com/google/uuid"
	"mmo/pkg/util"
)

type JobRepository interface {
	Create(ctx context.Context, j *Job) error
	GetByID(ctx context.Context, id uuid.UUID) (*Job, error)
	List(ctx context.Context, channelID *uuid.UUID, status JobStatus, p util.Pagination) ([]*Job, int, error)
	ListDue(ctx context.Context, before time.Time) ([]*Job, error)
	ListByDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*Job, error)
	Update(ctx context.Context, j *Job) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status JobStatus, errMsg string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type AnalyticsRepository interface {
	Upsert(ctx context.Context, a *Analytics) error
	ListByPublishJob(ctx context.Context, publishJobID uuid.UUID) ([]*Analytics, error)
	GetOverview(ctx context.Context, userID uuid.UUID) (map[string]int64, error)
}
