package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"mmo/internal/domain/user"
	apperr "mmo/pkg/errors"
	"mmo/pkg/util"
)

type UserRepo struct{ db *sqlx.DB }

func NewUserRepo(db *sqlx.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) List(ctx context.Context, pg util.Pagination) ([]*user.User, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&total); err != nil {
		return nil, 0, err
	}

	users := []*user.User{}
	if err := r.db.SelectContext(ctx, &users,
		`SELECT id, email, name, role, created_at, updated_at
		   FROM users ORDER BY created_at ASC LIMIT $1 OFFSET $2`,
		pg.Limit(), pg.Offset(),
	); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	var u user.User
	if err := r.db.GetContext(ctx, &u,
		`SELECT id, email, name, role, created_at, updated_at FROM users WHERE id = $1`, id,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) UpdateRole(ctx context.Context, id uuid.UUID, role string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`, role, id,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *UserRepo) Count(ctx context.Context) (int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&total)
	return total, err
}
