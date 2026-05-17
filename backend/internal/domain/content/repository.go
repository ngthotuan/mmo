package content

import (
	"context"

	"github.com/google/uuid"
	"mmo/pkg/util"
)

type TrendRepository interface {
	Create(ctx context.Context, t *TrendTopic) error
	GetByID(ctx context.Context, id uuid.UUID) (*TrendTopic, error)
	List(ctx context.Context, userID uuid.UUID, status string, p util.Pagination) ([]*TrendTopic, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}

type PlanRepository interface {
	Create(ctx context.Context, p *ContentPlan) error
	GetByID(ctx context.Context, id uuid.UUID) (*ContentPlan, error)
	List(ctx context.Context, userID uuid.UUID, status Status, p util.Pagination) ([]*ContentPlan, int, error)
	Update(ctx context.Context, p *ContentPlan) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error
	Delete(ctx context.Context, id uuid.UUID) error
}
