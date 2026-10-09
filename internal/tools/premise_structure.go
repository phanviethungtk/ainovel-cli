package tools

import (
	"strings"

	"github.com/voocel/ainovel-cli/internal/domain"
)

// Tên tiêu đề chuẩn của premise — khớp đúng với danh sách trong prompts/architect-*.md.
const (
	headingGenreTone     = "Thể loại và tông điệu"
	headingGenrePosition = "Định vị thể loại"
	headingConflict      = "Xung đột cốt lõi"
	headingGoal          = "Mục tiêu nhân vật chính"
	headingEnding        = "Hướng kết cục"
	headingTaboo         = "Vùng cấm viết"
	headingSellingPoints = "Điểm bán hàng khác biệt"
	headingHook          = "Điểm móc khác biệt"
	headingPromise       = "Cam kết thực hiện cốt lõi"
	headingEngine        = "Động cơ truyện"
	headingRelationArc   = "Tuyến quan hệ/phát triển"
	headingProgression   = "Lộ trình nâng cấp"
	headingMidTurn       = "Bước ngoặt giữa chuyện"
	headingFinalQuestion = "Mệnh đề kết cục"
	headingShortFit      = "Tính phù hợp truyện ngắn"
)

// premiseHeadingAliases ánh xạ tiêu đề (đã chuẩn hoá bằng normalizePremiseHeading) về tên chuẩn.
// Gồm: tên chuẩn, các biến thể tiếng Việt từng xuất hiện trong prompt, và tên tiếng Trung
// gốc (premise của truyện tạo trước khi Việt hoá vẫn được nhận).
var premiseHeadingAliases = buildPremiseHeadingAliases(map[string][]string{
	headingGenreTone:     {"Thể loại và sắc thái", "题材和基调"},
	headingGenrePosition: {"题材定位"},
	headingConflict:      {"核心冲突"},
	headingGoal:          {"Mục tiêu của nhân vật chính", "主角目标"},
	headingEnding:        {"Hướng kết thúc", "结局方向", "终局方向"},
	headingTaboo:         {"写作禁区"},
	headingSellingPoints: {"Điểm bán khác biệt", "差异化卖点"},
	headingHook:          {"差异化钩子"},
	headingPromise:       {"核心兑现承诺"},
	headingEngine:        {"故事引擎"},
	headingRelationArc:   {"关系/成长主线"},
	headingProgression:   {"升级路径"},
	headingMidTurn:       {"中段转折", "中期转向"},
	headingFinalQuestion: {"终局命题"},
	headingShortFit: {
		"Tại sao tác phẩm này phù hợp với truyện ngắn/thu hồi đơn tập",
		"短篇适配性", "本作为什么适合短篇/单卷收束",
	},
})

func buildPremiseHeadingAliases(variants map[string][]string) map[string]string {
	m := make(map[string]string)
	for canonical, names := range variants {
		m[normalizePremiseHeading(canonical)] = canonical
		for _, name := range names {
			m[normalizePremiseHeading(name)] = canonical
		}
	}
	return m
}

// normalizePremiseHeading bỏ phần chú thích sau tên tiêu đề (prompt liệt kê dạng
// "Hướng kết cục (hướng chủ đề…)" hay "Điểm móc khác biệt: …", model đôi khi chép cả),
// rồi gộp khoảng trắng và hạ chữ thường để không phụ thuộc cách viết hoa.
func normalizePremiseHeading(title string) string {
	if i := strings.IndexAny(title, "(（:："); i >= 0 {
		title = title[:i]
	}
	return strings.ToLower(strings.Join(strings.Fields(title), " "))
}

func parsePremiseSections(premise string) map[string]string {
	lines := strings.Split(premise, "\n")
	sections := make(map[string]string)
	var current string
	var body []string

	flush := func() {
		if current == "" {
			return
		}
		text := strings.TrimSpace(strings.Join(body, "\n"))
		if text != "" {
			sections[current] = text
		}
		body = body[:0]
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if heading, ok := canonicalPremiseHeading(trimmed); ok {
			flush()
			current = heading
			continue
		}
		if current != "" {
			body = append(body, line)
		}
	}
	flush()
	return sections
}

func canonicalPremiseHeading(line string) (string, bool) {
	if !strings.HasPrefix(line, "#") {
		return "", false
	}
	title := strings.TrimSpace(strings.TrimLeft(line, "#"))
	if title == "" {
		return "", false
	}
	canonical, ok := premiseHeadingAliases[normalizePremiseHeading(title)]
	return canonical, ok
}

func premiseStructure(premise string, tier domain.PlanningTier) map[string]any {
	sections := parsePremiseSections(premise)
	required := requiredPremiseHeadings(tier)
	found := make([]string, 0, len(required))
	var missing []string
	for _, heading := range required {
		if _, ok := sections[heading]; ok {
			found = append(found, heading)
			continue
		}
		missing = append(missing, heading)
	}

	structure := map[string]any{
		"template_ready": len(missing) == 0,
		"found":          found,
		"missing":        missing,
	}
	if len(sections) > 0 {
		structure["section_count"] = len(sections)
	}
	return structure
}

func requiredPremiseHeadings(tier domain.PlanningTier) []string {
	common := []string{
		headingGenreTone,
		headingGenrePosition,
		headingConflict,
		headingGoal,
		headingEnding,
		headingTaboo,
		headingSellingPoints,
		headingHook,
		headingPromise,
	}

	switch tier {
	case domain.PlanningTierLong:
		return append(common,
			headingEngine,
			headingRelationArc,
			headingProgression,
			headingMidTurn,
			headingFinalQuestion,
		)
	case domain.PlanningTierMid:
		return append(common,
			headingEngine,
			headingMidTurn,
		)
	case domain.PlanningTierShort:
		return append(common,
			headingShortFit,
		)
	default:
		return common
	}
}
