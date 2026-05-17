package video

import (
	"context"

	"github.com/google/uuid"
	"mmo/pkg/util"
)

type JobRepository interface {
	Create(ctx context.Context, j *Job) error
	GetByID(ctx context.Context, id uuid.UUID) (*Job, error)
	List(ctx context.Context, status JobStatus, p util.Pagination) ([]*Job, int, error)
	Update(ctx context.Context, j *Job) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status JobStatus, errMsg string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type TemplateRepository interface {
	Create(ctx context.Context, t *Template) error
	GetByID(ctx context.Context, id uuid.UUID) (*Template, error)
	GetDefault(ctx context.Context) (*Template, error)
	List(ctx context.Context, userID uuid.UUID) ([]*Template, error)
	Update(ctx context.Context, t *Template) error
	Delete(ctx context.Context, id uuid.UUID) error
}
