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
	"mmo/internal/domain/publish"
	apperr "mmo/pkg/errors"
	"mmo/pkg/util"
)

type PublishJobRepo struct{ db *sqlx.DB }

func NewPublishJobRepo(db *sqlx.DB) *PublishJobRepo { return &PublishJobRepo{db: db} }

// publishJobRow is a scan target that handles TEXT[] → []string via pq.StringArray.
type publishJobRow struct {
	ID              uuid.UUID         `db:"id"`
	VideoJobID      uuid.UUID         `db:"video_job_id"`
	ChannelID       uuid.UUID         `db:"channel_id"`
	ContentPlanID   *uuid.UUID        `db:"content_plan_id"`
	Platform        string            `db:"platform"`
	Caption         string            `db:"caption"`
	Hashtags        pq.StringArray    `db:"hashtags"`
	ScheduledAt     *time.Time        `db:"scheduled_at"`
	PublishedAt     *time.Time        `db:"published_at"`
	PlatformPostID  string            `db:"platform_post_id"`
	PlatformPostURL string            `db:"platform_post_url"`
	Status          publish.JobStatus `db:"status"`
	RetryCount      int               `db:"retry_count"`
	NextRetryAt     *time.Time        `db:"next_retry_at"`
	ErrorMessage    string            `db:"error_message"`
	CreatedAt       time.Time         `db:"created_at"`
	UpdatedAt       time.Time         `db:"updated_at"`
}

func (row publishJobRow) toEntity() *publish.Job {
	return &publish.Job{
		ID:              row.ID,
		VideoJobID:      row.VideoJobID,
		ChannelID:       row.ChannelID,
		ContentPlanID:   row.ContentPlanID,
		Platform:        row.Platform,
		Caption:         row.Caption,
		Hashtags:        []string(row.Hashtags),
		ScheduledAt:     row.ScheduledAt,
		PublishedAt:     row.PublishedAt,
		PlatformPostID:  row.PlatformPostID,
		PlatformPostURL: row.PlatformPostURL,
		Status:          row.Status,
		RetryCount:      row.RetryCount,
		NextRetryAt:     row.NextRetryAt,
		ErrorMessage:    row.ErrorMessage,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func (r *PublishJobRepo) Create(ctx context.Context, j *publish.Job) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO publish_jobs
			(id, video_job_id, channel_id, content_plan_id, platform, caption, hashtags,
			 scheduled_at, published_at, platform_post_id, platform_post_url,
			 status, retry_count, error_message)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		j.ID, j.VideoJobID, j.ChannelID, j.ContentPlanID, j.Platform, j.Caption,
		pq.Array(j.Hashtags), j.ScheduledAt, j.PublishedAt, j.PlatformPostID,
		j.PlatformPostURL, j.Status, j.RetryCount, j.ErrorMessage,
	)
	return err
}

func (r *PublishJobRepo) GetByID(ctx context.Context, id uuid.UUID) (*publish.Job, error) {
	var row publishJobRow
	if err := r.db.GetContext(ctx, &row, `SELECT * FROM publish_jobs WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	return row.toEntity(), nil
}

func (r *PublishJobRepo) List(ctx context.Context, channelID *uuid.UUID, status publish.JobStatus, p util.Pagination) ([]*publish.Job, int, error) {
	args := []any{}
	where := "1=1"

	if channelID != nil {
		args = append(args, *channelID)
		where += fmt.Sprintf(" AND channel_id = $%d", len(args))
	}
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(" AND status = $%d", len(args))
	}

	var total int
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM publish_jobs WHERE "+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, p.Limit(), p.Offset())
	var rows []publishJobRow
	if err := r.db.SelectContext(ctx, &rows,
		fmt.Sprintf("SELECT * FROM publish_jobs WHERE %s ORDER BY scheduled_at DESC LIMIT $%d OFFSET $%d",
			where, len(args)-1, len(args)), args...,
	); err != nil {
		return nil, 0, err
	}
	jobs := make([]*publish.Job, len(rows))
	for i, row := range rows {
		jobs[i] = row.toEntity()
	}
	return jobs, total, nil
}

func (r *PublishJobRepo) Update(ctx context.Context, j *publish.Job) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE publish_jobs SET
			video_job_id=$1, channel_id=$2, content_plan_id=$3, platform=$4,
			caption=$5, hashtags=$6, scheduled_at=$7, published_at=$8,
			platform_post_id=$9, platform_post_url=$10,
			status=$11, retry_count=$12, error_message=$13, updated_at=NOW()
		WHERE id=$14`,
		j.VideoJobID, j.ChannelID, j.ContentPlanID, j.Platform,
		j.Caption, pq.Array(j.Hashtags), j.ScheduledAt, j.PublishedAt,
		j.PlatformPostID, j.PlatformPostURL,
		j.Status, j.RetryCount, j.ErrorMessage, j.ID,
	)
	return err
}

func (r *PublishJobRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status publish.JobStatus, errMsg string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE publish_jobs SET status=$1, error_message=$2, updated_at=NOW() WHERE id=$3`,
		status, errMsg, id)
	return err
}

// Claim atomically flips a job from scheduled→publishing. Returns true only if
// this caller won the claim, so duplicate task deliveries (asynq is at-least-once)
// don't double-publish.
func (r *PublishJobRepo) Claim(ctx context.Context, id uuid.UUID) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE publish_jobs SET status='publishing', error_message='', updated_at=NOW()
		 WHERE id=$1 AND status='scheduled'`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// MarkFailed persists the failure, the incremented retry count, and the next
// retry time (nil if no more retries). Fixes the old bug where RetryCount was
// mutated in memory and dropped.
func (r *PublishJobRepo) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string, retryCount int, nextRetryAt *time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE publish_jobs SET status='failed', error_message=$1, retry_count=$2,
		 next_retry_at=$3, updated_at=NOW() WHERE id=$4`,
		errMsg, retryCount, nextRetryAt, id)
	return err
}

// Requeue resets a failed job back to scheduled so the normal publish path can
// claim and re-run it.
func (r *PublishJobRepo) Requeue(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE publish_jobs SET status='scheduled', next_retry_at=NULL, updated_at=NOW() WHERE id=$1`,
		id)
	return err
}

// ListRetryable returns failed jobs eligible for an automatic retry.
func (r *PublishJobRepo) ListRetryable(ctx context.Context, before time.Time, maxAttempts int) ([]*publish.Job, error) {
	var rows []publishJobRow
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT * FROM publish_jobs
		WHERE status = 'failed' AND retry_count < $1
		  AND next_retry_at IS NOT NULL AND next_retry_at <= $2
		ORDER BY next_retry_at ASC
		LIMIT 50`,
		maxAttempts, before,
	); err != nil {
		return nil, err
	}
	jobs := make([]*publish.Job, len(rows))
	for i, row := range rows {
		jobs[i] = row.toEntity()
	}
	return jobs, nil
}

// ExistsForVideoJobChannel reports whether a publish job already exists for a
// (video_job, channel) pair — used to make auto-publish idempotent on retries.
func (r *PublishJobRepo) ExistsForVideoJobChannel(ctx context.Context, videoJobID, channelID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM publish_jobs WHERE video_job_id=$1 AND channel_id=$2)`,
		videoJobID, channelID).Scan(&exists)
	return exists, err
}

// CountUnfinishedForPlan counts publish jobs for a content plan that are not yet
// in a terminal-success/cancelled state. Zero means every channel has published.
func (r *PublishJobRepo) CountUnfinishedForPlan(ctx context.Context, planID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM publish_jobs
		 WHERE content_plan_id=$1 AND status NOT IN ('published','cancelled')`,
		planID).Scan(&n)
	return n, err
}

func (r *PublishJobRepo) ListDue(ctx context.Context, before time.Time) ([]*publish.Job, error) {
	var rows []publishJobRow
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT * FROM publish_jobs
		WHERE status = 'scheduled' AND scheduled_at <= $1
		ORDER BY scheduled_at ASC
		LIMIT 50`,
		before,
	); err != nil {
		return nil, err
	}
	jobs := make([]*publish.Job, len(rows))
	for i, row := range rows {
		jobs[i] = row.toEntity()
	}
	return jobs, nil
}

func (r *PublishJobRepo) ListByDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*publish.Job, error) {
	var rows []publishJobRow
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT * FROM publish_jobs
		WHERE channel_id IN (SELECT id FROM channels WHERE user_id = $1)
		  AND scheduled_at >= $2
		  AND scheduled_at <= $3
		ORDER BY scheduled_at ASC`,
		userID, start, end,
	); err != nil {
		return nil, err
	}
	jobs := make([]*publish.Job, len(rows))
	for i, row := range rows {
		jobs[i] = row.toEntity()
	}
	return jobs, nil
}

func (r *PublishJobRepo) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM publish_jobs WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apperr.ErrNotFound
	}
	return nil
}
