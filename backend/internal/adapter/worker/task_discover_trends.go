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

	trends := make([]*content.TrendTopic, 0, 64)

	gTrends, err := h.googleClient.FetchDailyTrends(ctx, geo)
	if err != nil {
		logger.Warn("google trends fetch failed", zap.Error(err))
	} else {
		for _, t := range gTrends {
			trends = append(trends, h.buildTrend(payload.UserID, "google_trends", t.Title, t.Description,
				t.Keywords, t.Score, t.SourceURL, t))
		}
	}

	ytVideos, err := h.youtubeClient.FetchTrending(ctx, geo, "")
	if err != nil {
		logger.Warn("youtube fetch failed", zap.Error(err))
	} else {
		for _, v := range ytVideos {
			trends = append(trends, h.buildTrend(payload.UserID, "youtube", v.Title, v.Description,
				v.Keywords, 0, v.SourceURL, v))
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
			trends = append(trends, h.buildTrend(payload.UserID, "reddit", p.Title, p.Body,
				p.Keywords, float64(p.Score), p.URL, p))
		}
	}

	// Vietnamese news source: VnExpress RSS (only when geo is VN)
	if geo == "VN" {
		vnItems, err := h.vnexpressClient.FetchTrending(ctx, 5)
		if err != nil {
			logger.Warn("vnexpress fetch failed", zap.Error(err))
		} else {
			for _, item := range vnItems {
				trends = append(trends, h.buildTrend(payload.UserID, "vnexpress", item.Title, item.Description,
					item.Keywords, 0, item.SourceURL, item))
			}
		}
	}

	// Single batched insert; duplicates are skipped via ON CONFLICT DO NOTHING.
	if err := h.trendRepo.CreateBatch(ctx, trends); err != nil {
		logger.Error("batch insert trends failed", zap.Error(err))
		return err
	}

	logger.Info("trend discovery complete", zap.Int("fetched", len(trends)), zap.String("geo", geo))
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

// buildTrend constructs a TrendTopic entity (no DB write). De-duplication is
// handled by CreateBatch's ON CONFLICT, so no per-row existence check is needed.
func (h *TrendDiscoveryHandler) buildTrend(
	userIDStr, source, title, desc string,
	keywords []string,
	score float64,
	sourceURL string,
	rawData any,
) *content.TrendTopic {
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
		if uid, err := uuid.Parse(userIDStr); err == nil {
			t.UserID = &uid
		}
	}
	return t
}
