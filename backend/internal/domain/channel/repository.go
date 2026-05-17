package channel

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, ch *Channel) error
	GetByID(ctx context.Context, id uuid.UUID) (*Channel, error)
	GetByPlatformUserID(ctx context.Context, platform Platform, platformUserID string) (*Channel, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*Channel, error)
	Update(ctx context.Context, ch *Channel) error
	Delete(ctx context.Context, id uuid.UUID) error
	SetActive(ctx context.Context, id uuid.UUID, active bool) error
}
