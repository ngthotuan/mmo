package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"mmo/internal/domain/autopilot"
	apperr "mmo/pkg/errors"
)

type AutoPilotRepo struct{ db *sqlx.DB }

func NewAutoPilotRepo(db *sqlx.DB) *AutoPilotRepo { return &AutoPilotRepo{db: db} }

type autoPilotRow struct {
	ID              uuid.UUID      `db:"id"`
	UserID          uuid.UUID      `db:"user_id"`
	Name            string         `db:"name"`
	Niche           string         `db:"niche"`
	Voice           string         `db:"voice"`
	TargetPlatforms pq.StringArray `db:"target_platforms"`
	TrendFilter     string         `db:"trend_filter"`
	TrendSources    pq.StringArray `db:"trend_sources"`
	DailyCount      int            `db:"daily_count"`
	ScheduleTimes   pq.StringArray `db:"schedule_times"`
	AutoApprove     bool           `db:"auto_approve"`
	AutoPublish     bool           `db:"auto_publish"`
	Enabled         bool           `db:"enabled"`
	LastRunAt       *time.Time     `db:"last_run_at"`
	LastRunCount    int            `db:"last_run_count"`
	TotalVideos     int            `db:"total_videos"`
	CreatedAt       time.Time      `db:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at"`
}

func (row autoPilotRow) toEntity() *autopilot.Profile {
	return &autopilot.Profile{
		ID:              row.ID,
		UserID:          row.UserID,
		Name:            row.Name,
		Niche:           row.Niche,
		Voice:           row.Voice,
		TargetPlatforms: []string(row.TargetPlatforms),
		TrendFilter:     row.TrendFilter,
		TrendSources:    []string(row.TrendSources),
		DailyCount:      row.DailyCount,
		ScheduleTimes:   []string(row.ScheduleTimes),
		AutoApprove:     row.AutoApprove,
		AutoPublish:     row.AutoPublish,
		Enabled:         row.Enabled,
		LastRunAt:       row.LastRunAt,
		LastRunCount:    row.LastRunCount,
		TotalVideos:     row.TotalVideos,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func (r *AutoPilotRepo) Create(ctx context.Context, p *autopilot.Profile) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO auto_pilot_profiles
			(id, user_id, name, niche, voice, target_platforms, trend_filter, trend_sources,
			 daily_count, schedule_times, auto_approve, auto_publish, enabled)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		p.ID, p.UserID, p.Name, p.Niche, p.Voice,
		pq.Array(p.TargetPlatforms), p.TrendFilter, pq.Array(p.TrendSources),
		p.DailyCount, pq.Array(p.ScheduleTimes),
		p.AutoApprove, p.AutoPublish, p.Enabled,
	)
	return err
}

func (r *AutoPilotRepo) Update(ctx context.Context, p *autopilot.Profile) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE auto_pilot_profiles SET
			name=$1, niche=$2, voice=$3, target_platforms=$4, trend_filter=$5,
			trend_sources=$6, daily_count=$7, schedule_times=$8,
			auto_approve=$9, auto_publish=$10, enabled=$11
		WHERE id=$12 AND user_id=$13`,
		p.Name, p.Niche, p.Voice, pq.Array(p.TargetPlatforms), p.TrendFilter,
		pq.Array(p.TrendSources), p.DailyCount, pq.Array(p.ScheduleTimes),
		p.AutoApprove, p.AutoPublish, p.Enabled,
		p.ID, p.UserID,
	)
	return err
}

func (r *AutoPilotRepo) GetByID(ctx context.Context, id uuid.UUID) (*autopilot.Profile, error) {
	var row autoPilotRow
	if err := r.db.GetContext(ctx, &row,
		`SELECT * FROM auto_pilot_profiles WHERE id=$1`, id,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	return row.toEntity(), nil
}

func (r *AutoPilotRepo) List(ctx context.Context, userID uuid.UUID) ([]*autopilot.Profile, error) {
	var rows []autoPilotRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT * FROM auto_pilot_profiles WHERE user_id=$1 ORDER BY created_at DESC`,
		userID,
	); err != nil {
		return nil, err
	}
	out := make([]*autopilot.Profile, len(rows))
	for i, row := range rows {
		out[i] = row.toEntity()
	}
	return out, nil
}

// ListEnabled returns all profiles across all users that are currently enabled.
// Used by the auto-pilot tick task.
func (r *AutoPilotRepo) ListEnabled(ctx context.Context) ([]*autopilot.Profile, error) {
	var rows []autoPilotRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT * FROM auto_pilot_profiles WHERE enabled = TRUE`,
	); err != nil {
		return nil, err
	}
	out := make([]*autopilot.Profile, len(rows))
	for i, row := range rows {
		out[i] = row.toEntity()
	}
	return out, nil
}

func (r *AutoPilotRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM auto_pilot_profiles WHERE id=$1`, id)
	return err
}

func (r *AutoPilotRepo) Toggle(ctx context.Context, id uuid.UUID, enabled bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE auto_pilot_profiles SET enabled=$1 WHERE id=$2`, enabled, id)
	return err
}

// MarkRun updates last_run_at + counters after a successful tick.
func (r *AutoPilotRepo) MarkRun(ctx context.Context, id uuid.UUID, count int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE auto_pilot_profiles
		SET last_run_at = NOW(), last_run_count = $1, total_videos = total_videos + $1
		WHERE id = $2`, count, id)
	return err
}
