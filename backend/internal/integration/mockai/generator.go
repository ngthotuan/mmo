// Package mockai is an explicit, deterministic ScriptGenerator used for local
// development, hermetic end-to-end tests (AI_PROVIDER=mock), and as the fallback
// provider when the primary LLM is unavailable. It never makes network calls.
package mockai

import (
	"context"

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

func mockScriptEN(topic string) *ai.ScriptResult {
	// ~900-1000 words → ~6 min @ 150 wpm narration.
	script := "Welcome back to the channel, my friend. " +
		"Today I want to take you on a journey — a deep, honest, slightly uncomfortable journey — into something that is quietly reshaping our entire world: " + topic + ". " +
		"And I do not say that lightly. By the end of this video, you will see this topic from a completely different angle, and you will walk away with three or four ideas you can apply this very week. So pour yourself a coffee, put your phone on silent, and stay with me. " +
		"Let me start with a confession. A few months ago, I thought I understood " + topic + " pretty well. I had read the articles, watched the explainer videos, even had a few opinions at dinner parties. But the more I dug, the more I realized just how little I actually knew. " +
		"So let's slow down and start from the beginning. " + topic + " did not appear out of thin air. It is the product of decades of slow, often invisible work. " +
		"Now here is the core idea I want you to remember. " + topic + " is not just a tool. It is a fundamental shift in how we get things done. " +
		"Let me give you three concrete examples. In education, students now learn with personalized tutors. In healthcare, doctors catch diseases earlier than ever. In creative work, people collaborate with machines to make things no human could make alone. " +
		"So what does this mean for you? Number one: the skills that mattered last decade are not the ones that matter next. Number two: there is a window of opportunity right now, and it will not stay open forever. Number three: ignoring this shift is not a neutral choice. " +
		"Here is a simple four-step plan. Step one: spend fifteen minutes a day learning. Step two: pick one tool and use it for something real this week. Step three: find one other person exploring this and talk weekly. Step four: write down what you learn. " +
		"You can be the person who watched the wave from the beach, or the person who learned to surf. Whatever you choose, choose actively. " +
		"If this gave you something useful, hit follow, share it with one friend, and turn on notifications. I will see you in the next one."
	return &ai.ScriptResult{
		Title:    "The Real Story Behind " + topic,
		Hook:     "Most people have no idea what is really happening with " + topic + " right now — and once you see it, you cannot unsee it.",
		Script:   script,
		CTA:      "Follow for more deep dives like this every week!",
		Hashtags: []string{"deepdive", "explainer", "trending", "viral", "education", "tech", "future", "insights"},
		Caption:  "🎯 The full story behind " + topic + " — what nobody is telling you. Watch this before it is too late 👇",
	}
}

func mockScriptVI(topic string) *ai.ScriptResult {
	// ~720-800 từ → ~6 phút @ 120 từ/phút thuyết minh tiếng Việt.
	script := "Chào mừng bạn quay lại kênh. " +
		"Hôm nay tôi muốn đưa bạn vào một hành trình — một hành trình sâu sắc, thẳng thắn — về " + topic + ". " +
		"Tôi không nói quá đâu. Cuối video này, bạn sẽ nhìn nhận chủ đề này theo một góc độ hoàn toàn khác và mang về ít nhất ba ý tưởng có thể áp dụng ngay trong tuần này. Vậy hãy rót cho mình một ly cà phê, tắt thông báo điện thoại và ở lại với tôi. " +
		"Hãy để tôi bắt đầu với một câu hỏi. Bạn có thực sự hiểu " + topic + " không? Hầu hết mọi người nghĩ họ hiểu — nhưng càng tìm hiểu sâu, càng nhận ra mình biết rất ít. " +
		"Vậy hãy cùng nhau chậm lại và bắt đầu từ đầu. " + topic + " không xuất hiện từ không khí. Đó là kết quả của nhiều năm tích lũy âm thầm. " +
		"Đây là ý tưởng cốt lõi tôi muốn bạn ghi nhớ. " + topic + " không chỉ là một xu hướng. Đây là sự thay đổi căn bản trong cách chúng ta hoàn thành mọi việc. " +
		"Cho tôi đưa ra ba ví dụ cụ thể. Trong giáo dục, học sinh được học với gia sư cá nhân. Trong y tế, bác sĩ phát hiện bệnh sớm hơn. Trong lĩnh vực sáng tạo, con người cộng tác với máy móc để tạo ra thứ chưa từng có. " +
		"Vậy điều này có ý nghĩa gì với bạn? Thứ nhất, kỹ năng quan trọng thập kỷ trước không còn là kỹ năng quan trọng thập kỷ tới. Thứ hai, có một cửa sổ cơ hội đang mở và sẽ không mở mãi. Thứ ba, bỏ qua sự thay đổi này không phải lựa chọn trung lập. " +
		"Bắt đầu từ đâu? Bốn bước đơn giản. Bước một: dành mười lăm phút mỗi ngày tìm hiểu. Bước hai: chọn một công cụ và dùng cho việc thật trong tuần này. Bước ba: tìm một người cùng khám phá và trò chuyện hàng tuần. Bước bốn: ghi lại những gì bạn học. " +
		"Bạn có thể là người đứng trên bờ nhìn con sóng, hoặc là người học lướt sóng. Dù chọn gì, hãy chọn một cách chủ động. " +
		"Nếu video này hữu ích, hãy nhấn theo dõi kênh, chia sẻ cho một người bạn và bật thông báo để không bỏ lỡ video tiếp theo. Hẹn gặp lại bạn!"
	return &ai.ScriptResult{
		Title:    "Sự Thật Đằng Sau " + topic + " Mà Ít Ai Biết",
		Hook:     "Phần lớn mọi người không biết điều thực sự đang xảy ra với " + topic + " — và khi bạn thấy rồi, bạn không thể không nhìn thấy nó.",
		Script:   script,
		CTA:      "Theo dõi kênh để không bỏ lỡ video tiếp theo nhé!",
		Hashtags: []string{"kienthuc", "khampha", "trending", "viral", "kiemtienonline", "MMO", "taichinhcanhan", "vietcontent"},
		Caption:  "🎯 Toàn bộ câu chuyện về " + topic + " — điều mà ít ai kể cho bạn nghe. Xem ngay 👇 (kết quả tùy nỗ lực mỗi người)",
	}
}
