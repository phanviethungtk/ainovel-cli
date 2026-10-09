package rules

import (
	"regexp"
	"strings"

	"github.com/voocel/ainovel-cli/internal/domain"
)

// Lint kiểm tra đường đáy tích hợp sẵn: quét phần chính văn tìm tàn dư cơ chế,
// không liên quan đến quy tắc người dùng, luôn thực thi khi lưu chương.
// Cùng hợp đồng với Check — chỉ trả về sự thật (nguyên tắc sắt số một),
// không chặn luồng xử lý, để bộ phận đánh giá/người dùng phán quyết.
//
// Hiện có ba loại (toàn bộ từ lỗi thực chứng của sản phẩm chạy dài thực tế):
//   - markdown_residue: chính văn còn sót ** in đậm, dòng tiêu đề # ngoài dòng đầu (xuất txt sẽ lộ ký tự)
//   - non_cjk_fragments: chính văn tiếng Trung lẫn đoạn ký tự Latin (ví dụ "pattern")
//   - cjk_fragments: chính văn tiếng Việt lẫn Hán tự (mô hình trôi về tiếng Trung)
//
// Hai loại trộn ngôn ngữ loại trừ nhau: ngôn ngữ chính được xác định theo số ký tự Hán so với chữ Latin.
func Lint(text string) []Violation {
	var vs []Violation
	vs = appendMarkdownResidue(vs, text)
	if domain.IsHanDominant(text) {
		vs = appendNonCJKFragments(vs, text)
	} else {
		vs = appendCJKFragments(vs, text)
	}
	return vs
}

func appendMarkdownResidue(vs []Violation, text string) []Violation {
	if n := strings.Count(text, "**"); n > 0 {
		vs = append(vs, Violation{
			Rule:     "markdown_residue",
			Target:   "**",
			Actual:   n,
			Severity: SeverityWarning,
		})
	}
	headings := 0
	seenContent := false
	for line := range strings.SplitSeq(text, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		// Tiêu đề # ở dòng không trống đầu tiên là định dạng hợp lệ của file chương (không cố định số dòng, chấp nhận dòng trống dẫn đầu)
		first := !seenContent
		seenContent = true
		if !first && strings.HasPrefix(t, "#") {
			headings++
		}
	}
	if headings > 0 {
		vs = append(vs, Violation{
			Rule:     "markdown_residue",
			Target:   "#",
			Actual:   headings,
			Severity: SeverityWarning,
		})
	}
	return vs
}

var (
	latinFragmentRe = regexp.MustCompile(`[A-Za-z]{2,}`)
	hanFragmentRe   = regexp.MustCompile(`\p{Han}+`)
)

// appendNonCJKFragments báo cáo tổng số lần xuất hiện đoạn ký tự Latin và các ví dụ đã loại trùng.
// Tiếng Anh hợp lệ của thể loại hiện đại (tên thương hiệu/viết tắt) cũng sẽ bị phát hiện — sự thật mức warning, để bộ phận đánh giá phán quyết theo thể loại.
func appendNonCJKFragments(vs []Violation, text string) []Violation {
	return appendFragments(vs, "non_cjk_fragments", latinFragmentRe.FindAllString(text, -1))
}

// appendCJKFragments báo cáo Hán tự lẫn trong chính văn tiếng Việt (sự thật mức warning).
func appendCJKFragments(vs []Violation, text string) []Violation {
	return appendFragments(vs, "cjk_fragments", hanFragmentRe.FindAllString(text, -1))
}

func appendFragments(vs []Violation, rule string, matches []string) []Violation {
	if len(matches) == 0 {
		return vs
	}
	seen := make(map[string]struct{})
	var examples []string
	for _, m := range matches {
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		if len(examples) < 3 {
			examples = append(examples, m)
		}
	}
	return append(vs, Violation{
		Rule:     rule,
		Target:   strings.Join(examples, "、"),
		Actual:   len(matches),
		Severity: SeverityWarning,
	})
}
