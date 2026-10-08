package domain

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// ReviewInterval khoảng cách kiểm duyệt toàn cục (kích hoạt mỗi N chương).
const ReviewInterval = 5

// ShouldReview kiểm tra có cần kiểm duyệt toàn cục hay không dựa trên số chương đã hoàn thành (chế độ ngắn/trung).
func ShouldReview(completedCount int) (bool, string) {
	if completedCount > 0 && completedCount%ReviewInterval == 0 {
		return true, fmt.Sprintf("Đã hoàn thành %d chương, kích hoạt kiểm duyệt toàn cục", completedCount)
	}
	return false, ""
}

// ShouldArcReview kiểm tra có cần đánh giá cấp cung truyện/tập hay không trong chế độ dài.
func ShouldArcReview(isArcEnd, isVolumeEnd bool, volume, arc int) (bool, string) {
	if isVolumeEnd {
		return true, fmt.Sprintf("Tập %d cung truyện %d kết thúc (kết thúc tập), kích hoạt đánh giá cấp cung truyện + cấp tập", volume, arc)
	}
	if isArcEnd {
		return true, fmt.Sprintf("Tập %d cung truyện %d kết thúc, kích hoạt đánh giá cấp cung truyện", volume, arc)
	}
	return false, ""
}

// WordCount đếm "số chữ" của chính văn — đơn vị mà chapter_words và mọi mục tiêu độ dài dùng.
//   - Tiếng Trung (Hán tự chiếm ưu thế): đếm theo rune như quy ước 字数 (giữ nguyên số liệu cũ).
//   - Tiếng Việt / chữ Latin: đếm theo âm tiết (cụm chữ/số liền nhau). Một Hán tự tương ứng
//     khoảng một âm tiết, nên ngưỡng 3000-6000 giữ đúng độ dài chương dự định; đếm theo
//     ký tự sẽ khiến chương tiếng Việt ngắn đi khoảng 5 lần.
func WordCount(content string) int {
	if IsHanDominant(content) {
		return utf8.RuneCountInString(content)
	}
	n := 0
	inWord := false
	for _, r := range content {
		isWord := unicode.IsLetter(r) || unicode.IsDigit(r) || (inWord && unicode.Is(unicode.Mn, r))
		if isWord && (!inWord || unicode.Is(unicode.Han, r)) {
			n++
		}
		inWord = isWord
	}
	return n
}

// IsHanDominant trả về true khi văn bản là tiếng Trung: số Hán tự nhiều hơn số từ Latin
// (đếm theo từ, vì một từ "pattern" tương đương một chữ Hán chứ không phải bảy).
func IsHanDominant(text string) bool {
	han, latinWords := 0, 0
	inLatin := false
	for _, r := range text {
		isLatin := unicode.Is(unicode.Latin, r) || (inLatin && unicode.Is(unicode.Mn, r))
		if isLatin && !inLatin {
			latinWords++
		}
		inLatin = isLatin
		if unicode.Is(unicode.Han, r) {
			han++
		}
	}
	return han > latinWords
}
