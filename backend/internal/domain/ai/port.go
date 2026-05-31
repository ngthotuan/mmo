// Package ai defines the provider-agnostic port for AI script generation.
// Domain layer: stdlib-only imports.
package ai

import "context"

// ScriptResult is the structured output of a single script generation.
type ScriptResult struct {
	Title    string   `json:"title"`
	Hook     string   `json:"hook"`
	Script   string   `json:"script"`
	CTA      string   `json:"cta"`
	Hashtags []string `json:"hashtags"`
	Caption  string   `json:"caption"`
}

// ScriptRequest carries everything a generator needs to produce a script.
type ScriptRequest struct {
	Topic        string // trend title / subject
	Niche        string // e.g. "kiếm tiền online"
	Platform     string // tiktok | facebook | youtube
	DurationSecs int    // target narration length
	Language     string // "vi" | "en"
}

// ScriptGenerator is the port implemented by concrete providers (Gemini, mock, …).
// Swapping the LLM provider means providing a different implementation — no caller changes.
type ScriptGenerator interface {
	GenerateScript(ctx context.Context, req ScriptRequest) (*ScriptResult, error)
}
