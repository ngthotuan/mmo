package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"mmo/internal/domain/video"
	apperr "mmo/pkg/errors"
	"mmo/pkg/util"
)

type VideoJobRepo struct{ db *sqlx.DB }

func NewVideoJobRepo(db *sqlx.DB) *VideoJobRepo { return &VideoJobRepo{db: db} }

func (r *VideoJobRepo) Create(ctx context.Context, j *video.Job) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO video_jobs
			(id, content_plan_id, template_id, status, media_assets,
			 tts_audio_key, subtitle_key, output_video_key, output_video_url,
			 duration_seconds, file_size_bytes, retry_count)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		j.ID, j.ContentPlanID, j.TemplateID, j.Status, j.MediaAssets,
		j.TTSAudioKey, j.SubtitleKey, j.OutputVideoKey, j.OutputVideoURL,
		j.DurationSeconds, j.FileSizeBytes, j.RetryCount,
	)
	return err
}

func (r *VideoJobRepo) GetByID(ctx context.Context, id uuid.UUID) (*video.Job, error) {
	var j video.Job
	if err := r.db.GetContext(ctx, &j, `SELECT * FROM video_jobs WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	return &j, nil
}

func (r *VideoJobRepo) List(ctx context.Context, userID *uuid.UUID, status video.JobStatus, p util.Pagination) ([]*video.Job, int, error) {
	args := []any{}
	where := "1=1"

	if userID != nil {
		args = append(args, *userID)
		where = fmt.Sprintf("content_plan_id IN (SELECT id FROM content_plans WHERE user_id = $%d)", len(args))
	}
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(" AND status = $%d", len(args))
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM video_jobs WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, p.Limit(), p.Offset())
	var jobs []*video.Job
	if err := r.db.SelectContext(ctx, &jobs,
		fmt.Sprintf("SELECT * FROM video_jobs WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
			where, len(args)-1, len(args)), args...,
	); err != nil {
		return nil, 0, err
	}
	return jobs, total, nil
}

func (r *VideoJobRepo) Update(ctx context.Context, j *video.Job) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE video_jobs SET
			status=$1, media_assets=$2, tts_audio_key=$3, subtitle_key=$4,
			output_video_key=$5, output_video_url=$6, duration_seconds=$7,
			file_size_bytes=$8, ffmpeg_log=$9, retry_count=$10,
			error_message=$11, started_at=$12, completed_at=$13
		WHERE id=$14`,
		j.Status, j.MediaAssets, j.TTSAudioKey, j.SubtitleKey,
		j.OutputVideoKey, j.OutputVideoURL, j.DurationSeconds,
		j.FileSizeBytes, j.FFmpegLog, j.RetryCount,
		j.ErrorMessage, j.StartedAt, j.CompletedAt, j.ID,
	)
	return err
}

func (r *VideoJobRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status video.JobStatus, errMsg string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE video_jobs SET status=$1, error_message=$2 WHERE id=$3`,
		status, errMsg, id)
	return err
}

func (r *VideoJobRepo) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM video_jobs WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *VideoJobRepo) GetByContentPlanID(ctx context.Context, planID uuid.UUID) (*video.Job, error) {
	var j video.Job
	err := r.db.GetContext(ctx, &j, `SELECT * FROM video_jobs WHERE content_plan_id = $1 ORDER BY created_at DESC LIMIT 1`, planID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return &j, err
}

// ─── Template repository ─────────────────────────────────────────────────────

type VideoTemplateRepo struct{ db *sqlx.DB }

func NewVideoTemplateRepo(db *sqlx.DB) *VideoTemplateRepo { return &VideoTemplateRepo{db: db} }

func (r *VideoTemplateRepo) Create(ctx context.Context, t *video.Template) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO video_templates (id, user_id, name, type, config, is_default) VALUES ($1,$2,$3,$4,$5,$6)`,
		t.ID, t.UserID, t.Name, t.Type, t.Config, t.IsDefault)
	return err
}

func (r *VideoTemplateRepo) GetByID(ctx context.Context, id uuid.UUID) (*video.Template, error) {
	var t video.Template
	if err := r.db.GetContext(ctx, &t, `SELECT * FROM video_templates WHERE id=$1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *VideoTemplateRepo) GetDefault(ctx context.Context) (*video.Template, error) {
	var t video.Template
	err := r.db.GetContext(ctx, &t, `SELECT * FROM video_templates WHERE is_default=TRUE LIMIT 1`)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return &t, err
}

func (r *VideoTemplateRepo) List(ctx context.Context, userID uuid.UUID) ([]*video.Template, error) {
	var templates []*video.Template
	err := r.db.SelectContext(ctx, &templates,
		`SELECT * FROM video_templates WHERE user_id=$1 OR user_id IS NULL ORDER BY created_at DESC`, userID)
	return templates, err
}

func (r *VideoTemplateRepo) Update(ctx context.Context, t *video.Template) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE video_templates SET name=$1, type=$2, config=$3, is_default=$4 WHERE id=$5`,
		t.Name, t.Type, t.Config, t.IsDefault, t.ID)
	return err
}

func (r *VideoTemplateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM video_templates WHERE id=$1`, id)
	return err
}
