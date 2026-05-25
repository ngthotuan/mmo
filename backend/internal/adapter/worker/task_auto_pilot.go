package worker

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"mmo/internal/usecase"
	"mmo/pkg/logger"
	"go.uber.org/zap"
)

type AutoPilotTickHandler struct {
	uc *usecase.AutoPilotUsecase
}

func NewAutoPilotTickHandler(uc *usecase.AutoPilotUsecase) *AutoPilotTickHandler {
	return &AutoPilotTickHandler{uc: uc}
}

func (h *AutoPilotTickHandler) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	now := time.Now()
	created, err := h.uc.TickAll(ctx, now)
	if err != nil {
		logger.Error("auto-pilot tick failed", zap.Error(err))
		return err
	}
	if created > 0 {
		logger.Info("auto-pilot tick complete",
			zap.Int("plans_created", created), zap.Time("at", now))
	}
	return nil
}
