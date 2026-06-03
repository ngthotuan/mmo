package usecase

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"mmo/internal/domain/user"
	apperr "mmo/pkg/errors"
	"mmo/pkg/util"
)

type AdminUsecase struct {
	users user.Repository
}

func NewAdminUsecase(users user.Repository) *AdminUsecase {
	return &AdminUsecase{users: users}
}

func (u *AdminUsecase) ListUsers(ctx context.Context, pg util.Pagination) ([]*user.User, int, error) {
	return u.users.List(ctx, pg)
}

// UpdateRole changes a user's role. actingUserID is the admin performing the
// change; an admin may not change their own role (prevents self-lockout).
func (u *AdminUsecase) UpdateRole(ctx context.Context, actingUserID, targetID uuid.UUID, role string) (*user.User, error) {
	if !user.IsAssignableRole(role) {
		return nil, apperr.New(http.StatusBadRequest, "invalid role: must be admin, member, or viewer")
	}
	if actingUserID == targetID {
		return nil, apperr.New(http.StatusBadRequest, "you cannot change your own role")
	}
	if err := u.users.UpdateRole(ctx, targetID, role); err != nil {
		return nil, err
	}
	return u.users.GetByID(ctx, targetID)
}
