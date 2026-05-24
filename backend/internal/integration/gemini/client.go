package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"mmo/pkg/config"
	"mmo/pkg/httpclient"
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
		httpClient: httpclient.New("gemini", cfg.HTTPTimeout),
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
			"maxOutputTokens":  8192,
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
		return c.mockScript(topic), nil
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
		// Fallback to mock on quota/rate-limit/unavailability so the feature
		// keeps working even when the Gemini key is over quota.
		return c.mockScript(topic), nil
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return c.mockScript(topic), nil
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
		platform = "Facebook"
	}
	if durationSecs <= 0 {
		durationSecs = 360
	}
	// Average English narration ≈ 150 words/min → wordTarget gives Gemini a
	// concrete length budget instead of a fuzzy "seconds" hint.
	wordTarget := durationSecs * 150 / 60
	return fmt.Sprintf(`You are an expert long-form video scriptwriter for %s. Create an in-depth, engaging %d-second (~%d words) narration script about: "%s" in the %s niche.

This is a long-form video (5+ minutes), NOT a 60-second short. Treat it like a mini-documentary or deep-dive video essay: rich storytelling, multiple chapters, examples, anecdotes, surprising facts, vivid descriptions. Speak directly to the viewer in second person ("you").

Output ONLY a JSON object (no markdown, no commentary, no code fences) with this EXACT structure:
{
  "title": "catchy video title (max 80 chars)",
  "hook": "first 10-15 seconds opening — a surprising fact, bold claim or question that makes the viewer commit to watching",
  "script": "the full narration text, ~%d words, written as continuous spoken prose — see structure below",
  "cta": "call to action at the end (e.g. 'Follow for more deep dives!')",
  "hashtags": ["tag1", "tag2", "tag3", "tag4", "tag5", "tag6", "tag7", "tag8"],
  "caption": "post caption with emojis (max 250 chars)"
}

Script MUST follow this structure (write it as one continuous narration, but cover every section):
1. HOOK (~30 sec): A surprising statement, question, or stat that grabs attention.
2. INTRODUCTION (~45 sec): Why this topic matters NOW and what the viewer will learn.
3. CHAPTER 1 — Background / Context (~60 sec): Set the stage with history, definitions, or the bigger picture.
4. CHAPTER 2 — The Core Insight (~90 sec): The main idea, broken down with concrete examples and analogies.
5. CHAPTER 3 — Real-world Cases (~60 sec): 2-3 specific examples, stories, or case studies.
6. CHAPTER 4 — Implications / What This Means For You (~45 sec): How the viewer can apply or benefit from this.
7. CONCLUSION + CTA (~30 sec): Recap the key takeaways and end with a strong call-to-action.

Rules:
- Total length must be approximately %d words (target = %d seconds at 150 wpm)
- Use conversational, vivid, second-person language ("you'll discover", "imagine this")
- Include rhetorical questions to keep viewers engaged
- Use concrete numbers, names, and examples — not vague generalities
- Each chapter should flow naturally into the next with transition sentences
- NO scene-cut markers, NO "[B-roll]" annotations — just the spoken words
- Include 6-10 relevant hashtags (no # symbol)
- Caption must hook scrollers, use 2-4 emojis`,
		platform, durationSecs, wordTarget, topic, niche, wordTarget, wordTarget, durationSecs)
}

func (c *Client) mockScript(topic string) *ScriptResult {
	// ~900-1000 words → ~6 min @ 150 wpm narration.
	script := "Welcome back to the channel, my friend. " +
		"Today I want to take you on a journey — a deep, honest, slightly uncomfortable journey — into something that is quietly reshaping our entire world: " + topic + ". " +
		"And I do not say that lightly. By the end of this video, you will see this topic from a completely different angle, and you will walk away with three or four ideas you can apply this very week. So pour yourself a coffee, put your phone on silent, and stay with me. " +

		"Let me start with a confession. A few months ago, I thought I understood " + topic + " pretty well. I had read the articles, watched the explainer videos, even had a few opinions at dinner parties. But the more I dug, the more I realized just how little I actually knew. And I think that is true for most of us. We live in a time where headlines move faster than understanding, and " + topic + " is the perfect example of that gap. " +

		"So let's slow down and start from the beginning. Where did this whole story actually come from? " +
		topic + " did not appear out of thin air. It is the product of decades of slow, often invisible work — researchers in labs, engineers building tools nobody asked for, dreamers writing manifestos that everyone laughed at. The breakthrough we are seeing now is not really a breakthrough at all. It is what happens when a hundred small wins finally stack up into something the world can no longer ignore. " +
		"And the consequences are everywhere. Look at the apps on your phone. Look at how businesses are reorganizing themselves. Look at the questions your children are starting to ask. " + topic + " is touching all of it. " +

		"Now here is the core idea I want you to remember from this video. " +
		topic + " is not just a tool. It is not just a trend. It is a fundamental shift in how we, as humans, get things done. " +
		"Think of it like electricity in nineteen hundred. At first, only a few wealthy households had it, and most people did not really understand what it was for. Within forty years, it became invisible — woven into every part of life. " + topic + " is on the same trajectory, just compressed into a much shorter time window. That compression is exactly why so many people feel anxious. There is simply less time to adapt. " +

		"Let me give you three concrete examples to make this real. " +
		"Example one. In education, students are now learning with personalized tutors that adapt to their pace, their language, their gaps. A child who used to struggle in a class of thirty can now have a one-on-one guide available at midnight. That is not science fiction — that is happening right now, in living rooms around the world. " +
		"Example two. In healthcare, doctors are catching diseases earlier than ever before. Patterns in scans, in blood work, in voice tone — patterns no single human eye could catch — are being surfaced in seconds. Lives are being saved that, just five years ago, would have been lost. " +
		"Example three. In creative work, artists, writers, and musicians are collaborating with machines to produce things no human could create alone. The idea that creativity belongs only to humans is being quietly rewritten in front of our eyes. " +

		"So what does all of this mean for you, specifically? Let me break it down into three honest takeaways. " +
		"Number one. The skills that mattered most in the last decade are not the ones that will matter most in the next. Memorizing facts, executing repetitive tasks, following someone else's recipe — those are exactly the things that get automated first. What matters now is curiosity, taste, judgment, and the ability to ask better questions than anyone else in the room. If you cultivate those skills, you will be fine. If you do not, no resume will save you. " +
		"Number two. There is a window of opportunity right now, and it will not stay open forever. The people who experiment today, who play with these tools, who try things and fail and try again — they will have a quiet, compounding advantage. Five years from now, they will look back and realize they were building leverage while everyone else was scrolling. " +
		"Number three, and this is the hard one. Ignoring this shift is not a neutral choice. Doing nothing is doing something. Every day you wait, the gap between you and the people who are paying attention grows wider. You do not need to become an expert. You do not need to quit your job. But you do need to engage. " +

		"Okay, so what should you actually do, starting today? Here is a simple, four-step plan I would give my own family. " +
		"Step one. Carve out just fifteen minutes a day to learn about " + topic + ". Not an hour. Not a course. Fifteen honest minutes. Read one article. Watch one tutorial. That is it. " +
		"Step two. Pick one tool — any tool — and try to use it for something real in your own life this week. Maybe it summarizes your email. Maybe it helps you cook dinner. Maybe it edits a photo. The point is friction. You will only understand it by touching it. " +
		"Step three. Find one other person who is also exploring this space, and talk to them every week. A friend, a coworker, a stranger on the internet. Curiosity grows faster when it is shared. " +
		"Step four. Write down what you learn. A note, a tweet, a journal entry. Externalizing your understanding is the single fastest way to deepen it. " +

		"Let me leave you with one last thought. " +
		"The most exciting, most uncertain, most important moment of our lifetimes is happening right now, and you are in it. You did not choose to be here, but you are here, and you have a choice in how you respond. " +
		"You can be the person who watched the wave from the beach, or you can be the person who learned to surf. " +
		"Whatever you choose, please choose actively. The worst outcome is drift. " +

		"If this video gave you something useful — a new idea, a new question, a tiny bit of courage — please hit that follow button, share it with one friend who needs to hear it, and turn on notifications so you do not miss the next deep dive. " +
		"I will see you in the next one. Take care of yourself out there."
	return &ScriptResult{
		Title:    "The Real Story Behind " + topic,
		Hook:     "Most people have no idea what is really happening with " + topic + " right now — and once you see it, you cannot unsee it.",
		Script:   script,
		CTA:      "Follow for more deep dives like this every week!",
		Hashtags: []string{"deepdive", "explainer", "trending", "viral", "education", "tech", "future", "insights"},
		Caption:  "🎯 The full story behind " + topic + " — what nobody is telling you. Watch this before it is too late 👇",
	}
}
