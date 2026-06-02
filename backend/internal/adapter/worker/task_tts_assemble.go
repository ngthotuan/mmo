package worker

import (
	"context"
	"encoding/json"
	"fmt"
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
	"mmo/internal/integration/edgetts"
	"mmo/pkg/logger"
)

// ─── TTS Handler ─────────────────────────────────────────────────────────────

type TTSHandler struct {
	videoRepo   *repository.VideoJobRepo
	tts         *edgetts.Client
	r2          *storage.R2Client
	queueClient *asynq.Client
	assembler   *ffmpeg.Assembler
}

func NewTTSHandler(
	videoRepo *repository.VideoJobRepo,
	ttsClient *edgetts.Client,
	r2 *storage.R2Client,
	queueClient *asynq.Client,
	assembler *ffmpeg.Assembler,
) *TTSHandler {
	return &TTSHandler{
		videoRepo:   videoRepo,
		tts:         ttsClient,
		r2:          r2,
		queueClient: queueClient,
		assembler:   assembler,
	}
}

func (h *TTSHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var p struct {
		JobID           string   `json:"job_id"`
		PlanID          string   `json:"plan_id"`
		Script          string   `json:"script"`
		Voice           string   `json:"voice"`
		SceneNarrations []string `json:"scene_narrations"`
	}
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		return err
	}

	jobID, err := uuid.Parse(p.JobID)
	if err != nil {
		return err
	}

	job, err := h.videoRepo.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("video job not found: %w", err)
	}

	tmpDir, err := h.assembler.TempDir(jobID.String())
	if err != nil {
		return err
	}

	logger.Info("generating TTS", zap.String("job_id", p.JobID), zap.Int("scenes", len(p.SceneNarrations)))

	voice := p.Voice
	if voice == "" {
		voice = h.tts.DefaultVoice()
	}

	var audioPath, srtPath string
	var sceneDurations []float64

	if len(p.SceneNarrations) > 0 {
		audioPath, srtPath, sceneDurations, err = h.generateSceneTTS(ctx, p.SceneNarrations, voice, tmpDir)
	}
	if len(p.SceneNarrations) == 0 || err != nil {
		// Fallback: single TTS of the whole script.
		if err != nil {
			logger.Warn("per-scene TTS failed, falling back to single", zap.Error(err))
		}
		result, gerr := h.tts.Generate(ctx, p.Script, voice, tmpDir)
		if gerr != nil {
			_ = h.videoRepo.UpdateStatus(ctx, jobID, video.JobStatusFailed, "TTS generation failed: "+gerr.Error())
			return gerr
		}
		audioPath = result.AudioPath
		if s, cerr := edgetts.VTTToSRT(result.SubtitlePath); cerr == nil {
			srtPath = s
		}
		sceneDurations = nil
	}

	job.TTSAudioKey = fmt.Sprintf("media/tts/%s/audio.mp3", jobID)
	if srtPath != "" {
		job.SubtitleKey = fmt.Sprintf("media/tts/%s/subtitle.srt", jobID)
	}
	job.Status = video.JobStatusAssembling
	if err := h.videoRepo.Update(ctx, job); err != nil {
		return err
	}

	// Chain to video assembly
	payload, _ := json.Marshal(map[string]any{
		"job_id":          p.JobID,
		"audio_path":      audioPath,
		"subtitle_path":   srtPath,
		"scene_durations": sceneDurations,
	})
	assembleTask := asynq.NewTask(queue.TaskAssembleVideo, payload, asynq.Queue(queue.QueueVideo))
	if _, err := h.queueClient.EnqueueContext(ctx, assembleTask); err != nil {
		return fmt.Errorf("enqueue assemble: %w", err)
	}

	logger.Info("TTS done, assembly queued", zap.String("job_id", p.JobID))
	return nil
}

// generateSceneTTS synthesizes each scene's narration separately, concatenates
// the audio into one narration track, and merges the per-scene subtitles onto a
// single timeline. Returns the full audio path, merged SRT path, and the exact
// per-scene durations (indexed by scene) used to align b-roll during assembly.
func (h *TTSHandler) generateSceneTTS(ctx context.Context, narrations []string, voice, tmpDir string) (string, string, []float64, error) {
	n := len(narrations)
	audios := make([]string, n)
	srts := make([]string, n)
	durations := make([]float64, n)

	for i, text := range narrations {
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		res, err := h.tts.Generate(ctx, text, voice, tmpDir)
		if err != nil {
			return "", "", nil, fmt.Errorf("scene %d tts: %w", i, err)
		}
		if s, cerr := edgetts.VTTToSRT(res.SubtitlePath); cerr == nil {
			srts[i] = s
		}
		audios[i] = res.AudioPath
		durations[i] = h.assembler.ProbeDuration(ctx, res.AudioPath)
	}

	nonEmpty := make([]string, 0, n)
	for _, a := range audios {
		if a != "" {
			nonEmpty = append(nonEmpty, a)
		}
	}
	if len(nonEmpty) == 0 {
		return "", "", nil, fmt.Errorf("no scene audio produced")
	}

	fullAudio, err := h.assembler.ConcatAudio(ctx, nonEmpty, tmpDir)
	if err != nil {
		return "", "", nil, err
	}

	offsets := make([]float64, n)
	cum := 0.0
	for i := 0; i < n; i++ {
		offsets[i] = cum
		cum += durations[i]
	}
	fullSRT := filepath.Join(tmpDir, fmt.Sprintf("subtitle_full_%d.srt", time.Now().UnixNano()))
	if err := edgetts.MergeSRTFiles(srts, offsets, fullSRT); err != nil {
		fullSRT = ""
	}
	return fullAudio, fullSRT, durations, nil
}

// ─── Video Assembly Handler ───────────────────────────────────────────────────

type VideoAssemblyHandler struct {
	videoRepo   *repository.VideoJobRepo
	assembler   *ffmpeg.Assembler
	r2          *storage.R2Client
	queueClient *asynq.Client
}

func NewVideoAssemblyHandler(
	videoRepo *repository.VideoJobRepo,
	assembler *ffmpeg.Assembler,
	r2 *storage.R2Client,
	queueClient *asynq.Client,
) *VideoAssemblyHandler {
	return &VideoAssemblyHandler{
		videoRepo:   videoRepo,
		assembler:   assembler,
		r2:          r2,
		queueClient: queueClient,
	}
}

func (h *VideoAssemblyHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var p struct {
		JobID          string    `json:"job_id"`
		AudioPath      string    `json:"audio_path"`
		SubtitlePath   string    `json:"subtitle_path"`
		SceneDurations []float64 `json:"scene_durations"`
	}
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		return err
	}

	jobID, err := uuid.Parse(p.JobID)
	if err != nil {
		return err
	}

	job, err := h.videoRepo.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("video job not found: %w", err)
	}

	tmpDir, err := h.assembler.TempDir(jobID.String())
	if err != nil {
		return err
	}

	// Parse media assets from job
	var assets []mediaAssetJSON
	_ = json.Unmarshal(job.MediaAssets, &assets)

	logger.Info("assembling video", zap.String("job_id", p.JobID), zap.Int("assets", len(assets)))

	now := time.Now()
	job.StartedAt = &now

	ffAssets := make([]ffmpeg.MediaAsset, 0, len(assets))
	for _, a := range assets {
		localPath := filepath.Join(tmpDir, filepath.Base(a.R2Key))
		ffAssets = append(ffAssets, ffmpeg.MediaAsset{
			Path:     localPath,
			Type:     a.Type,
			Duration: a.Duration,
		})
	}

	hasVideo := len(ffAssets) > 0 && ffAssets[0].Type == "video"

	var result *ffmpeg.AssembleResult
	switch {
	case len(p.SceneDurations) > 0 && hasVideo:
		// Scene-aligned: group each scene's clips and play them during that
		// scene's narration so visuals track the content.
		sceneClips := make([][]ffmpeg.MediaAsset, len(p.SceneDurations))
		for i, a := range assets {
			if a.Type != "video" || a.Scene < 0 || a.Scene >= len(sceneClips) {
				continue
			}
			sceneClips[a.Scene] = append(sceneClips[a.Scene], ffAssets[i])
		}
		logger.Info("scene-aligned assembly", zap.String("job_id", p.JobID), zap.Int("scenes", len(p.SceneDurations)))
		result, err = h.assembler.AssembleScenes(ctx, sceneClips, p.SceneDurations, p.AudioPath, p.SubtitlePath, tmpDir)
	case hasVideo:
		result, err = h.assembler.AssembleBRoll(ctx, ffAssets, p.AudioPath, p.SubtitlePath, tmpDir)
	case len(ffAssets) > 0:
		result, err = h.assembler.AssembleSlideshow(ctx, ffAssets, p.AudioPath, p.SubtitlePath, tmpDir)
	default:
		// No media assets — use text-on-color fallback
		result, err = h.assembler.AssembleTextOnVideo(ctx,
			ffmpeg.MediaAsset{Path: "", Type: "image"},
			"", p.AudioPath, tmpDir)
	}

	if err != nil {
		_ = h.videoRepo.UpdateStatus(ctx, jobID, video.JobStatusFailed, "FFmpeg error: "+err.Error())
		return fmt.Errorf("assembly failed: %w", err)
	}

	job.FFmpegLog = result.Log
	job.DurationSeconds = result.DurationSeconds
	job.FileSizeBytes = result.FileSizeBytes
	job.Status = video.JobStatusUploading
	if err := h.videoRepo.Update(ctx, job); err != nil {
		return err
	}

	// Chain to R2 upload
	payload, _ := json.Marshal(map[string]string{
		"job_id":     p.JobID,
		"video_path": result.VideoPath,
	})
	uploadTask := asynq.NewTask(queue.TaskUploadToR2, payload, asynq.Queue(queue.QueueVideo))
	if _, err := h.queueClient.EnqueueContext(ctx, uploadTask); err != nil {
		return fmt.Errorf("enqueue upload: %w", err)
	}

	logger.Info("assembly done, upload queued", zap.String("job_id", p.JobID))
	return nil
}
