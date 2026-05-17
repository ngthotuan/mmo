package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"mmo/pkg/config"
)

type Client struct {
	apiKey      string
	model       string
	apiBase     string
	httpClient  *http.Client
}

type ScriptResult struct {
	Title    string   `json:"title"`
	Hook     string   `json:"hook"`
	Script   string   `json:"script"`
	CTA      string   `json:"cta"`
	Hashtags []string `json:"hashtags"`
	Caption  string   `json:"caption"`
}

func New(cfg config.GeminiConfig) *Client {
	return &Client{
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		apiBase:    cfg.APIBase,
		httpClient: &http.Client{Timeout: cfg.HTTPTimeout},
	}
}

func (c *Client) GenerateScript(ctx context.Context, topic, niche, platform string, durationSecs int) (*ScriptResult, error) {
	if c.apiKey == "" {
		return c.mockScript(topic), nil
	}

	prompt := buildScriptPrompt(topic, niche, platform, durationSecs)

	payload := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]any{
			"temperature":      0.8,
			"topP":             0.95,
			"maxOutputTokens":  1024,
			"responseMimeType": "application/json",
		},
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", c.apiBase, c.model, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse gemini response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("gemini error: %s", result.Error.Message)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini returned empty response")
	}

	text := result.Candidates[0].Content.Parts[0].Text
	var script ScriptResult
	if err := json.Unmarshal([]byte(text), &script); err != nil {
		script = ScriptResult{
			Title:  topic,
			Script: text,
			Hook:   "",
			CTA:    "Follow for more!",
		}
	}
	return &script, nil
}

func buildScriptPrompt(topic, niche, platform string, durationSecs int) string {
	if niche == "" {
		niche = "general"
	}
	if platform == "" {
		platform = "TikTok"
	}
	if durationSecs <= 0 {
		durationSecs = 60
	}
	return fmt.Sprintf(`You are a viral short-form content creator for %s. Create a %d-second video script about: "%s" in the %s niche.

Output ONLY a JSON object (no markdown, no explanation) with this exact structure:
{
  "title": "catchy video title (max 80 chars)",
  "hook": "first 3 seconds opening line to grab attention",
  "script": "full narration script, natural speaking pace, ~%d seconds when spoken",
  "cta": "call to action at the end (e.g. 'Follow for more tips!')",
  "hashtags": ["tag1", "tag2", "tag3", "tag4", "tag5"],
  "caption": "post caption with emojis (max 200 chars)"
}

Rules:
- Hook must be a question or surprising statement
- Script must be conversational and engaging, not formal
- Include 5-8 relevant hashtags (no # symbol)
- Caption must be engaging with 2-3 emojis`, platform, durationSecs, topic, niche, durationSecs)
}

func (c *Client) mockScript(topic string) *ScriptResult {
	return &ScriptResult{
		Title:    "Did you know about " + topic + "?",
		Hook:     "You won't believe this about " + topic + "...",
		Script:   "Here's what everyone is talking about: " + topic + ". This is trending right now and here's why it matters to you. [Add your key points here] The bottom line is simple and actionable. Don't miss out on this!",
		CTA:      "Follow for more daily trending content!",
		Hashtags: []string{"trending", "viral", "fyp", "foryou", "content"},
		Caption:  "🔥 " + topic + " is trending! Here's everything you need to know 👇",
	}
}
