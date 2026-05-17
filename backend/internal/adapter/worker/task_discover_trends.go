package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/content"
	"mmo/internal/integration/googletrends"
	"mmo/internal/integration/reddit"
	"mmo/internal/integration/youtube"
	"mmo/pkg/config"
	"mmo/pkg/logger"
	"go.uber.org/zap"
)

type TrendDiscoveryHandler struct {
	trendRepo     *repository.TrendRepo
	googleClient  *googletrends.Client
	youtubeClient *youtube.Client
	redditClient  *reddit.Client
	cfg           *config.Config
}

func NewTrendDiscoveryHandler(
	trendRepo *repository.TrendRepo,
	cfg *config.Config,
	googleClient *googletrends.Client,
	youtubeClient *youtube.Client,
	redditClient *reddit.Client,
) *TrendDiscoveryHandler {
	return &TrendDiscoveryHandler{
		trendRepo:     trendRepo,
		googleClient:  googleClient,
		youtubeClient: youtubeClient,
		redditClient:  redditClient,
		cfg:           cfg,
	}
}

func (h *TrendDiscoveryHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload struct {
		UserID string `json:"user_id"`
	}
	_ = json.Unmarshal(task.Payload(), &payload)

	total := 0

	gTrends, err := h.googleClient.FetchDailyTrends(ctx, "US")
	if err != nil {
		logger.Warn("google trends fetch failed", zap.Error(err))
	} else {
		for _, t := range gTrends {
			if err := h.saveTrend(ctx, payload.UserID, "google_trends", t.Title, t.Description,
				t.Keywords, t.Score, t.SourceURL, t); err == nil {
				total++
			}
		}
	}

	ytVideos, err := h.youtubeClient.FetchTrending(ctx, "US", "")
	if err != nil {
		logger.Warn("youtube fetch failed", zap.Error(err))
	} else {
		for _, v := range ytVideos {
			if err := h.saveTrend(ctx, payload.UserID, "youtube", v.Title, v.Description,
				v.Keywords, 0, v.SourceURL, v); err == nil {
				total++
			}
		}
	}

	subreddits := []string{"marketing", "entrepreneur", "personalfinance", "fitness", "technology"}
	for _, sub := range subreddits {
		posts, err := h.redditClient.FetchTopPosts(ctx, sub, "day", 5)
		if err != nil {
			logger.Warn("reddit fetch failed", zap.String("subreddit", sub), zap.Error(err))
			continue
		}
		for _, p := range posts {
			if err := h.saveTrend(ctx, payload.UserID, "reddit", p.Title, p.Body,
				p.Keywords, float64(p.Score), p.URL, p); err == nil {
				total++
			}
		}
	}

	logger.Info("trend discovery complete", zap.Int("new_trends", total))
	return nil
}

func (h *TrendDiscoveryHandler) saveTrend(
	ctx context.Context,
	userIDStr, source, title, desc string,
	keywords []string,
	score float64,
	sourceURL string,
	rawData any,
) error {
	exists, _ := h.trendRepo.ExistsBySourceAndTitle(ctx, source, title)
	if exists {
		return nil
	}

	raw, _ := json.Marshal(rawData)
	t := &content.TrendTopic{
		ID:            uuid.New(),
		Source:        source,
		Title:         title,
		Description:   desc,
		Keywords:      keywords,
		TrendingScore: score,
		SourceURL:     sourceURL,
		RawData:       raw,
		Status:        "new",
		DiscoveredAt:  time.Now(),
		CreatedAt:     time.Now(),
	}

	if userIDStr != "" {
		uid, err := uuid.Parse(userIDStr)
		if err == nil {
			t.UserID = &uid
		}
	}

	return h.trendRepo.Create(ctx, t)
}
