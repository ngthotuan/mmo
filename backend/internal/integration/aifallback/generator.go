// Package aifallback wraps a primary ScriptGenerator and falls back to a
// secondary one (typically mockai) when the primary fails. Unlike the old
// silent-mock behaviour inside the Gemini client, every fallback is logged so
// quota/outage problems are observable instead of hidden.
package aifallback

import (
	"context"

	"mmo/internal/domain/ai"
	"mmo/pkg/logger"

	"go.uber.org/zap"
)

type Generator struct {
	primary  ai.ScriptGenerator
	fallback ai.ScriptGenerator
}

var _ ai.ScriptGenerator = (*Generator)(nil)

func New(primary, fallback ai.ScriptGenerator) *Generator {
	return &Generator{primary: primary, fallback: fallback}
}

func (g *Generator) GenerateScript(ctx context.Context, req ai.ScriptRequest) (*ai.ScriptResult, error) {
	res, err := g.primary.GenerateScript(ctx, req)
	if err != nil {
		logger.Warn("ai primary generator failed, using fallback",
			zap.String("topic", req.Topic), zap.Error(err))
		return g.fallback.GenerateScript(ctx, req)
	}
	return res, nil
}
