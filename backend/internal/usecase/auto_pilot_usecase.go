package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/autopilot"
	"mmo/internal/domain/content"
	"mmo/internal/integration/gemini"
	"mmo/internal/infrastructure/queue"
	apperr "mmo/pkg/errors"
	"mmo/pkg/logger"
	"go.uber.org/zap"
)

type AutoPilotUsecase struct {
	profileRepo        *repository.AutoPilotRepo
	trendRepo          *repository.TrendRepo
	planRepo           *repository.ContentPlanRepo
	gemini             *gemini.Client
	queueClient        *asynq.Client
	targetDurationSecs int
	language           string
}

func NewAutoPilotUsecase(
	profileRepo *repository.AutoPilotRepo,
	trendRepo *repository.TrendRepo,
	planRepo *repository.ContentPlanRepo,
	gemini *gemini.Client,
	queueClient *asynq.Client,
	targetDurationSecs int,
	language string,
) *AutoPilotUsecase {
	return &AutoPilotUsecase{
		profileRepo:        profileRepo,
		trendRepo:          trendRepo,
		planRepo:           planRepo,
		gemini:             gemini,
		queueClient:        queueClient,
		targetDurationSecs: targetDurationSecs,
		language:           language,
	}
}

// ─── CRUD ─────────────────────────────────────────────────────────────────────

func (uc *AutoPilotUsecase) List(ctx context.Context, userID uuid.UUID) ([]*autopilot.Profile, error) {
	return uc.profileRepo.List(ctx, userID)
}

func (uc *AutoPilotUsecase) Get(ctx context.Context, userID, id uuid.UUID) (*autopilot.Profile, error) {
	p, err := uc.profileRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.UserID != userID {
		return nil, apperr.ErrForbidden
	}
	return p, nil
}

func (uc *AutoPilotUsecase) Create(ctx context.Context, p *autopilot.Profile) (*autopilot.Profile, error) {
	if p.Name == "" {
		return nil, apperr.New(400, "name is required")
	}
	if p.DailyCount <= 0 {
		p.DailyCount = 1
	}
	if p.DailyCount > 20 {
		return nil, apperr.New(400, "daily_count must be ≤ 20")
	}
	if len(p.TargetPlatforms) == 0 {
		return nil, apperr.New(400, "target_platforms is required")
	}
	if len(p.ScheduleTimes) == 0 {
		return nil, apperr.New(400, "schedule_times is required (e.g. [\"09:00\",\"19:00\"])")
	}
	for _, t := range p.ScheduleTimes {
		if !isValidHHMM(t) {
			return nil, apperr.New(400, "invalid time format, must be HH:MM 24h")
		}
	}
	p.ID = uuid.New()
	if err := uc.profileRepo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (uc *AutoPilotUsecase) Update(ctx context.Context, userID uuid.UUID, p *autopilot.Profile) (*autopilot.Profile, error) {
	existing, err := uc.Get(ctx, userID, p.ID)
	if err != nil {
		return nil, err
	}
	p.UserID = existing.UserID
	if err := uc.profileRepo.Update(ctx, p); err != nil {
		return nil, err
	}
	return uc.Get(ctx, userID, p.ID)
}

func (uc *AutoPilotUsecase) Delete(ctx context.Context, userID, id uuid.UUID) error {
	if _, err := uc.Get(ctx, userID, id); err != nil {
		return err
	}
	return uc.profileRepo.Delete(ctx, id)
}

func (uc *AutoPilotUsecase) Toggle(ctx context.Context, userID, id uuid.UUID, enabled bool) error {
	if _, err := uc.Get(ctx, userID, id); err != nil {
		return err
	}
	return uc.profileRepo.Toggle(ctx, id, enabled)
}

// ─── RunTick ──────────────────────────────────────────────────────────────────

// TickAll is called by the cron worker every N minutes. It picks enabled profiles
// whose scheduled time falls within the window since their last run, then runs them.
func (uc *AutoPilotUsecase) TickAll(ctx context.Context, now time.Time) (int, error) {
	profiles, err := uc.profileRepo.ListEnabled(ctx)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, p := range profiles {
		if !uc.shouldRun(p, now) {
			continue
		}
		n, err := uc.RunProfile(ctx, p)
		if err != nil {
			logger.Error("auto-pilot profile run failed",
				zap.String("profile_id", p.ID.String()), zap.Error(err))
			continue
		}
		total += n
		_ = uc.profileRepo.MarkRun(ctx, p.ID, n)
	}
	return total, nil
}

// RunProfile generates and queues videos for a single profile.
// Returns the number of plans created (and approved if AutoApprove).
func (uc *AutoPilotUsecase) RunProfile(ctx context.Context, p *autopilot.Profile) (int, error) {
	trends, err := uc.trendRepo.ListNewMatching(ctx, p.UserID, p.TrendFilter, p.TrendSources, p.DailyCount*2)
	if err != nil {
		return 0, err
	}
	if len(trends) == 0 {
		logger.Info("auto-pilot: no matching trends",
			zap.String("profile_id", p.ID.String()), zap.String("filter", p.TrendFilter))
		return 0, nil
	}

	count := 0
	for _, t := range trends {
		if count >= p.DailyCount {
			break
		}
		if err := uc.generatePlanForTrend(ctx, p, t); err != nil {
			logger.Warn("auto-pilot: plan generation failed",
				zap.String("topic_id", t.ID.String()), zap.Error(err))
			continue
		}
		count++
	}
	logger.Info("auto-pilot ran profile",
		zap.String("profile_id", p.ID.String()),
		zap.String("name", p.Name),
		zap.Int("plans_created", count))
	return count, nil
}

func (uc *AutoPilotUsecase) generatePlanForTrend(ctx context.Context, p *autopilot.Profile, t *content.TrendTopic) error {
	platform := "TikTok"
	if len(p.TargetPlatforms) > 0 {
		platform = p.TargetPlatforms[0]
	}
	result, err := uc.gemini.GenerateScript(ctx, t.Title, p.Niche, platform, uc.targetDurationSecs, uc.language)
	if err != nil {
		return fmt.Errorf("gemini: %w", err)
	}

	meta, _ := json.Marshal(map[string]any{
		"hook": result.Hook, "cta": result.CTA,
		"hashtags": result.Hashtags, "caption": result.Caption,
	})

	planID := uuid.New()
	profileID := p.ID
	topicID := t.ID
	plan := &content.ContentPlan{
		ID:                 planID,
		UserID:             p.UserID,
		TrendTopicID:       &topicID,
		AutoPilotProfileID: &profileID,
		Title:              result.Title,
		Niche:              p.Niche,
		TargetPlatforms:    p.TargetPlatforms,
		Script:             result.Script,
		ScriptMetadata:     meta,
		Status:             content.StatusDraft,
		AutoApprove:        p.AutoApprove,
		Voice:              p.Voice,
	}
	if err := uc.planRepo.Create(ctx, plan); err != nil {
		return fmt.Errorf("create plan: %w", err)
	}
	_ = uc.trendRepo.UpdateStatus(ctx, t.ID, "used")

	if p.AutoApprove {
		if err := uc.planRepo.UpdateStatus(ctx, planID, content.StatusApproved); err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]string{"content_plan_id": planID.String()})
		task := asynq.NewTask(queue.TaskCollectMedia, payload, asynq.Queue(queue.QueueVideo))
		if _, err := uc.queueClient.EnqueueContext(ctx, task); err != nil {
			return fmt.Errorf("enqueue media: %w", err)
		}
	}
	return nil
}

// shouldRun decides whether the profile is due for a tick at `now`.
//
// Rules:
//   - profile has at least one schedule_time slot within (now - 30min, now]
//   - last_run_at is older than the matching slot (so we don't double-run within the same slot)
func (uc *AutoPilotUsecase) shouldRun(p *autopilot.Profile, now time.Time) bool {
	if len(p.ScheduleTimes) == 0 {
		return false
	}
	windowStart := now.Add(-30 * time.Minute)
	for _, hhmm := range p.ScheduleTimes {
		slot, ok := parseHHMM(hhmm, now)
		if !ok {
			continue
		}
		if slot.After(windowStart) && !slot.After(now) {
			if p.LastRunAt == nil || p.LastRunAt.Before(slot) {
				return true
			}
		}
	}
	return false
}

func isValidHHMM(s string) bool {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return false
	}
	if len(parts[0]) != 2 || len(parts[1]) != 2 {
		return false
	}
	h, m := 0, 0
	if _, err := fmt.Sscanf(parts[0], "%d", &h); err != nil {
		return false
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &m); err != nil {
		return false
	}
	return h >= 0 && h < 24 && m >= 0 && m < 60
}

func parseHHMM(s string, ref time.Time) (time.Time, bool) {
	if !isValidHHMM(s) {
		return time.Time{}, false
	}
	parts := strings.Split(s, ":")
	var h, m int
	_, _ = fmt.Sscanf(parts[0], "%d", &h)
	_, _ = fmt.Sscanf(parts[1], "%d", &m)
	loc := ref.Location()
	return time.Date(ref.Year(), ref.Month(), ref.Day(), h, m, 0, 0, loc), true
}
