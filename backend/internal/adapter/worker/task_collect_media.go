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
	"go.uber.org/zap"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/video"
	"mmo/internal/infrastructure/ffmpeg"
	"mmo/internal/infrastructure/queue"
	"mmo/internal/infrastructure/storage"
	"mmo/internal/integration/pexels"
	"mmo/internal/integration/pixabay"
	"mmo/pkg/logger"
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
	maxClips    int
}

type mediaAssetJSON struct {
	Type     string  `json:"type"`
	URL      string  `json:"url"`
	R2Key    string  `json:"r2_key"`
	Duration float64 `json:"duration"`
	Scene    int     `json:"scene"` // index of the scene this clip belongs to (-1 if none)
}

// sceneMeta mirrors the scene fields stored in ContentPlan.ScriptMetadata.
type sceneMeta struct {
	Narration   string `json:"narration"`
	VisualQuery string `json:"visual_query"`
	Keyword     string `json:"keyword"`
}

func parseScenes(scriptMetadata []byte) []sceneMeta {
	var meta struct {
		Scenes []sceneMeta `json:"scenes"`
	}
	_ = json.Unmarshal(scriptMetadata, &meta)
	return meta.Scenes
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
	maxClips int,
) *MediaCollectHandler {
	if maxClips <= 0 {
		maxClips = 15
	}
	return &MediaCollectHandler{
		planRepo:    planRepo,
		videoRepo:   videoRepo,
		pexels:      pexelsClient,
		pixabay:     pixabayClient,
		r2:          r2,
		queueClient: queueClient,
		httpClient:  &http.Client{Timeout: httpTimeout},
		assembler:   assembler,
		maxClips:    maxClips,
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
		MediaAssets:   []byte("[]"),
	}
	if err := h.videoRepo.Create(ctx, job); err != nil {
		return fmt.Errorf("create video job: %w", err)
	}

	// Update content plan status
	_ = h.planRepo.UpdateStatus(ctx, planID, "video_queued")

	logger.Info("collecting media", zap.String("job_id", jobID.String()), zap.String("title", plan.Title))

	tmpDir, err := h.assembler.TempDir(jobID.String())
	if err != nil {
		return err
	}

	scenes := parseScenes(plan.ScriptMetadata)
	var assets []mediaAssetJSON
	clipIdx := 0
	fallbackQuery := plan.Title

	if len(scenes) > 0 {
		// Per-scene collection: the footage for each scene is searched with that
		// scene's own visual_query, so visuals track the narration.
		const clipsPerScene = 4
		if q := strings.TrimSpace(scenes[0].VisualQuery); q != "" {
			fallbackQuery = q
		}
		for i, sc := range scenes {
			q := strings.TrimSpace(sc.VisualQuery)
			if q == "" {
				q = strings.TrimSpace(sc.Keyword)
			}
			if q == "" {
				continue
			}
			got, next := h.downloadClips(ctx, tmpDir, jobID.String(), q, clipsPerScene, clipIdx)
			clipIdx = next
			for j := range got {
				got[j].Scene = i
			}
			assets = append(assets, got...)
			logger.Info("scene media collected", zap.Int("scene", i), zap.String("query", q), zap.Int("clips", len(got)))
		}
	} else {
		// Fallback: generic keyword/title queries, no scene tagging.
		queries := visualKeywordQueries(plan.ScriptMetadata)
		if len(queries) == 0 {
			queries = buildSearchQueries(plan.Title)
		}
		fallbackQuery = queries[0]
		target := h.maxClips
		perQuery := target/len(queries) + 1
		for _, q := range queries {
			if len(assets) >= target {
				break
			}
			got, next := h.downloadClips(ctx, tmpDir, jobID.String(), q, perQuery, clipIdx)
			clipIdx = next
			for j := range got {
				got[j].Scene = -1
			}
			assets = append(assets, got...)
		}
	}

	// Last-resort: photos for a slideshow if no video was collected at all.
	if len(assets) == 0 {
		photos, _ := h.pexels.SearchPhotos(ctx, fallbackQuery, h.maxClips)
		for i, ph := range photos {
			imgURL := ph.Src.Large
			localPath := filepath.Join(tmpDir, fmt.Sprintf("img_%d.jpg", i))
			if err := downloadFile(ctx, h.httpClient, imgURL, localPath); err != nil {
				continue
			}
			assets = append(assets, mediaAssetJSON{
				Type:     "image",
				URL:      imgURL,
				R2Key:    fmt.Sprintf("media/images/%s/img_%d.jpg", jobID, i),
				Duration: 4,
				Scene:    -1,
			})
		}
	}

	assetsJSON, _ := json.Marshal(assets)
	job.MediaAssets = assetsJSON
	job.Status = video.JobStatusTTSGenerating
	if err := h.videoRepo.Update(ctx, job); err != nil {
		return err
	}

	// Chain to TTS task. Pass per-scene narrations so TTS can produce exact
	// per-scene durations for scene-aligned assembly.
	narrations := make([]string, 0, len(scenes))
	for _, s := range scenes {
		narrations = append(narrations, s.Narration)
	}
	ttsPayload, _ := json.Marshal(map[string]any{
		"job_id":           jobID.String(),
		"plan_id":          planID.String(),
		"script":           plan.Script,
		"voice":            plan.Voice,
		"scene_narrations": narrations,
	})
	ttsTask := asynq.NewTask(queue.TaskGenerateTTS, ttsPayload, asynq.Queue(queue.QueueVideo))
	if _, err := h.queueClient.EnqueueContext(ctx, ttsTask); err != nil {
		logger.Error("failed to enqueue TTS task", zap.Error(err))
		return err
	}

	logger.Info("media collected, TTS queued",
		zap.String("job_id", jobID.String()), zap.Int("assets", len(assets)))
	return nil
}

// downloadClips searches Pexels (then Pixabay as supplement) for one query and
// downloads up to `want` video clips into tmpDir, numbered from startIdx. Returns
// the collected assets (Scene unset) and the next free index.
func (h *MediaCollectHandler) downloadClips(ctx context.Context, tmpDir, jobID, query string, want, startIdx int) ([]mediaAssetJSON, int) {
	var out []mediaAssetJSON
	idx := startIdx
	add := func(url string, dur float64) {
		localPath := filepath.Join(tmpDir, fmt.Sprintf("clip_%d.mp4", idx))
		if err := downloadFile(ctx, h.httpClient, url, localPath); err != nil {
			logger.Warn("download clip failed", zap.String("url", url), zap.Error(err))
			return
		}
		out = append(out, mediaAssetJSON{
			Type:     "video",
			URL:      url,
			R2Key:    fmt.Sprintf("media/videos/%s/clip_%d.mp4", jobID, idx),
			Duration: dur,
		})
		idx++
	}

	if vids, err := h.pexels.SearchVideos(ctx, query, want); err != nil {
		logger.Warn("pexels failed", zap.String("query", query), zap.Error(err))
	} else {
		for _, v := range vids {
			if len(out) >= want {
				break
			}
			if u := pexels.BestVideoURL(v); u != "" {
				add(u, float64(v.Duration))
			}
		}
	}

	if len(out) < want {
		if pb, err := h.pixabay.SearchVideos(ctx, query, want); err != nil {
			logger.Warn("pixabay failed", zap.String("query", query), zap.Error(err))
		} else {
			for _, v := range pb {
				if len(out) >= want {
					break
				}
				u := v.Videos.Large.URL
				if u == "" {
					u = v.Videos.Medium.URL
				}
				if u != "" {
					add(u, float64(v.Duration))
				}
			}
		}
	}
	return out, idx
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

// visualKeywordQueries pulls the AI-provided English `visual_keywords` out of a
// plan's ScriptMetadata JSON. Returns deduped, non-empty terms (max 8).
func visualKeywordQueries(scriptMetadata []byte) []string {
	var meta struct {
		VisualKeywords []string `json:"visual_keywords"`
	}
	if err := json.Unmarshal(scriptMetadata, &meta); err != nil {
		return nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(meta.VisualKeywords))
	for _, k := range meta.VisualKeywords {
		k = strings.TrimSpace(k)
		if k == "" || seen[strings.ToLower(k)] {
			continue
		}
		seen[strings.ToLower(k)] = true
		out = append(out, k)
		if len(out) >= 8 {
			break
		}
	}
	return out
}

// buildSearchQueries returns 3-5 distinct search terms derived from the title,
// plus generic fallbacks. Diverse queries → diverse b-roll.
func buildSearchQueries(title string) []string {
	kw := extractKeywordsFromTitle(title)
	var qs []string
	if len(kw) >= 2 {
		qs = append(qs, strings.Join(kw[:2], " "))
	}
	for _, w := range kw {
		if len(qs) >= 5 {
			break
		}
		qs = append(qs, w)
	}
	// Always include a generic visual fallback so long videos have b-roll variety.
	qs = append(qs, "technology", "people working", "city lights")
	seen := map[string]bool{}
	out := qs[:0]
	for _, q := range qs {
		q = strings.TrimSpace(q)
		if q == "" || seen[q] {
			continue
		}
		seen[q] = true
		out = append(out, q)
	}
	return out
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
