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
	finalizeResult(&script)
	if strings.TrimSpace(script.Script) == "" {
		return nil, fmt.Errorf("gemini: empty script body")
	}
	return &script, nil
}

// finalizeResult fills derived fields from the scene breakdown so the rest of
// the pipeline (and the fallback single-script path) always has Script +
// VisualKeywords populated, regardless of which fields the model returned.
func finalizeResult(s *ai.ScriptResult) {
	if len(s.Scenes) == 0 {
		return
	}
	if strings.TrimSpace(s.Script) == "" {
		parts := make([]string, 0, len(s.Scenes))
		for _, sc := range s.Scenes {
			if t := strings.TrimSpace(sc.Narration); t != "" {
				parts = append(parts, t)
			}
		}
		s.Script = strings.Join(parts, " ")
	}
	if len(s.VisualKeywords) == 0 {
		for _, sc := range s.Scenes {
			if q := strings.TrimSpace(sc.VisualQuery); q != "" {
				s.VisualKeywords = append(s.VisualKeywords, q)
			}
		}
	}
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
		sceneCount := durationSecs / 25 // ~25s of narration per scene
		if sceneCount < 8 {
			sceneCount = 8
		}
		return fmt.Sprintf(`Bạn là chuyên gia sản xuất video dạng phóng sự/giải thích chuyên sâu cho %s. Hãy xây dựng một video TIẾNG VIỆT dài ~%d giây (~%d từ thuyết minh) về chủ đề: "%s" trong lĩnh vực %s.

Video được chia thành NHIỀU CẢNH (scene) nối tiếp nhau thành MỘT mạch nội dung thống nhất, sâu sắc, hấp dẫn — như một mini-documentary. MỖI CẢNH gồm: lời thuyết minh cho cảnh đó + từ khoá hình ảnh BẰNG TIẾNG ANH để tìm video minh hoạ ĐÚNG với nội dung của chính cảnh đó (vì kho video stock dùng tiếng Anh).
%s
TOÀN BỘ chữ tiếng Việt (trừ visual_query là tiếng Anh). Chỉ trả về MỘT JSON object (không markdown, không code fence) đúng cấu trúc:
{
  "title": "tiêu đề hấp dẫn (tối đa 80 ký tự)",
  "hook": "câu mở đầu 10-15 giây gây tò mò/sốc khiến người xem ở lại",
  "cta": "lời kêu gọi hành động cuối video",
  "hashtags": ["tag1","tag2","tag3","tag4","tag5","tag6","tag7","tag8"],
  "caption": "caption đăng bài kèm 2-4 emoji (tối đa 250 ký tự)",
  "scenes": [
    { "narration": "2-4 câu thuyết minh cho cảnh này (tiếng Việt)", "visual_query": "english stock footage search describing THIS scene's visuals", "keyword": "nhãn ngắn 1-3 từ tiếng Việt cho cảnh" }
  ]
}

YÊU CẦU VỀ SCENES:
- Tạo %d-%d cảnh, nối tiếp theo cấu trúc: (1) mở đầu gây chú ý → (2) vì sao quan trọng hiện nay → (3) bối cảnh/định nghĩa → (4) phân tích cốt lõi với ví dụ → (5) 2-3 case/câu chuyện thực tế → (6) cách áp dụng cho người xem → (7) kết luận + CTA.
- TỔNG narration khoảng %d từ (mục tiêu %d giây ở 120 từ/phút). Mỗi cảnh nói MỘT ý, KHÔNG lặp lại ý/câu/cụm đã nói; mạch phát triển liên tục, nhất quán từ đầu đến cuối.
- visual_query PHẢI mô tả đúng hình ảnh cho nội dung cảnh đó, cụ thể & dễ tìm trên Pexels/Pixabay. VD cảnh nói về làm việc tại nhà → "person working laptop home office"; cảnh nói về tiền/thu nhập → "counting cash money".
- Dùng số liệu, ví dụ cụ thể, ngôi thứ hai ("bạn"), câu hỏi tu từ. KHÔNG chèn chỉ dẫn quay phim trong narration.
- hashtags: 6-10 từ hợp Việt Nam, không có ký tự '#'.`,
			platform, durationSecs, wordTarget, req.Topic, niche, nicheGuidanceVI(niche), sceneCount, sceneCount+4, wordTarget, durationSecs)
	}

	// Average English narration ≈ 150 words/min
	wordTarget := durationSecs * 150 / 60
	sceneCount := durationSecs / 25
	if sceneCount < 8 {
		sceneCount = 8
	}
	return fmt.Sprintf(`You are an expert mini-documentary video producer for %s. Build an in-depth, engaging %d-second (~%d words) video in English about: "%s" in the %s niche.

The video is split into SCENES that connect into ONE coherent, deep, engaging thread. EACH SCENE has: the narration spoken during it + an ENGLISH stock-footage search query describing the visuals for THAT scene (so footage matches the content).

Output ONLY a JSON object (no markdown, no code fences) with this structure:
{
  "title": "catchy title (max 80 chars)",
  "hook": "10-15s opening that makes the viewer commit",
  "cta": "call to action",
  "hashtags": ["tag1","tag2","tag3","tag4","tag5","tag6","tag7","tag8"],
  "caption": "post caption with 2-4 emojis (max 250 chars)",
  "scenes": [
    { "narration": "2-4 sentences spoken in this scene", "visual_query": "english stock footage search for THIS scene's visuals", "keyword": "short 1-3 word on-screen label" }
  ]
}

SCENE RULES:
- Create %d-%d scenes following: (1) attention hook → (2) why it matters now → (3) background → (4) core analysis with examples → (5) 2-3 real cases → (6) how the viewer applies it → (7) conclusion + CTA.
- TOTAL narration ~%d words (target %d seconds at 150 wpm). Each scene = ONE new idea, never repeat ideas/sentences; one continuous, consistent thread.
- visual_query MUST match that scene's content, concrete and easy to find on Pexels/Pixabay (e.g. "person working laptop home office", "counting cash money").
- Use concrete numbers/examples, second person, rhetorical questions. No camera directions in narration.
- 6-10 hashtags, no '#'.`,
		platform, durationSecs, wordTarget, req.Topic, niche, sceneCount, sceneCount+4, wordTarget, durationSecs)
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
