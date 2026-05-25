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
	"mmo/internal/integration/vnexpress"
	"mmo/internal/integration/youtube"
	"mmo/pkg/config"
	"mmo/pkg/logger"
	"go.uber.org/zap"
)

type TrendDiscoveryHandler struct {
	trendRepo       *repository.TrendRepo
	googleClient    *googletrends.Client
	youtubeClient   *youtube.Client
	redditClient    *reddit.Client
	vnexpressClient *vnexpress.Client
	cfg             *config.Config
}

func NewTrendDiscoveryHandler(
	trendRepo *repository.TrendRepo,
	cfg *config.Config,
	googleClient *googletrends.Client,
	youtubeClient *youtube.Client,
	redditClient *reddit.Client,
) *TrendDiscoveryHandler {
	return &TrendDiscoveryHandler{
		trendRepo:       trendRepo,
		googleClient:    googleClient,
		youtubeClient:   youtubeClient,
		redditClient:    redditClient,
		vnexpressClient: vnexpress.New(cfg.MediaCollect.HTTPTimeout),
		cfg:             cfg,
	}
}

func (h *TrendDiscoveryHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload struct {
		UserID string `json:"user_id"`
	}
	_ = json.Unmarshal(task.Payload(), &payload)

	geo := h.cfg.Content.Geo
	if geo == "" {
		geo = "US"
	}

	total := 0

	gTrends, err := h.googleClient.FetchDailyTrends(ctx, geo)
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

	ytVideos, err := h.youtubeClient.FetchTrending(ctx, geo, "")
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

	subreddits := h.getSubreddits()
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

	// Vietnamese news source: VnExpress RSS (only when geo is VN)
	if geo == "VN" {
		vnItems, err := h.vnexpressClient.FetchTrending(ctx, 5)
		if err != nil {
			logger.Warn("vnexpress fetch failed", zap.Error(err))
		} else {
			for _, item := range vnItems {
				if err := h.saveTrend(ctx, payload.UserID, "vnexpress", item.Title, item.Description,
					item.Keywords, 0, item.SourceURL, item); err == nil {
					total++
				}
			}
		}
	}

	logger.Info("trend discovery complete", zap.Int("new_trends", total), zap.String("geo", geo))
	return nil
}

func (h *TrendDiscoveryHandler) getSubreddits() []string {
	if h.cfg.Content.Language == "vi" {
		return []string{
			"VietNam", "vietnam", "viettech",
			"kinh_doanh", "hoclaptrinh",
			"marketing", "entrepreneur", "technology",
		}
	}
	return []string{"marketing", "entrepreneur", "personalfinance", "fitness", "technology"}
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
