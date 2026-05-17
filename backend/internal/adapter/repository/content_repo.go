package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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
	var t content.TrendTopic
	err := r.db.GetContext(ctx, &t, `SELECT * FROM trend_topics WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return &t, err
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
	rows := []*content.TrendTopic{}
	if err := r.db.SelectContext(ctx, &rows,
		fmt.Sprintf("SELECT * FROM trend_topics WHERE %s ORDER BY discovered_at DESC LIMIT $%d OFFSET $%d",
			where, len(args)-1, len(args)), args...,
	); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *TrendRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE trend_topics SET status=$1 WHERE id=$2`, status, id)
	return err
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

func (r *ContentPlanRepo) Create(ctx context.Context, p *content.ContentPlan) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO content_plans
			(id, user_id, trend_topic_id, video_template_id, title, niche,
			 target_platforms, script, script_metadata, status, auto_approve, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		p.ID, p.UserID, p.TrendTopicID, p.VideoTemplateID, p.Title, p.Niche,
		pq.Array(p.TargetPlatforms), p.Script, p.ScriptMetadata,
		p.Status, p.AutoApprove, p.Notes,
	)
	return err
}

func (r *ContentPlanRepo) GetByID(ctx context.Context, id uuid.UUID) (*content.ContentPlan, error) {
	var p content.ContentPlan
	err := r.db.GetContext(ctx, &p, `SELECT * FROM content_plans WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return &p, err
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
	plans := []*content.ContentPlan{}
	if err := r.db.SelectContext(ctx, &plans,
		fmt.Sprintf("SELECT * FROM content_plans WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
			where, len(args)-1, len(args)), args...,
	); err != nil {
		return nil, 0, err
	}
	return plans, total, nil
}

func (r *ContentPlanRepo) Update(ctx context.Context, p *content.ContentPlan) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE content_plans SET
			title=$1, niche=$2, target_platforms=$3, script=$4, script_metadata=$5,
			status=$6, auto_approve=$7, notes=$8, video_template_id=$9
		WHERE id=$10`,
		p.Title, p.Niche, pq.Array(p.TargetPlatforms), p.Script, p.ScriptMetadata,
		p.Status, p.AutoApprove, p.Notes, p.VideoTemplateID, p.ID,
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
