package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"mmo/internal/domain/publish"
)

type AnalyticsRepo struct{ db *sqlx.DB }

func NewAnalyticsRepo(db *sqlx.DB) *AnalyticsRepo { return &AnalyticsRepo{db: db} }

func (r *AnalyticsRepo) Upsert(ctx context.Context, a *publish.Analytics) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE post_analytics SET
			synced_at=$1, views=$2, likes=$3, comments=$4, shares=$5,
			saves=$6, reach=$7, impressions=$8, play_time_seconds=$9, raw_data=$10
		WHERE publish_job_id=$11`,
		a.SyncedAt, a.Views, a.Likes, a.Comments, a.Shares,
		a.Saves, a.Reach, a.Impressions, a.PlayTimeSeconds, a.RawData,
		a.PublishJobID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		_, err = r.db.ExecContext(ctx, `
			INSERT INTO post_analytics
				(id, publish_job_id, channel_id, platform, synced_at,
				 views, likes, comments, shares, saves, reach, impressions, play_time_seconds, raw_data)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
			a.ID, a.PublishJobID, a.ChannelID, a.Platform, a.SyncedAt,
			a.Views, a.Likes, a.Comments, a.Shares, a.Saves, a.Reach,
			a.Impressions, a.PlayTimeSeconds, a.RawData,
		)
	}
	return err
}

func (r *AnalyticsRepo) GetByPublishJob(ctx context.Context, publishJobID uuid.UUID) (*publish.Analytics, error) {
	var a publish.Analytics
	err := r.db.GetContext(ctx, &a, `SELECT * FROM post_analytics WHERE publish_job_id=$1`, publishJobID)
	return &a, err
}

type OverviewStats struct {
	TotalViews    int64 `db:"total_views"    json:"total_views"`
	TotalLikes    int64 `db:"total_likes"    json:"total_likes"`
	TotalComments int64 `db:"total_comments" json:"total_comments"`
	TotalShares   int64 `db:"total_shares"   json:"total_shares"`
	PostCount     int   `db:"post_count"     json:"post_count"`
}

func (r *AnalyticsRepo) Overview(ctx context.Context, userID uuid.UUID, since time.Time) (*OverviewStats, error) {
	var s OverviewStats
	err := r.db.GetContext(ctx, &s, `
		SELECT
			COALESCE(SUM(a.views),0)    AS total_views,
			COALESCE(SUM(a.likes),0)    AS total_likes,
			COALESCE(SUM(a.comments),0) AS total_comments,
			COALESCE(SUM(a.shares),0)   AS total_shares,
			COUNT(*)                     AS post_count
		FROM post_analytics a
		JOIN publish_jobs pj ON pj.id = a.publish_job_id
		JOIN channels c ON c.id = pj.channel_id
		WHERE c.user_id = $1 AND a.synced_at >= $2`,
		userID, since,
	)
	return &s, err
}

type PostAnalyticsSummary struct {
	PublishJobID uuid.UUID `db:"publish_job_id" json:"publish_job_id"`
	Platform     string    `db:"platform"       json:"platform"`
	SyncedAt     time.Time `db:"synced_at"      json:"synced_at"`
	Views        int64     `db:"views"          json:"views"`
	Likes        int64     `db:"likes"          json:"likes"`
	Comments     int64     `db:"comments"       json:"comments"`
	Shares       int64     `db:"shares"         json:"shares"`
}

type TimeseriesPoint struct {
	Date     string `db:"day"         json:"date"`
	Views    int64  `db:"total_views" json:"views"`
	Likes    int64  `db:"total_likes" json:"likes"`
	Comments int64  `db:"total_comments" json:"comments"`
}

func (r *AnalyticsRepo) Timeseries(ctx context.Context, userID uuid.UUID, since time.Time) ([]TimeseriesPoint, error) {
	var rows []TimeseriesPoint
	err := r.db.SelectContext(ctx, &rows, `
		SELECT
			TO_CHAR(a.synced_at::date, 'YYYY-MM-DD') AS day,
			COALESCE(SUM(a.views),0)    AS total_views,
			COALESCE(SUM(a.likes),0)    AS total_likes,
			COALESCE(SUM(a.comments),0) AS total_comments
		FROM post_analytics a
		JOIN publish_jobs pj ON pj.id = a.publish_job_id
		JOIN channels c ON c.id = pj.channel_id
		WHERE c.user_id = $1 AND a.synced_at >= $2
		GROUP BY a.synced_at::date
		ORDER BY a.synced_at::date ASC`, userID, since)
	return rows, err
}

func (r *AnalyticsRepo) ListPosts(ctx context.Context, userID uuid.UUID, page, perPage int) ([]PostAnalyticsSummary, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM post_analytics a
		JOIN publish_jobs pj ON pj.id=a.publish_job_id
		JOIN channels c ON c.id=pj.channel_id
		WHERE c.user_id=$1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	rows := make([]PostAnalyticsSummary, 0)
	err := r.db.SelectContext(ctx, &rows, `
		SELECT a.publish_job_id, a.platform, a.synced_at, a.views, a.likes, a.comments, a.shares
		FROM post_analytics a
		JOIN publish_jobs pj ON pj.id=a.publish_job_id
		JOIN channels c ON c.id=pj.channel_id
		WHERE c.user_id=$1
		ORDER BY a.synced_at DESC
		LIMIT $2 OFFSET $3`, userID, perPage, offset)
	return rows, total, err
}
