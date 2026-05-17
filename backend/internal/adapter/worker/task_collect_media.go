package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/video"
	"mmo/internal/infrastructure/ffmpeg"
	"mmo/internal/infrastructure/queue"
	"mmo/internal/infrastructure/storage"
	"mmo/internal/integration/pexels"
	"mmo/internal/integration/pixabay"
	"mmo/pkg/logger"
	"go.uber.org/zap"
)


type MediaCollectHandler struct {
	planRepo    *repository.ContentPlanRepo
	videoRepo   *repository.VideoJobRepo
	pexels      *pexels.Client
	pixabay     *pixabay.Client
	r2          *storage.R2Client
	queueClient *asynq.Client
	httpClient  *http.Client
	assembler   *ffmpeg.Assembler
}

type mediaAssetJSON struct {
	Type     string  `json:"type"`
	URL      string  `json:"url"`
	R2Key    string  `json:"r2_key"`
	Duration float64 `json:"duration"`
}

func NewMediaCollectHandler(
	planRepo *repository.ContentPlanRepo,
	videoRepo *repository.VideoJobRepo,
	pexelsClient *pexels.Client,
	pixabayClient *pixabay.Client,
	r2 *storage.R2Client,
	queueClient *asynq.Client,
	assembler *ffmpeg.Assembler,
	httpTimeout time.Duration,
) *MediaCollectHandler {
	return &MediaCollectHandler{
		planRepo:    planRepo,
		videoRepo:   videoRepo,
		pexels:      pexelsClient,
		pixabay:     pixabayClient,
		r2:          r2,
		queueClient: queueClient,
		httpClient:  &http.Client{Timeout: httpTimeout},
		assembler:   assembler,
	}
}

func (h *MediaCollectHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var p struct {
		ContentPlanID string `json:"content_plan_id"`
	}
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		return err
	}

	planID, err := uuid.Parse(p.ContentPlanID)
	if err != nil {
		return err
	}

	plan, err := h.planRepo.GetByID(ctx, planID)
	if err != nil {
		return fmt.Errorf("content plan not found: %w", err)
	}

	// Create video job
	jobID := uuid.New()
	job := &video.Job{
		ID:            jobID,
		ContentPlanID: planID,
		Status:        video.JobStatusMediaCollecting,
	}
	if err := h.videoRepo.Create(ctx, job); err != nil {
		return fmt.Errorf("create video job: %w", err)
	}

	// Update content plan status
	_ = h.planRepo.UpdateStatus(ctx, planID, "video_queued")

	logger.Info("collecting media", zap.String("job_id", jobID.String()), zap.String("title", plan.Title))

	// Extract keywords for media search
	keywords := extractKeywordsFromTitle(plan.Title)
	query := keywords[0]
	if len(keywords) > 1 {
		query = strings.Join(keywords[:2], " ")
	}

	tmpDir, err := h.assembler.TempDir(jobID.String())
	if err != nil {
		return err
	}

	var assets []mediaAssetJSON

	// Try Pexels videos first
	videos, err := h.pexels.SearchVideos(ctx, query, 3)
	if err != nil {
		logger.Warn("pexels failed, trying pixabay", zap.Error(err))
	}
	for i, v := range videos {
		if len(assets) >= 3 {
			break
		}
		videoURL := pexels.BestVideoURL(v)
		if videoURL == "" {
			continue
		}
		localPath := filepath.Join(tmpDir, fmt.Sprintf("clip_%d.mp4", i))
		if err := downloadFile(ctx, h.httpClient, videoURL, localPath); err != nil {
			logger.Warn("download clip failed", zap.Error(err))
			continue
		}
		r2Key := fmt.Sprintf("media/videos/%s/clip_%d.mp4", jobID, i)
		assets = append(assets, mediaAssetJSON{
			Type:     "video",
			URL:      videoURL,
			R2Key:    r2Key,
			Duration: float64(v.Duration),
		})
	}

	// Fallback to Pixabay if not enough assets
	if len(assets) < 3 {
		pbVideos, err := h.pixabay.SearchVideos(ctx, query, 3)
		if err != nil {
			logger.Warn("pixabay failed", zap.Error(err))
		}
		for i, v := range pbVideos {
			if len(assets) >= 3 {
				break
			}
			videoURL := v.Videos.Large.URL
			if videoURL == "" {
				videoURL = v.Videos.Medium.URL
			}
			if videoURL == "" {
				continue
			}
			localPath := filepath.Join(tmpDir, fmt.Sprintf("pbclip_%d.mp4", i))
			if err := downloadFile(ctx, h.httpClient, videoURL, localPath); err != nil {
				continue
			}
			r2Key := fmt.Sprintf("media/videos/%s/pbclip_%d.mp4", jobID, i)
			assets = append(assets, mediaAssetJSON{
				Type:     "video",
				URL:      videoURL,
				R2Key:    r2Key,
				Duration: float64(v.Duration),
			})
		}
	}

	// Fall back to Pexels photos if still no video
	if len(assets) == 0 {
		photos, _ := h.pexels.SearchPhotos(ctx, query, 5)
		for i, ph := range photos {
			if len(assets) >= 5 {
				break
			}
			imgURL := ph.Src.Large
			localPath := filepath.Join(tmpDir, fmt.Sprintf("img_%d.jpg", i))
			if err := downloadFile(ctx, h.httpClient, imgURL, localPath); err != nil {
				continue
			}
			r2Key := fmt.Sprintf("media/images/%s/img_%d.jpg", jobID, i)
			assets = append(assets, mediaAssetJSON{
				Type:     "image",
				URL:      imgURL,
				R2Key:    r2Key,
				Duration: 4,
			})
		}
	}

	assetsJSON, _ := json.Marshal(assets)
	job.MediaAssets = assetsJSON
	job.Status = video.JobStatusTTSGenerating
	if err := h.videoRepo.Update(ctx, job); err != nil {
		return err
	}

	// Chain to TTS task
	ttsPayload, _ := json.Marshal(map[string]string{
		"job_id":  jobID.String(),
		"plan_id": planID.String(),
		"script":  plan.Script,
	})
	ttsTask := asynq.NewTask(queue.TaskGenerateTTS, ttsPayload, asynq.Queue(queue.QueueDefault))
	if _, err := h.queueClient.EnqueueContext(ctx, ttsTask); err != nil {
		logger.Error("failed to enqueue TTS task", zap.Error(err))
		return err
	}

	logger.Info("media collected, TTS queued",
		zap.String("job_id", jobID.String()), zap.Int("assets", len(assets)))
	return nil
}

func downloadFile(ctx context.Context, client *http.Client, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := openForWrite(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func extractKeywordsFromTitle(title string) []string {
	words := strings.Fields(title)
	var kw []string
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "in": true, "on": true, "at": true,
		"to": true, "of": true, "and": true, "or": true, "but": true,
		"for": true, "with": true, "this": true, "that": true, "it": true,
	}
	seen := map[string]bool{}
	for _, w := range words {
		w = strings.ToLower(strings.Trim(w, ".,!?\"'()[]"))
		if len(w) > 3 && !stopWords[w] && !seen[w] {
			seen[w] = true
			kw = append(kw, w)
		}
	}
	if len(kw) == 0 {
		return []string{"nature", "lifestyle"}
	}
	if len(kw) > 5 {
		return kw[:5]
	}
	return kw
}
