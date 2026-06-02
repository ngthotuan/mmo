// Package ai defines the provider-agnostic port for AI script generation.
// Domain layer: stdlib-only imports.
package ai

import "context"

// Scene is one segment of the video: a chunk of narration paired with the
// footage that should be shown WHILE it is spoken, so visuals track the content.
type Scene struct {
	Narration   string `json:"narration"`    // 2-4 sentences spoken during this scene
	VisualQuery string `json:"visual_query"` // ENGLISH stock-footage search for THIS scene
	Keyword     string `json:"keyword"`      // short on-screen label for the scene (localized)
}

// ScriptResult is the structured output of a single script generation.
type ScriptResult struct {
	Title    string   `json:"title"`
	Hook     string   `json:"hook"`
	Script   string   `json:"script"`
	CTA      string   `json:"cta"`
	Hashtags []string `json:"hashtags"`
	Caption  string   `json:"caption"`
	// VisualKeywords are ENGLISH stock-footage search terms describing the
	// content's visuals, so the b-roll matches the narration topic (Pexels /
	// Pixabay are English-keyword). Empty → fall back to title-derived queries.
	VisualKeywords []string `json:"visual_keywords"`
	// Scenes is the preferred, scene-by-scene breakdown. When present, the
	// pipeline produces per-scene footage + narration so visuals match content.
	// Script (above) is kept as the joined narration for fallback/compatibility.
	Scenes []Scene `json:"scenes"`
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
