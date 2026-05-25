package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"mmo/internal/domain/content"
	apperr "mmo/pkg/errors"
	"mmo/pkg/util"
)

// ─── TrendTopic repository ────────────────────────────────────────────────────

type TrendRepo struct{ db *sqlx.DB }

func NewTrendRepo(db *sqlx.DB) *TrendRepo { return &TrendRepo{db: db} }

// trendTopicRow is a scan target that handles TEXT[] → []string via pq.StringArray.
type trendTopicRow struct {
	ID            uuid.UUID      `db:"id"`
	UserID        *uuid.UUID     `db:"user_id"`
	Source        string         `db:"source"`
	Title         string         `db:"title"`
	Description   string         `db:"description"`
	Keywords      pq.StringArray `db:"keywords"`
	TrendingScore float64        `db:"trending_score"`
	SourceURL     *string        `db:"source_url"`
	RawData       []byte         `db:"raw_data"`
	Status        string         `db:"status"`
	DiscoveredAt  time.Time      `db:"discovered_at"`
	CreatedAt     time.Time      `db:"created_at"`
}

func (row trendTopicRow) toEntity() *content.TrendTopic {
	return &content.TrendTopic{
		ID:            row.ID,
		UserID:        row.UserID,
		Source:        row.Source,
		Title:         row.Title,
		Description:   row.Description,
		Keywords:      []string(row.Keywords),
		TrendingScore: row.TrendingScore,
		SourceURL:     func() string { if row.SourceURL != nil { return *row.SourceURL }; return "" }(),
		RawData:       row.RawData,
		Status:        row.Status,
		DiscoveredAt:  row.DiscoveredAt,
		CreatedAt:     row.CreatedAt,
	}
}

func (r *TrendRepo) Create(ctx context.Context, t *content.TrendTopic) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trend_topics
			(id, user_id, source, title, description, keywords, trending_score, source_url, raw_data, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		t.ID, t.UserID, t.Source, t.Title, t.Description,
		pq.Array(t.Keywords), t.TrendingScore, t.SourceURL, t.RawData, t.Status,
	)
	return err
}

func (r *TrendRepo) GetByID(ctx context.Context, id uuid.UUID) (*content.TrendTopic, error) {
	var row trendTopicRow
	if err := r.db.GetContext(ctx, &row, `SELECT * FROM trend_topics WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	return row.toEntity(), nil
}

func (r *TrendRepo) List(ctx context.Context, userID uuid.UUID, status string, p util.Pagination) ([]*content.TrendTopic, int, error) {
	args := []any{userID}
	where := "user_id = $1"
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(" AND status = $%d", len(args))
	}

	var total int
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM trend_topics WHERE "+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, p.Limit(), p.Offset())
	var rows []trendTopicRow
	if err := r.db.SelectContext(ctx, &rows,
		fmt.Sprintf("SELECT * FROM trend_topics WHERE %s ORDER BY discovered_at DESC LIMIT $%d OFFSET $%d",
			where, len(args)-1, len(args)), args...,
	); err != nil {
		return nil, 0, err
	}

	trends := make([]*content.TrendTopic, len(rows))
	for i, row := range rows {
		trends[i] = row.toEntity()
	}
	return trends, total, nil
}

func (r *TrendRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE trend_topics SET status=$1 WHERE id=$2`, status, id)
	return err
}

// ListNewMatching returns up to `limit` new trend topics matching the filter/sources.
// `filter` is a case-insensitive substring match against title; empty = no filter.
// `sources` is an allowlist (e.g. ["google_trends","vnexpress"]); empty = any source.
// Trends with NULL user_id (system-wide discovery) and trends owned by the user are both included.
func (r *TrendRepo) ListNewMatching(ctx context.Context, userID uuid.UUID, filter string, sources []string, limit int) ([]*content.TrendTopic, error) {
	if limit <= 0 {
		limit = 10
	}
	args := []any{userID}
	where := "status = 'new' AND (user_id IS NULL OR user_id = $1)"
	if filter != "" {
		args = append(args, "%"+filter+"%")
		where += fmt.Sprintf(" AND title ILIKE $%d", len(args))
	}
	if len(sources) > 0 {
		args = append(args, pq.Array(sources))
		where += fmt.Sprintf(" AND source = ANY($%d)", len(args))
	}
	args = append(args, limit)
	q := fmt.Sprintf(
		"SELECT * FROM trend_topics WHERE %s ORDER BY trending_score DESC, discovered_at DESC LIMIT $%d",
		where, len(args))
	var rows []trendTopicRow
	if err := r.db.SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, err
	}
	out := make([]*content.TrendTopic, len(rows))
	for i, row := range rows {
		out[i] = row.toEntity()
	}
	return out, nil
}

// ExistsBySourceAndTitle prevents duplicate trend topics from the same source.
func (r *TrendRepo) ExistsBySourceAndTitle(ctx context.Context, source, title string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM trend_topics WHERE source=$1 AND title=$2 AND discovered_at > NOW() - INTERVAL '24 hours'`,
		source, title,
	).Scan(&count)
	return count > 0, err
}

// ─── ContentPlan repository ───────────────────────────────────────────────────

type ContentPlanRepo struct{ db *sqlx.DB }

func NewContentPlanRepo(db *sqlx.DB) *ContentPlanRepo { return &ContentPlanRepo{db: db} }

// contentPlanRow is a scan target that handles VARCHAR(50)[] → []string via pq.StringArray.
type contentPlanRow struct {
	ID                 uuid.UUID      `db:"id"`
	UserID             uuid.UUID      `db:"user_id"`
	TrendTopicID       *uuid.UUID     `db:"trend_topic_id"`
	VideoTemplateID    *uuid.UUID     `db:"video_template_id"`
	AutoPilotProfileID *uuid.UUID     `db:"auto_pilot_profile_id"`
	Title              string         `db:"title"`
	Niche              string         `db:"niche"`
	TargetPlatforms    pq.StringArray `db:"target_platforms"`
	Script             string         `db:"script"`
	ScriptMetadata     []byte         `db:"script_metadata"`
	Status             content.Status `db:"status"`
	AutoApprove        bool           `db:"auto_approve"`
	Voice              string         `db:"voice"`
	Notes              string         `db:"notes"`
	CreatedAt          time.Time      `db:"created_at"`
	UpdatedAt          time.Time      `db:"updated_at"`
}

func (row contentPlanRow) toEntity() *content.ContentPlan {
	return &content.ContentPlan{
		ID:                 row.ID,
		UserID:             row.UserID,
		TrendTopicID:       row.TrendTopicID,
		VideoTemplateID:    row.VideoTemplateID,
		AutoPilotProfileID: row.AutoPilotProfileID,
		Title:              row.Title,
		Niche:              row.Niche,
		TargetPlatforms:    []string(row.TargetPlatforms),
		Script:             row.Script,
		ScriptMetadata:     row.ScriptMetadata,
		Status:             row.Status,
		AutoApprove:        row.AutoApprove,
		Voice:              row.Voice,
		Notes:              row.Notes,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func (r *ContentPlanRepo) Create(ctx context.Context, p *content.ContentPlan) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO content_plans
			(id, user_id, trend_topic_id, video_template_id, auto_pilot_profile_id, title, niche,
			 target_platforms, script, script_metadata, status, auto_approve, voice, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		p.ID, p.UserID, p.TrendTopicID, p.VideoTemplateID, p.AutoPilotProfileID, p.Title, p.Niche,
		pq.Array(p.TargetPlatforms), p.Script, p.ScriptMetadata,
		p.Status, p.AutoApprove, p.Voice, p.Notes,
	)
	return err
}

func (r *ContentPlanRepo) GetByID(ctx context.Context, id uuid.UUID) (*content.ContentPlan, error) {
	var row contentPlanRow
	if err := r.db.GetContext(ctx, &row, `SELECT * FROM content_plans WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	return row.toEntity(), nil
}

func (r *ContentPlanRepo) List(ctx context.Context, userID uuid.UUID, status content.Status, p util.Pagination) ([]*content.ContentPlan, int, error) {
	args := []any{userID}
	where := "user_id = $1"
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(" AND status = $%d", len(args))
	}

	var total int
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM content_plans WHERE "+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, p.Limit(), p.Offset())
	var rows []contentPlanRow
	if err := r.db.SelectContext(ctx, &rows,
		fmt.Sprintf("SELECT * FROM content_plans WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
			where, len(args)-1, len(args)), args...,
	); err != nil {
		return nil, 0, err
	}

	plans := make([]*content.ContentPlan, len(rows))
	for i, row := range rows {
		plans[i] = row.toEntity()
	}
	return plans, total, nil
}

func (r *ContentPlanRepo) Update(ctx context.Context, p *content.ContentPlan) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE content_plans SET
			title=$1, niche=$2, target_platforms=$3, script=$4, script_metadata=$5,
			status=$6, auto_approve=$7, voice=$8, notes=$9, video_template_id=$10
		WHERE id=$11`,
		p.Title, p.Niche, pq.Array(p.TargetPlatforms), p.Script, p.ScriptMetadata,
		p.Status, p.AutoApprove, p.Voice, p.Notes, p.VideoTemplateID, p.ID,
	)
	return err
}

func (r *ContentPlanRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status content.Status) error {
	_, err := r.db.ExecContext(ctx, `UPDATE content_plans SET status=$1 WHERE id=$2`, status, id)
	return err
}

func (r *ContentPlanRepo) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM content_plans WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apperr.ErrNotFound
	}
	return nil
}
