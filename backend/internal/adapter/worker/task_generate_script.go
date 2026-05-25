package worker

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/content"
	"mmo/internal/integration/gemini"
	"mmo/internal/infrastructure/queue"
	"mmo/pkg/logger"
	"go.uber.org/zap"
)

type ScriptGenHandler struct {
	trendRepo          *repository.TrendRepo
	planRepo           *repository.ContentPlanRepo
	gemini             *gemini.Client
	queueClient        *asynq.Client
	targetDurationSecs int
	language           string
}

func NewScriptGenHandler(
	trendRepo *repository.TrendRepo,
	planRepo *repository.ContentPlanRepo,
	geminiClient *gemini.Client,
	queueClient *asynq.Client,
	targetDurationSecs int,
	language string,
) *ScriptGenHandler {
	return &ScriptGenHandler{
		trendRepo:          trendRepo,
		planRepo:           planRepo,
		gemini:             geminiClient,
		queueClient:        queueClient,
		targetDurationSecs: targetDurationSecs,
		language:           language,
	}
}

type scriptPayload struct {
	TopicID     string   `json:"topic_id"`
	UserID      string   `json:"user_id"`
	Niche       string   `json:"niche"`
	Platforms   []string `json:"platforms"`
	AutoApprove bool     `json:"auto_approve"`
}

func (h *ScriptGenHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var p scriptPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		return err
	}

	topicID, err := uuid.Parse(p.TopicID)
	if err != nil {
		return err
	}
	userID, err := uuid.Parse(p.UserID)
	if err != nil {
		return err
	}

	topic, err := h.trendRepo.GetByID(ctx, topicID)
	if err != nil {
		logger.Error("trend topic not found", zap.String("topic_id", p.TopicID), zap.Error(err))
		return err
	}

	if len(p.Platforms) == 0 {
		p.Platforms = []string{"tiktok"}
	}
	platform := p.Platforms[0]

	result, err := h.gemini.GenerateScript(ctx, topic.Title, p.Niche, platform, h.targetDurationSecs, h.language)
	if err != nil {
		logger.Error("gemini script generation failed", zap.Error(err))
		return err
	}

	meta, _ := json.Marshal(map[string]any{
		"hook": result.Hook, "cta": result.CTA,
		"hashtags": result.Hashtags, "caption": result.Caption,
	})

	plan := &content.ContentPlan{
		ID:              uuid.New(),
		UserID:          userID,
		TrendTopicID:    &topicID,
		Title:           result.Title,
		Niche:           p.Niche,
		TargetPlatforms: p.Platforms,
		Script:          result.Script,
		ScriptMetadata:  meta,
		Status:          content.StatusDraft,
		AutoApprove:     p.AutoApprove,
	}

	if err := h.planRepo.Create(ctx, plan); err != nil {
		logger.Error("failed to save content plan", zap.Error(err))
		return err
	}

	_ = h.trendRepo.UpdateStatus(ctx, topicID, "used")

	logger.Info("script generated", zap.String("plan_id", plan.ID.String()), zap.String("title", plan.Title))

	if p.AutoApprove {
		_ = h.planRepo.UpdateStatus(ctx, plan.ID, content.StatusApproved)
		payload, _ := json.Marshal(map[string]string{"content_plan_id": plan.ID.String()})
		mediaTask := asynq.NewTask(queue.TaskCollectMedia, payload, asynq.Queue(queue.QueueVideo))
		if _, err := h.queueClient.EnqueueContext(ctx, mediaTask); err != nil {
			logger.Warn("failed to enqueue media collection", zap.Error(err))
		}
	}

	return nil
}
