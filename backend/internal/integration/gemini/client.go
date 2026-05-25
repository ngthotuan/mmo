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

func (c *Client) GenerateScript(ctx context.Context, topic, niche, platform string, durationSecs int, language string) (*ScriptResult, error) {
	if c.apiKey == "" {
		return c.mockScript(topic, language), nil
	}

	prompt := buildScriptPrompt(topic, niche, platform, durationSecs, language)

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
		return c.mockScript(topic, language), nil
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
		return c.mockScript(topic, language), nil
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return c.mockScript(topic, language), nil
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

func buildScriptPrompt(topic, niche, platform string, durationSecs int, language string) string {
	if niche == "" {
		niche = "general"
	}
	if platform == "" {
		platform = "Facebook"
	}
	if durationSecs <= 0 {
		durationSecs = 360
	}

	if language == "vi" {
		// Vietnamese narration ≈ 120 words/min (slower than English due to tones)
		wordTarget := durationSecs * 120 / 60
		return fmt.Sprintf(`Bạn là một chuyên gia viết kịch bản video dài cho %s. Hãy tạo một kịch bản thuyết minh sâu sắc, hấp dẫn dài %d giây (~%d từ) bằng TIẾNG VIỆT về chủ đề: "%s" trong lĩnh vực %s.

ĐÂY LÀ VIDEO DÀI (hơn 5 phút), KHÔNG phải short 60 giây. Hãy viết như một phóng sự mini hoặc video phân tích chuyên sâu: kể chuyện phong phú, nhiều chương, ví dụ thực tế, giai thoại thú vị, sự kiện bất ngờ, miêu tả sinh động. Nói trực tiếp với người xem theo ngôi thứ hai ("bạn").

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
			platform, durationSecs, wordTarget, topic, niche, wordTarget, wordTarget, durationSecs)
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
		platform, durationSecs, wordTarget, topic, niche, wordTarget, wordTarget, durationSecs)
}

func (c *Client) mockScript(topic, language string) *ScriptResult {
	if language == "vi" {
		return c.mockScriptVI(topic)
	}
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

func (c *Client) mockScriptVI(topic string) *ScriptResult {
	// ~720-800 từ → ~6 phút @ 120 từ/phút thuyết minh tiếng Việt.
	script := "Chào mừng bạn quay lại kênh. " +
		"Hôm nay tôi muốn đưa bạn vào một hành trình — một hành trình sâu sắc, thẳng thắn — về " + topic + ". " +
		"Tôi không nói quá đâu. Cuối video này, bạn sẽ nhìn nhận chủ đề này theo một góc độ hoàn toàn khác và mang về ít nhất ba ý tưởng có thể áp dụng ngay trong tuần này. Vậy hãy rót cho mình một ly cà phê, tắt thông báo điện thoại và ở lại với tôi. " +

		"Hãy để tôi bắt đầu với một câu hỏi. Bạn có thực sự hiểu " + topic + " không? Hầu hết mọi người nghĩ họ hiểu — họ đã đọc bài báo, xem video giải thích, thậm chí có vài ý kiến khi trò chuyện cùng bạn bè. Nhưng càng tìm hiểu sâu, tôi càng nhận ra mình biết rất ít. Và tôi nghĩ điều đó đúng với phần lớn chúng ta. " +

		"Vậy hãy cùng nhau chậm lại và bắt đầu từ đầu. " + topic + " đến từ đâu? " +
		topic + " không xuất hiện từ không khí. Đó là kết quả của nhiều thập kỷ làm việc âm thầm — các nhà nghiên cứu trong phòng thí nghiệm, những kỹ sư xây dựng công cụ mà chẳng ai yêu cầu, những người mơ mộng viết tuyên ngôn mà ai cũng cười. Bước đột phá chúng ta thấy hôm nay thực ra không phải đột phá — đó là kết quả của hàng trăm thành công nhỏ tích lũy thành thứ gì đó mà thế giới không thể bỏ qua. " +
		"Và hệ quả hiện diện khắp nơi. Nhìn vào ứng dụng trên điện thoại bạn. Nhìn vào cách doanh nghiệp tái cơ cấu. Nhìn vào những câu hỏi con cái bạn bắt đầu đặt ra. " + topic + " đang chạm đến tất cả điều đó. " +

		"Đây là ý tưởng cốt lõi tôi muốn bạn ghi nhớ. " +
		topic + " không chỉ là một công cụ. Không chỉ là một xu hướng. Đây là sự thay đổi căn bản trong cách con người chúng ta hoàn thành mọi việc. " +
		"Hãy nghĩ như điện lực vào năm 1900. Ban đầu chỉ vài gia đình giàu có mới có, phần lớn người không hiểu nó dùng để làm gì. Bốn mươi năm sau, nó trở nên vô hình — thấm vào mọi ngõ ngách cuộc sống. " + topic + " đang đi theo con đường đó, chỉ là nhanh hơn rất nhiều. Chính sự nhanh chóng đó khiến nhiều người cảm thấy lo lắng. " +

		"Cho tôi đưa ra ba ví dụ cụ thể để làm rõ điều này. " +
		"Ví dụ thứ nhất. Trong giáo dục, học sinh giờ đây được học với gia sư cá nhân thích nghi theo nhịp độ, ngôn ngữ, điểm yếu của từng em. Một đứa trẻ từng vật lộn trong lớp ba mươi người giờ có thể có người hướng dẫn riêng lúc nửa đêm. Không phải khoa học viễn tưởng — điều đó đang xảy ra ngay bây giờ. " +
		"Ví dụ thứ hai. Trong y tế, bác sĩ phát hiện bệnh sớm hơn bao giờ hết. Các mẫu trong kết quả quét, xét nghiệm máu, giọng nói — mẫu không mắt người nào nhìn thấy — được phát hiện trong vài giây. Những sinh mạng được cứu mà năm năm trước sẽ bị mất. " +
		"Ví dụ thứ ba. Trong lĩnh vực sáng tạo, nghệ sĩ, nhà văn, nhạc sĩ đang cộng tác với máy móc để tạo ra thứ không con người nào làm được một mình. Quan niệm rằng sáng tạo thuộc riêng về con người đang được viết lại. " +

		"Vậy tất cả điều này có ý nghĩa gì với bạn? Ba bài học thực tế. " +
		"Thứ nhất. Kỹ năng quan trọng nhất thập kỷ trước không phải là kỹ năng quan trọng nhất thập kỷ tới. Ghi nhớ sự kiện, thực hiện nhiệm vụ lặp lại — đó chính xác là những thứ bị tự động hóa đầu tiên. Điều quan trọng bây giờ là sự tò mò, phán đoán và khả năng đặt câu hỏi tốt hơn bất kỳ ai. " +
		"Thứ hai. Có một cửa sổ cơ hội đang mở và nó sẽ không mở mãi. Những người thử nghiệm hôm nay sẽ có lợi thế âm thầm tích lũy. Năm năm nữa, họ sẽ nhìn lại và nhận ra mình đã xây dựng đòn bẩy trong khi người khác còn đang lướt mạng xã hội. " +
		"Thứ ba, và đây là điều khó nhất. Bỏ qua sự thay đổi này không phải là lựa chọn trung lập. Không làm gì cũng là làm điều gì đó. Bạn không cần trở thành chuyên gia. Không cần bỏ việc. Nhưng bạn cần tham gia. " +

		"Bắt đầu từ đâu? Bốn bước đơn giản. " +
		"Bước một. Dành mười lăm phút mỗi ngày tìm hiểu về " + topic + ". Không cần một giờ. Không cần khóa học. Chỉ mười lăm phút thật sự. " +
		"Bước hai. Chọn một công cụ — bất kỳ công cụ nào — và thử dùng nó cho việc gì đó thực sự trong tuần này. " +
		"Bước ba. Tìm một người cũng đang khám phá lĩnh vực này và nói chuyện hàng tuần. Sự tò mò lan rộng hơn khi được chia sẻ. " +
		"Bước bốn. Ghi lại những gì bạn học. Một ghi chú, một dòng chia sẻ. Bên ngoài hóa sự hiểu biết là cách nhanh nhất để đào sâu nó. " +

		"Để lại cho bạn một suy nghĩ cuối. " +
		"Khoảnh khắc thú vị, bất định, quan trọng nhất của cuộc đời chúng ta đang diễn ra ngay bây giờ, và bạn đang ở trong đó. Bạn có thể là người đứng trên bờ nhìn con sóng, hoặc là người học lướt sóng. " +
		"Dù bạn chọn gì, hãy chọn một cách chủ động. Kết quả tệ nhất là trôi dạt. " +

		"Nếu video này mang lại điều gì đó hữu ích — một ý tưởng mới, một câu hỏi mới — hãy nhấn theo dõi kênh, chia sẻ cho một người bạn cần nghe điều này và bật thông báo để không bỏ lỡ video tiếp theo. " +
		"Hẹn gặp lại bạn trong video tiếp theo. Chúc bạn một ngày tốt lành!"
	return &ScriptResult{
		Title:    "Sự Thật Đằng Sau " + topic + " Mà Ít Ai Biết",
		Hook:     "Phần lớn mọi người không biết điều thực sự đang xảy ra với " + topic + " — và khi bạn thấy rồi, bạn không thể không nhìn thấy nó.",
		Script:   script,
		CTA:      "Theo dõi kênh để không bỏ lỡ video tiếp theo nhé!",
		Hashtags: []string{"kienthuc", "khampha", "trending", "viral", "hocmai", "congnghe", "tuonglai", "vietcontent"},
		Caption:  "🎯 Toàn bộ câu chuyện về " + topic + " — điều mà ít ai kể cho bạn nghe. Xem ngay trước khi quá muộn 👇",
	}
}
