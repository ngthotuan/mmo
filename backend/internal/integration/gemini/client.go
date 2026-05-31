package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"mmo/internal/domain/ai"
	"mmo/pkg/config"
	"mmo/pkg/httpclient"
)

// ErrNoAPIKey is returned when the Gemini API key is not configured. Callers
// (or the aifallback wrapper) decide how to react — there is NO silent mock here.
var ErrNoAPIKey = errors.New("gemini: api key not configured")

type Client struct {
	apiKey     string
	model      string
	apiBase    string
	httpClient *http.Client
}

// Compile-time guarantee that Client satisfies the provider-agnostic port.
var _ ai.ScriptGenerator = (*Client)(nil)

func New(cfg config.GeminiConfig) *Client {
	return &Client{
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		apiBase:    cfg.APIBase,
		httpClient: httpclient.New("gemini", cfg.HTTPTimeout),
	}
}

func (c *Client) GenerateScript(ctx context.Context, req ai.ScriptRequest) (*ai.ScriptResult, error) {
	if c.apiKey == "" {
		return nil, ErrNoAPIKey
	}

	prompt := buildScriptPrompt(req)

	payload := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]any{{"text": prompt}}},
		},
		"generationConfig": map[string]any{
			"temperature":      0.85,
			"topP":             0.95,
			"maxOutputTokens":  8192,
			"responseMimeType": "application/json",
		},
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", c.apiBase, c.model, c.apiKey)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
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
		return nil, fmt.Errorf("gemini: parse response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("gemini api error: %s", result.Error.Message)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini: empty response")
	}

	text := result.Candidates[0].Content.Parts[0].Text
	var script ai.ScriptResult
	if err := json.Unmarshal([]byte(text), &script); err != nil {
		return nil, fmt.Errorf("gemini: parse script json: %w", err)
	}
	if strings.TrimSpace(script.Script) == "" {
		return nil, fmt.Errorf("gemini: empty script body")
	}
	return &script, nil
}

func buildScriptPrompt(req ai.ScriptRequest) string {
	niche := req.Niche
	if niche == "" {
		niche = "general"
	}
	platform := req.Platform
	if platform == "" {
		platform = "Facebook"
	}
	durationSecs := req.DurationSecs
	if durationSecs <= 0 {
		durationSecs = 360
	}

	if req.Language == "vi" {
		// Vietnamese narration ≈ 120 words/min (slower than English due to tones)
		wordTarget := durationSecs * 120 / 60
		return fmt.Sprintf(`Bạn là một chuyên gia viết kịch bản video dài cho %s. Hãy tạo một kịch bản thuyết minh sâu sắc, hấp dẫn dài %d giây (~%d từ) bằng TIẾNG VIỆT về chủ đề: "%s" trong lĩnh vực %s.

ĐÂY LÀ VIDEO DÀI (hơn 5 phút), KHÔNG phải short 60 giây. Hãy viết như một phóng sự mini hoặc video phân tích chuyên sâu: kể chuyện phong phú, nhiều chương, ví dụ thực tế, giai thoại thú vị, sự kiện bất ngờ, miêu tả sinh động. Nói trực tiếp với người xem theo ngôi thứ hai ("bạn").
%s
TOÀN BỘ nội dung (title, hook, script, cta, hashtags, caption) phải viết hoàn toàn bằng TIẾNG VIỆT, phù hợp với khán giả Việt Nam.

Chỉ trả về một JSON object (không markdown, không giải thích, không code fence) với cấu trúc CHÍNH XÁC này:
{
  "title": "tiêu đề video hấp dẫn (tối đa 80 ký tự, tiếng Việt)",
  "hook": "câu mở đầu 10-15 giây — một sự thật bất ngờ, tuyên bố mạnh mẽ hoặc câu hỏi khiến người xem phải ở lại xem",
  "script": "toàn bộ văn bản thuyết minh, ~%d từ, viết như lời nói tự nhiên liên tục — xem cấu trúc bên dưới",
  "cta": "lời kêu gọi hành động cuối video (ví dụ: 'Theo dõi kênh để không bỏ lỡ video tiếp theo nhé!')",
  "hashtags": ["tag1", "tag2", "tag3", "tag4", "tag5", "tag6", "tag7", "tag8"],
  "caption": "caption đăng bài với emoji (tối đa 250 ký tự, tiếng Việt)"
}

Kịch bản PHẢI theo cấu trúc này (viết thành một bài thuyết minh liên tục, bao gồm đủ mọi phần):
1. MỞ ĐẦU (~30 giây): Một tuyên bố, câu hỏi hoặc thống kê bất ngờ thu hút sự chú ý ngay lập tức.
2. GIỚI THIỆU (~45 giây): Tại sao chủ đề này quan trọng HIỆN NAY và người xem sẽ học được gì.
3. CHƯƠNG 1 — Bối cảnh / Lịch sử (~60 giây): Dẫn dắt vào chủ đề với lịch sử, định nghĩa hoặc bức tranh toàn cảnh.
4. CHƯƠNG 2 — Phân tích Cốt lõi (~90 giây): Ý tưởng chính, được phân tích với ví dụ cụ thể và so sánh dễ hiểu.
5. CHƯƠNG 3 — Ví dụ Thực tế (~60 giây): 2-3 câu chuyện, trường hợp thực tế hoặc case study cụ thể.
6. CHƯƠNG 4 — Ý nghĩa với Bạn (~45 giây): Người xem có thể áp dụng hoặc được lợi gì từ thông tin này.
7. KẾT LUẬN + CTA (~30 giây): Tóm tắt điểm chính và kết thúc bằng lời kêu gọi hành động mạnh mẽ.

Quy tắc:
- Tổng độ dài khoảng %d từ (mục tiêu = %d giây ở tốc độ 120 từ/phút)
- Dùng ngôn ngữ thân thiện, sinh động, ngôi thứ hai ("bạn sẽ khám phá", "hãy tưởng tượng")
- Đặt câu hỏi tu từ để giữ người xem tập trung
- Dùng số liệu, tên người/tổ chức, ví dụ cụ thể — tránh nói chung chung
- Mỗi chương chuyển tiếp tự nhiên sang chương tiếp theo
- KHÔNG có chỉ dẫn quay phim, KHÔNG có "[B-roll]" — chỉ lời thuyết minh
- 6-10 hashtag liên quan, phù hợp với Việt Nam (không có ký tự #)
- Caption phải thu hút người lướt feed, dùng 2-4 emoji`,
			platform, durationSecs, wordTarget, req.Topic, niche, nicheGuidanceVI(niche), wordTarget, wordTarget, durationSecs)
	}

	// Average English narration ≈ 150 words/min
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
		platform, durationSecs, wordTarget, req.Topic, niche, wordTarget, wordTarget, durationSecs)
}

// nicheGuidanceVI injects extra, niche-aware guidance for the MMO / make-money-online
// vertical when the niche signals money/finance topics. Returns "" otherwise so the
// generic structure applies. Kept in code because it is tightly coupled to the prompt.
func nicheGuidanceVI(niche string) string {
	n := strings.ToLower(niche)
	moneyKeywords := []string{"mmo", "kiếm tiền", "kiem tien", "tài chính", "tai chinh", "đầu tư", "dau tu", "affiliate", "kinh doanh", "online", "thu nhập", "thu nhap", "làm giàu", "lam giau"}
	isMoney := false
	for _, k := range moneyKeywords {
		if strings.Contains(n, k) {
			isMoney = true
			break
		}
	}
	if !isMoney {
		return ""
	}
	return `
ĐỊNH HƯỚNG NICHE (KIẾM TIỀN ONLINE / MMO):
- HOOK nên dùng con số thu nhập cụ thể, đáng tin (ví dụ "cách tôi kiếm thêm 15 triệu/tháng" hoặc "phá vỡ 3 lầm tưởng khiến bạn mãi không kiếm được tiền online").
- Nội dung thực dụng: nêu phương pháp/công cụ cụ thể, bước làm rõ ràng, rủi ro & chi phí thật — TRÁNH hứa hẹn làm giàu nhanh phi thực tế.
- CTA hướng người xem theo dõi kênh và bình luận một từ khóa để nhận thêm tài liệu/hướng dẫn.
- Hashtags ưu tiên: kiemtienonline, MMO, affiliate, kiemtientainha, taichinhcanhan, lamgiau, kinhdoanhonline.
- Caption nên có một câu disclaimer nhẹ (kết quả tùy nỗ lực từng người).
`
}
