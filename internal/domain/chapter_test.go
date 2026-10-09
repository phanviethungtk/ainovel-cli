package domain

import "testing"

func TestWordCount(t *testing.T) {
	cases := []struct {
		name string
		text string
		want int
	}{
		// Tiếng Trung: giữ quy ước 字数 theo rune (kể cả dấu câu), số liệu cũ không đổi.
		{"chinese runes", "他迈步向前。", 6},
		{"chinese with latin word", "他发现了一个pattern。", 14},
		// Tiếng Việt: đếm theo âm tiết, dấu câu không tính.
		{"vietnamese syllables", "Hắn bước về phía trước, đêm dần sâu.", 8},
		{"vietnamese with number", "Năm 2026, trời mưa.", 4},
		{"vietnamese markdown title", "# Chương 1\nHắn đi.", 4},
		{"empty", "", 0},
	}
	for _, c := range cases {
		if got := WordCount(c.text); got != c.want {
			t.Errorf("%s: WordCount(%q) = %d, want %d", c.name, c.text, got, c.want)
		}
	}
}
