package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"mmo/internal/domain/channel"
	apperr "mmo/pkg/errors"
)

// ChannelRepoWithAll is the concrete type; it implements channel.Repository
// plus ListAllActive used by the token refresh worker.
type ChannelRepoWithAll struct {
	db *sqlx.DB
}

func NewChannelRepo(db *sqlx.DB) *ChannelRepoWithAll {
	return &ChannelRepoWithAll{db: db}
}


func (r *ChannelRepoWithAll) Create(ctx context.Context, ch *channel.Channel) error {
	meta, _ := json.Marshal(ch.Metadata)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO channels
			(id, user_id, platform, platform_user_id, username, display_name, avatar_url,
			 access_token, refresh_token, token_expires_at, page_id, is_active, dry_run, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		ch.ID, ch.UserID, ch.Platform, ch.PlatformUserID, ch.Username, ch.DisplayName,
		ch.AvatarURL, ch.AccessToken, ch.RefreshToken, ch.TokenExpiresAt, ch.PageID,
		ch.IsActive, ch.DryRun, meta,
	)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.ErrConflict
		}
		return fmt.Errorf("create channel: %w", err)
	}
	return nil
}

func (r *ChannelRepoWithAll) GetByID(ctx context.Context, id uuid.UUID) (*channel.Channel, error) {
	var row channelRow
	if err := r.db.GetContext(ctx, &row, `SELECT * FROM channels WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	return row.toChannel(), nil
}

func (r *ChannelRepoWithAll) GetByPlatformUserID(ctx context.Context, platform channel.Platform, platformUserID string) (*channel.Channel, error) {
	var row channelRow
	err := r.db.GetContext(ctx, &row,
		`SELECT * FROM channels WHERE platform = $1 AND platform_user_id = $2`,
		platform, platformUserID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	return row.toChannel(), nil
}

func (r *ChannelRepoWithAll) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*channel.Channel, error) {
	var rows []channelRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT * FROM channels WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	); err != nil {
		return nil, err
	}
	out := make([]*channel.Channel, len(rows))
	for i, row := range rows {
		out[i] = row.toChannel()
	}
	return out, nil
}

func (r *ChannelRepoWithAll) Update(ctx context.Context, ch *channel.Channel) error {
	meta, _ := json.Marshal(ch.Metadata)
	_, err := r.db.ExecContext(ctx, `
		UPDATE channels SET
			username=$1, display_name=$2, avatar_url=$3,
			access_token=$4, refresh_token=$5, token_expires_at=$6,
			is_active=$7, metadata=$8
		WHERE id=$9`,
		ch.Username, ch.DisplayName, ch.AvatarURL,
		ch.AccessToken, ch.RefreshToken, ch.TokenExpiresAt,
		ch.IsActive, meta, ch.ID,
	)
	return err
}

func (r *ChannelRepoWithAll) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM channels WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *ChannelRepoWithAll) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE channels SET is_active=$1 WHERE id=$2`, active, id)
	return err
}

// ListAllActive returns all active channels across all users — used by token refresh job.
func (r *ChannelRepoWithAll) ListAllActive(ctx context.Context) ([]*channel.Channel, error) {
	var rows []channelRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT * FROM channels WHERE is_active = TRUE`); err != nil {
		return nil, err
	}
	out := make([]*channel.Channel, len(rows))
	for i, row := range rows {
		out[i] = row.toChannel()
	}
	return out, nil
}

// channelRow maps DB columns (metadata as []byte for JSONB).
type channelRow struct {
	channel.Channel
	MetadataJSON []byte `db:"metadata"`
}

func (r channelRow) toChannel() *channel.Channel {
	ch := r.Channel
	ch.Metadata = r.MetadataJSON
	return &ch
}
