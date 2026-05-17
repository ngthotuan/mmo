package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/video"
	"mmo/internal/infrastructure/ffmpeg"
	"mmo/internal/infrastructure/queue"
	"mmo/internal/infrastructure/storage"
	"mmo/internal/integration/edgetts"
	"mmo/pkg/logger"
	"go.uber.org/zap"
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
		JobID  string `json:"job_id"`
		PlanID string `json:"plan_id"`
		Script string `json:"script"`
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

	logger.Info("generating TTS", zap.String("job_id", p.JobID))

	result, err := h.tts.Generate(ctx, p.Script, h.tts.DefaultVoice(), tmpDir)
	if err != nil {
		_ = h.videoRepo.UpdateStatus(ctx, jobID, video.JobStatusFailed, "TTS generation failed: "+err.Error())
		return err
	}

	// Convert VTT → SRT for FFmpeg
	srtPath, err := edgetts.VTTToSRT(result.SubtitlePath)
	if err != nil {
		logger.Warn("subtitle conversion failed", zap.Error(err))
		srtPath = ""
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
	payload, _ := json.Marshal(map[string]string{
		"job_id":       p.JobID,
		"audio_path":   result.AudioPath,
		"subtitle_path": srtPath,
	})
	assembleTask := asynq.NewTask(queue.TaskAssembleVideo, payload, asynq.Queue(queue.QueueDefault))
	if _, err := h.queueClient.EnqueueContext(ctx, assembleTask); err != nil {
		return fmt.Errorf("enqueue assemble: %w", err)
	}

	logger.Info("TTS done, assembly queued", zap.String("job_id", p.JobID))
	return nil
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
		JobID        string `json:"job_id"`
		AudioPath    string `json:"audio_path"`
		SubtitlePath string `json:"subtitle_path"`
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

	var result *ffmpeg.AssembleResult
	if len(ffAssets) > 0 && ffAssets[0].Type == "video" {
		result, err = h.assembler.AssembleBRoll(ctx, ffAssets, p.AudioPath, p.SubtitlePath, tmpDir)
	} else if len(ffAssets) > 0 {
		result, err = h.assembler.AssembleSlideshow(ctx, ffAssets, p.AudioPath, p.SubtitlePath, tmpDir)
	} else {
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
	uploadTask := asynq.NewTask(queue.TaskUploadToR2, payload, asynq.Queue(queue.QueueDefault))
	if _, err := h.queueClient.EnqueueContext(ctx, uploadTask); err != nil {
		return fmt.Errorf("enqueue upload: %w", err)
	}

	logger.Info("assembly done, upload queued", zap.String("job_id", p.JobID))
	return nil
}
