package stylestat

import (
	"strings"
	"testing"
)

func chapterWith(body string) string {
	return "# 标题\n" + body
}

func TestComputeBelowMinChapters(t *testing.T) {
	in := Input{Chapters: []string{"a", "b", "c", "d"}}
	if Compute(in) != nil {
		t.Fatal("below minChapters should return nil")
	}
}

func TestComputePatterns(t *testing.T) {
	body := "他不是愤怒，而是恐惧。沉默了几息。像一盏灯。\n正文。\n"
	chapters := make([]string, 6)
	for i := range chapters {
		chapters[i] = chapterWith(body)
	}
	s := Compute(Input{Chapters: chapters})
	if s == nil {
		t.Fatal("expected stats")
	}
	want := map[string]int{
		"Câu chỉnh chuẩn『không phải… mà là…』":               6,
		"Lượng từ thời gian『một hơi thở/trong chớp mắt』":    6,
		"So sánh trực tiếp『như một/như thể/tựa như』":        6,
		"Nhịp im lặng『im lặng/không nói gì/không quay đầu』": 6,
	}
	for _, p := range s.Patterns {
		if w, ok := want[p.Name]; ok && p.Total != w {
			t.Errorf("%s total: got %d want %d", p.Name, p.Total, w)
		}
		if p.PerChapter != 1.0 {
			t.Errorf("%s per_chapter: got %v want 1.0", p.Name, p.PerChapter)
		}
	}
	if len(s.Patterns) != 4 {
		t.Errorf("want 4 pattern classes, got %d: %+v", len(s.Patterns), s.Patterns)
	}
}

func TestComputeTopPhrasesWithStopwords(t *testing.T) {
	// "青云山巅" xuất hiện với tần suất cao; "陆九渊" là tên nhân vật nên bị lọc bỏ
	line := "众人望向青云山巅，陆九渊负手而立。\n"
	chapters := make([]string, 10)
	for i := range chapters {
		chapters[i] = chapterWith(strings.Repeat(line, 3))
	}
	s := Compute(Input{Chapters: chapters, Stopwords: []string{"陆九渊"}})
	if s == nil {
		t.Fatal("expected stats")
	}
	var hasMountain, hasName bool
	for _, p := range s.TopPhrases {
		if strings.Contains(p.Text, "青云山") {
			hasMountain = true
		}
		if strings.Contains(p.Text, "九渊") || strings.Contains(p.Text, "陆九") {
			hasName = true
		}
	}
	if !hasMountain {
		t.Errorf("expected 青云山 phrase mined, got %+v", s.TopPhrases)
	}
	if hasName {
		t.Errorf("character name should be filtered, got %+v", s.TopPhrases)
	}
}

func TestComputeRepeatedSentences(t *testing.T) {
	motto := "此生未能远行，望你替我看看远方的山海。"
	chapters := make([]string, 6)
	for i := range chapters {
		body := "平常正文，没有什么重复。\n"
		if i%2 == 0 {
			body += motto + "\n"
		}
		chapters[i] = chapterWith(body)
	}
	s := Compute(Input{Chapters: chapters})
	if s == nil {
		t.Fatal("expected stats")
	}
	if len(s.RepeatedSentences) == 0 {
		t.Fatalf("expected repeated sentence, got none")
	}
	got := s.RepeatedSentences[0]
	if got.Chapters != 3 || got.Count != 3 {
		t.Errorf("repeated sentence: %+v", got)
	}
	if !strings.HasPrefix(got.Text, "此生未能远行") {
		t.Errorf("text: %q", got.Text)
	}
}

func TestComputeEndingAndOpening(t *testing.T) {
	short := chapterWith("一整夜没有睡。\n正文很长很长很长。\n他走了。")
	long := chapterWith("白天的事。\n正文。\n这是一个非常非常非常长的结尾句子，远远超过三十个字符的阈值长度，用来测试中位数。")
	chapters := []string{short, short, short, long, long}
	s := Compute(Input{Chapters: chapters})
	if s == nil {
		t.Fatal("expected stats")
	}
	if s.Ending.ShortRatio != 0.6 {
		t.Errorf("short_ratio: got %v want 0.6", s.Ending.ShortRatio)
	}
	if s.OpeningTimeRate != 0.6 {
		t.Errorf("opening_time_rate: got %v want 0.6", s.OpeningTimeRate)
	}
}

func TestComputeTitleFormats(t *testing.T) {
	chapters := make([]string, 5)
	for i := range chapters {
		chapters[i] = chapterWith("正文。")
	}
	// Dùng lẫn lộn → báo cáo
	s := Compute(Input{Chapters: chapters, Titles: []string{"第一章 风起", "云涌", "第3章 雷动"}})
	if s.TitleFormats == nil || s.TitleFormats.WithPrefix != 2 || s.TitleFormats.WithoutPrefix != 1 {
		t.Errorf("title formats: %+v", s.TitleFormats)
	}
	// Đồng nhất → không báo cáo
	s = Compute(Input{Chapters: chapters, Titles: []string{"风起", "云涌"}})
	if s.TitleFormats != nil {
		t.Errorf("uniform titles should not report: %+v", s.TitleFormats)
	}
}

func TestComputePatternsVietnamese(t *testing.T) {
	body := "Hắn không phải tức giận, mà là sợ hãi. Hắn im lặng trong chớp mắt. Ánh mắt như thể một ngọn đèn.\n"
	chapters := make([]string, 6)
	for i := range chapters {
		chapters[i] = "# Chương 1\n" + body
	}
	s := Compute(Input{Chapters: chapters})
	if s == nil {
		t.Fatal("expected stats")
	}
	if len(s.Patterns) != 4 {
		t.Fatalf("want 4 pattern classes, got %d: %+v", len(s.Patterns), s.Patterns)
	}
	for _, p := range s.Patterns {
		if p.Total != 6 {
			t.Errorf("%s total: got %d want 6", p.Name, p.Total)
		}
	}
}

func TestComputeTopPhrasesVietnamese(t *testing.T) {
	// "khóe môi khẽ nhếch lên" là cửa miệng; "Lâm Phong" là tên nhân vật nên mọi cụm chứa tên bị lọc
	line := "Lâm Phong khóe môi khẽ nhếch lên, nhìn về phía núi xa.\n"
	chapters := make([]string, 10)
	for i := range chapters {
		chapters[i] = "# Chương 1\n" + strings.Repeat(line, 3)
	}
	s := Compute(Input{Chapters: chapters, Stopwords: []string{"Lâm Phong"}})
	if s == nil {
		t.Fatal("expected stats")
	}
	var hasTic, hasName bool
	for _, p := range s.TopPhrases {
		if strings.Contains(p.Text, "khóe môi khẽ nhếch") {
			hasTic = true
		}
		if strings.Contains(strings.ToLower(p.Text), "phong") || strings.Contains(strings.ToLower(p.Text), "lâm") {
			hasName = true
		}
	}
	if !hasTic {
		t.Errorf("expected Vietnamese tic phrase mined, got %+v", s.TopPhrases)
	}
	if hasName {
		t.Errorf("character name should be filtered, got %+v", s.TopPhrases)
	}
}

func TestComputeVietnameseOpeningAndTitles(t *testing.T) {
	chapters := []string{
		"# Chương 1\nĐêm xuống, gió lạnh.",
		"# Chương 2\nSáng sớm, sương mù dày đặc.",
		"# Chương 3\nHắn bước vào quán trọ.",
		"# Chương 4\nHắn thức dậy trong cơn đau.",
		"# Chương 5\nTiếng chuông vang lên.",
	}
	s := Compute(Input{Chapters: chapters, Titles: []string{"Chương 1 Đêm mưa", "chương 2", "Gặp gỡ"}})
	if s == nil {
		t.Fatal("expected stats")
	}
	if s.OpeningTimeRate != 0.6 {
		t.Errorf("opening_time_rate: got %v want 0.6", s.OpeningTimeRate)
	}
	if s.TitleFormats == nil || s.TitleFormats.WithPrefix != 2 || s.TitleFormats.WithoutPrefix != 1 {
		t.Errorf("title formats: %+v", s.TitleFormats)
	}
}

func TestComputeRepeatedSentencesVietnamese(t *testing.T) {
	sent := "Hắn hít sâu một hơi rồi chậm rãi bước về phía trước"
	chapters := make([]string, 5)
	for i := range chapters {
		chapters[i] = "# Chương 1\nMở đầu khác nhau " + strings.Repeat("x", i) + ". " + sent + ". Kết."
	}
	s := Compute(Input{Chapters: chapters})
	if s == nil || len(s.RepeatedSentences) == 0 || s.RepeatedSentences[0].Chapters != 5 {
		t.Fatalf("expected repeated Vietnamese sentence across 5 chapters, got %+v", s)
	}
}
