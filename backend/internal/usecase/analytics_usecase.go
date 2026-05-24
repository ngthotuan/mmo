package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"mmo/internal/adapter/repository"
)

type AnalyticsUsecase struct {
	repo *repository.AnalyticsRepo
}

func NewAnalyticsUsecase(repo *repository.AnalyticsRepo) *AnalyticsUsecase {
	return &AnalyticsUsecase{repo: repo}
}

func (u *AnalyticsUsecase) Overview(ctx context.Context, userID uuid.UUID, days int) (*repository.OverviewStats, error) {
	since := time.Now().AddDate(0, 0, -days)
	return u.repo.Overview(ctx, userID, since)
}

func (u *AnalyticsUsecase) ListPosts(ctx context.Context, userID uuid.UUID, page, perPage int) ([]repository.PostAnalyticsSummary, int, error) {
	return u.repo.ListPosts(ctx, userID, page, perPage)
}

func (u *AnalyticsUsecase) Timeseries(ctx context.Context, userID uuid.UUID, days int) ([]repository.TimeseriesPoint, error) {
	since := time.Now().AddDate(0, 0, -days)
	return u.repo.Timeseries(ctx, userID, since)
}
