package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/content"
	"mmo/internal/integration/gemini"
	"mmo/internal/infrastructure/queue"
	apperr "mmo/pkg/errors"
	"github.com/hibiken/asynq"
)

type ContentUsecase struct {
	trendRepo   *repository.TrendRepo
	planRepo    *repository.ContentPlanRepo
	gemini      *gemini.Client
	queueClient *asynq.Client
}

func NewContentUsecase(
	trendRepo *repository.TrendRepo,
	planRepo *repository.ContentPlanRepo,
	geminiClient *gemini.Client,
	queueClient *asynq.Client,
) *ContentUsecase {
	return &ContentUsecase{
		trendRepo:   trendRepo,
		planRepo:    planRepo,
		gemini:      geminiClient,
		queueClient: queueClient,
	}
}

// ListTrends returns paginated trend topics for a user.
func (uc *ContentUsecase) ListTrends(ctx context.Context, userID uuid.UUID, status string, page, perPage int) ([]*content.TrendTopic, int, error) {
	p := paginationOf(page, perPage)
	return uc.trendRepo.List(ctx, userID, status, p)
}

// TriggerDiscover enqueues a manual trend discovery job.
func (uc *ContentUsecase) TriggerDiscover(ctx context.Context, userID uuid.UUID) error {
	payload, _ := json.Marshal(map[string]string{"user_id": userID.String()})
	task := asynq.NewTask(queue.TaskDiscoverTrends, payload, asynq.Queue(queue.QueueLow))
	_, err := uc.queueClient.EnqueueContext(ctx, task)
	return err
}

// ListPlans returns paginated content plans.
func (uc *ContentUsecase) ListPlans(ctx context.Context, userID uuid.UUID, status string, page, perPage int) ([]*content.ContentPlan, int, error) {
	p := paginationOf(page, perPage)
	return uc.planRepo.List(ctx, userID, content.Status(status), p)
}

// GetPlan returns a single content plan, verifying ownership.
func (uc *ContentUsecase) GetPlan(ctx context.Context, userID, planID uuid.UUID) (*content.ContentPlan, error) {
	plan, err := uc.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan.UserID != userID {
		return nil, apperr.ErrForbidden
	}
	return plan, nil
}

// UpdatePlan allows editing title, script, notes, target_platforms.
func (uc *ContentUsecase) UpdatePlan(ctx context.Context, userID, planID uuid.UUID, title, niche, script, notes string, platforms []string) (*content.ContentPlan, error) {
	plan, err := uc.GetPlan(ctx, userID, planID)
	if err != nil {
		return nil, err
	}
	if title != "" {
		plan.Title = title
	}
	if niche != "" {
		plan.Niche = niche
	}
	if script != "" {
		plan.Script = script
	}
	if notes != "" {
		plan.Notes = notes
	}
	if len(platforms) > 0 {
		plan.TargetPlatforms = platforms
	}
	if err := uc.planRepo.Update(ctx, plan); err != nil {
		return nil, err
	}
	return plan, nil
}

// ApprovePlan marks a plan as approved and enqueues video creation.
func (uc *ContentUsecase) ApprovePlan(ctx context.Context, userID, planID uuid.UUID) error {
	plan, err := uc.GetPlan(ctx, userID, planID)
	if err != nil {
		return err
	}
	if plan.Status != content.StatusDraft {
		return apperr.Newf(400, "plan must be in draft status to approve, current: %s", plan.Status)
	}
	if err := uc.planRepo.UpdateStatus(ctx, planID, content.StatusApproved); err != nil {
		return err
	}
	// Enqueue video creation pipeline
	payload, _ := json.Marshal(map[string]string{"content_plan_id": planID.String()})
	task := asynq.NewTask(queue.TaskCollectMedia, payload, asynq.Queue(queue.QueueDefault))
	_, err = uc.queueClient.EnqueueContext(ctx, task)
	return err
}

// RejectPlan marks a plan as rejected.
func (uc *ContentUsecase) RejectPlan(ctx context.Context, userID, planID uuid.UUID) error {
	plan, err := uc.GetPlan(ctx, userID, planID)
	if err != nil {
		return err
	}
	if plan.Status != content.StatusDraft {
		return apperr.Newf(400, "only draft plans can be rejected")
	}
	return uc.planRepo.UpdateStatus(ctx, planID, content.StatusRejected)
}

// RegenerateScript re-runs Gemini for a plan's topic.
func (uc *ContentUsecase) RegenerateScript(ctx context.Context, userID, planID uuid.UUID) (*content.ContentPlan, error) {
	plan, err := uc.GetPlan(ctx, userID, planID)
	if err != nil {
		return nil, err
	}

	result, err := uc.gemini.GenerateScript(ctx, plan.Title, plan.Niche,
		firstPlatform(plan.TargetPlatforms), 60)
	if err != nil {
		return nil, err
	}

	meta, _ := json.Marshal(map[string]any{
		"hook": result.Hook, "cta": result.CTA,
		"hashtags": result.Hashtags, "caption": result.Caption,
	})
	plan.Script = result.Script
	plan.ScriptMetadata = meta
	plan.Status = content.StatusDraft

	if err := uc.planRepo.Update(ctx, plan); err != nil {
		return nil, err
	}
	return plan, nil
}

// DeletePlan deletes a plan (draft or rejected only).
func (uc *ContentUsecase) DeletePlan(ctx context.Context, userID, planID uuid.UUID) error {
	plan, err := uc.GetPlan(ctx, userID, planID)
	if err != nil {
		return err
	}
	if plan.Status != content.StatusDraft && plan.Status != content.StatusRejected {
		return apperr.Newf(400, "only draft or rejected plans can be deleted")
	}
	return uc.planRepo.Delete(ctx, planID)
}

// GenerateScriptForTrend creates a ContentPlan from a TrendTopic via Gemini.
func (uc *ContentUsecase) GenerateScriptForTrend(ctx context.Context, userID, topicID uuid.UUID, niche string, platforms []string, autoApprove bool) (*content.ContentPlan, error) {
	topic, err := uc.trendRepo.GetByID(ctx, topicID)
	if err != nil {
		return nil, err
	}

	result, err := uc.gemini.GenerateScript(ctx, topic.Title, niche, firstPlatform(platforms), 60)
	if err != nil {
		return nil, err
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
		Niche:           niche,
		TargetPlatforms: platforms,
		Script:          result.Script,
		ScriptMetadata:  meta,
		Status:          content.StatusDraft,
		AutoApprove:     autoApprove,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := uc.planRepo.Create(ctx, plan); err != nil {
		return nil, err
	}

	_ = uc.trendRepo.UpdateStatus(ctx, topicID, "used")

	if autoApprove {
		_ = uc.planRepo.UpdateStatus(ctx, plan.ID, content.StatusApproved)
		payload, _ := json.Marshal(map[string]string{"content_plan_id": plan.ID.String()})
		task := asynq.NewTask(queue.TaskCollectMedia, payload, asynq.Queue(queue.QueueDefault))
		_, _ = uc.queueClient.EnqueueContext(ctx, task)
	}

	return plan, nil
}

func firstPlatform(platforms []string) string {
	if len(platforms) > 0 {
		return platforms[0]
	}
	return "TikTok"
}
