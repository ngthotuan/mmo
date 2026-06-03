package user

import (
	"context"

	"github.com/google/uuid"
	"mmo/pkg/util"
)

type Repository interface {
	List(ctx context.Context, pg util.Pagination) ([]*User, int, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateRole(ctx context.Context, id uuid.UUID, role string) error
	Count(ctx context.Context) (int, error)
}
