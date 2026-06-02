// Package mockai is an explicit, deterministic ScriptGenerator used for local
// development, hermetic end-to-end tests (AI_PROVIDER=mock), and as the fallback
// provider when the primary LLM is unavailable. It never makes network calls.
package mockai

import (
	"context"
	"strings"

	"mmo/internal/domain/ai"
)

type Generator struct{}

var _ ai.ScriptGenerator = (*Generator)(nil)

func New() *Generator { return &Generator{} }

func (g *Generator) GenerateScript(ctx context.Context, req ai.ScriptRequest) (*ai.ScriptResult, error) {
	if req.Language == "vi" {
		return mockScriptVI(req.Topic), nil
	}
	return mockScriptEN(req.Topic), nil
}

// build assembles a ScriptResult from scenes, deriving Script (joined narration)
// and VisualKeywords (the scene queries) so both code paths stay consistent.
func build(title, hook, cta, caption string, hashtags []string, scenes []ai.Scene) *ai.ScriptResult {
	parts := make([]string, 0, len(scenes))
	kws := make([]string, 0, len(scenes))
	for _, s := range scenes {
		parts = append(parts, s.Narration)
		kws = append(kws, s.VisualQuery)
	}
	return &ai.ScriptResult{
		Title:          title,
		Hook:           hook,
		Script:         strings.Join(parts, " "),
		CTA:            cta,
		Hashtags:       hashtags,
		Caption:        caption,
		VisualKeywords: kws,
		Scenes:         scenes,
	}
}

func mockScriptVI(topic string) *ai.ScriptResult {
	scenes := []ai.Scene{
		{Narration: "Phần lớn mọi người hiểu sai hoàn toàn về " + topic + ". Trong vài phút tới, bạn sẽ thấy một góc nhìn khác hẳn — và mang về vài ý tưởng áp dụng được ngay tuần này.", VisualQuery: "person thinking looking at laptop", Keyword: "Mở đầu"},
		{Narration: "Vì sao điều này quan trọng ngay lúc này? Bởi thị trường đang thay đổi nhanh, và người nắm bắt sớm sẽ có lợi thế rõ rệt so với người đứng ngoài.", VisualQuery: "busy city street time lapse", Keyword: "Vì sao quan trọng"},
		{Narration: topic + " không xuất hiện từ con số không. Nó là kết quả của nhiều năm tích lũy âm thầm, từ những người kiên trì thử nghiệm và học hỏi mỗi ngày.", VisualQuery: "hands writing notes notebook desk", Keyword: "Bối cảnh"},
		{Narration: "Đây là ý tưởng cốt lõi: đừng tìm cách làm giàu nhanh. Hãy xây một hệ thống nhỏ, đo lường kết quả, rồi tối ưu dần. Kỷ luật quan trọng hơn may mắn.", VisualQuery: "financial charts growth on screen", Keyword: "Cốt lõi"},
		{Narration: "Lấy ví dụ thực tế: một người bắt đầu với mười lăm phút mỗi ngày, chọn đúng một công cụ, kiên trì ba tháng — và tạo ra nguồn thu nhập phụ ổn định đầu tiên.", VisualQuery: "person working laptop home office", Keyword: "Ví dụ thực tế"},
		{Narration: "Vậy bạn nên làm gì? Bước một: học mỗi ngày một chút. Bước hai: bắt tay làm thật. Bước ba: đo lường và điều chỉnh. Nếu thấy hữu ích, hãy theo dõi kênh để xem video tiếp theo nhé!", VisualQuery: "counting cash money success", Keyword: "Hành động"},
	}
	return build(
		"Sự Thật Đằng Sau "+topic+" Mà Ít Ai Biết",
		"Phần lớn mọi người không biết điều thực sự đang xảy ra với "+topic+" — và khi thấy rồi, bạn không thể không nhìn thấy nó.",
		"Theo dõi kênh để không bỏ lỡ video tiếp theo nhé!",
		"🎯 Toàn bộ câu chuyện về "+topic+" — điều ít ai kể cho bạn 👇 (kết quả tùy nỗ lực mỗi người)",
		[]string{"kienthuc", "kiemtienonline", "MMO", "taichinhcanhan", "khampha", "lamgiau"},
		scenes,
	)
}

func mockScriptEN(topic string) *ai.ScriptResult {
	scenes := []ai.Scene{
		{Narration: "Most people get " + topic + " completely wrong. In the next few minutes you'll see a different angle — and walk away with ideas you can use this week.", VisualQuery: "person thinking looking at laptop", Keyword: "Intro"},
		{Narration: "Why does this matter right now? The market is shifting fast, and the people who move early get a real edge over everyone waiting on the sidelines.", VisualQuery: "busy city street time lapse", Keyword: "Why now"},
		{Narration: topic + " did not appear from nothing. It's the product of years of quiet, consistent work by people who kept testing and learning every day.", VisualQuery: "hands writing notes notebook desk", Keyword: "Background"},
		{Narration: "Here's the core idea: stop chasing get-rich-quick. Build a small system, measure results, then optimize. Discipline beats luck every time.", VisualQuery: "financial charts growth on screen", Keyword: "Core idea"},
		{Narration: "A real example: someone starts with fifteen minutes a day, picks one tool, stays consistent for three months — and builds their first steady side income.", VisualQuery: "person working laptop home office", Keyword: "Example"},
		{Narration: "So what should you do? Step one: learn a little daily. Step two: actually start. Step three: measure and adjust. If this helped, follow for the next one!", VisualQuery: "counting cash money success", Keyword: "Action"},
	}
	return build(
		"The Real Story Behind "+topic,
		"Most people have no idea what is really happening with "+topic+" right now — and once you see it, you cannot unsee it.",
		"Follow for more deep dives like this every week!",
		"🎯 The full story behind "+topic+" — what nobody is telling you 👇",
		[]string{"deepdive", "explainer", "trending", "viral", "education", "insights"},
		scenes,
	)
}
