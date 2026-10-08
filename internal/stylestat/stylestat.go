// Package stylestat thực hiện thống kê phong cách toàn bộ tác phẩm dựa trên phần chính văn đã viết,
// chỉ xuất ra các con số thực tế khách quan.
//
// Động lực: Cửa sổ đánh giá nội cung (~10 chương) vốn mù với các khuôn mẫu cố định ở cấp toàn tác phẩm —
// tic câu trúc có thể xuất hiện vài chục lần mỗi chương, cuối chương đồng dạng, lặp lại xuyên chương.
// Nhìn từng chương riêng lẻ thì mỗi chỗ đều "bình thường", chỉ thống kê toàn tác phẩm mới phơi bày được.
// Thống kê giao cho code (xác định, không ảo giác), phán xét giao cho LLM (editor căn cứ số liệu
// để đánh giá từng chiều, writer dựa đó tự tránh).
package stylestat

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// minChapters — ít hơn số chương này thì không xuất thống kê: mẫu quá nhỏ, tần suất không có ý nghĩa.
const minChapters = 5

// phraseWindow — khai thác cụm từ động chỉ xét N chương gần nhất: writer cần tránh "cửa miệng hiện tại".
const phraseWindow = 20

// Input là dữ liệu đầu vào để thống kê. Chapters xếp tăng dần theo số chương; Stopwords là
// danh từ riêng như tên nhân vật — bỏ qua khi khai thác cụm từ động (tên xuất hiện tự nhiên
// có tần suất cao, không phải vấn đề phong cách).
type Input struct {
	Chapters  []string
	Titles    []string
	Stopwords []string
}

// Stats là kết quả thống kê phong cách toàn tác phẩm. Tất cả trường đều là số liệu thực tế,
// không chứa bất kỳ nhận định hay chỉ thị nào.
type Stats struct {
	Chapters          int            `json:"chapters"`
	Patterns          []PatternStat  `json:"patterns,omitempty"`
	TopPhrases        []PhraseStat   `json:"top_phrases,omitempty"`
	RepeatedSentences []SentenceStat `json:"repeated_sentences,omitempty"`
	Ending            EndingStat     `json:"ending"`
	OpeningTimeRate   float64        `json:"opening_time_rate"`
	TitleFormats      *TitleStat     `json:"title_formats,omitempty"`
}

// PatternStat là số đếm toàn tác phẩm cho một lớp khuôn câu cố định (tic văn phong AI phổ biến).
type PatternStat struct {
	Name       string  `json:"name"`
	Total      int     `json:"total"`
	PerChapter float64 `json:"per_chapter"`
}

// PhraseStat là cụm từ xuất hiện nhiều được khai thác trong phraseWindow chương gần nhất.
type PhraseStat struct {
	Text  string `json:"text"`
	Count int    `json:"count"`
}

// SentenceStat là câu dài lặp lại từng chữ xuyên nhiều chương (bằng chứng trực tiếp của lặp lại phơi bày).
type SentenceStat struct {
	Text     string `json:"text"`
	Chapters int    `json:"chapters"`
	Count    int    `json:"count"`
}

// EndingStat là phân bố hình thức dòng cuối chương. Kết thúc ngắn tự nó hợp lệ,
// chỉ khi đồng dạng toàn tác phẩm mới là vấn đề.
type EndingStat struct {
	ShortRatio  float64 `json:"short_ratio"`
	MedianRunes int     `json:"median_runes"`
}

// TitleStat là số đếm việc dùng lẫn lộn tiền tố "Chương N" trong tiêu đề chương
// (dùng lẫn = dấu vết cơ chế lộ ra trong sản phẩm).
type TitleStat struct {
	WithPrefix    int `json:"with_prefix"`
	WithoutPrefix int `json:"without_prefix"`
}

// patternDefs là các khuôn câu AI phổ biến. Số đếm là xấp xỉ (regex không phân tích ngữ pháp),
// mục đích là so sánh theo chiều dọc với đường cơ sở của chính tác phẩm, độ chính xác tuyệt đối không quan trọng.
// Mỗi lớp khớp cả mẫu tiếng Việt lẫn tiếng Trung (truyện tạo trước khi Việt hoá).
var patternDefs = []struct {
	name string
	re   *regexp.Regexp
}{
	{"Câu chỉnh chuẩn『không phải… mà là…』", regexp.MustCompile(`(?i)không phải[^.!?。！？\n]{1,40}?mà (?:là|chính là)|不是[^。！？\n]{1,24}?[，、]?(?:而)?是`)},
	{"Lượng từ thời gian『một hơi thở/trong chớp mắt』", regexp.MustCompile(`(?i)(?:một|hai|ba|mấy|vài|nửa|mười) (?:hơi thở|nhịp thở)|trong (?:chớp|nháy) mắt|[一两二三四五六七八九十几数半][息瞬]`)},
	{"So sánh trực tiếp『như một/như thể/tựa như』", regexp.MustCompile(`(?i)như một|như thể|tựa như|hệt như|giống như|像一|仿佛|如同|宛如`)},
	{"Nhịp im lặng『im lặng/không nói gì/không quay đầu』", regexp.MustCompile(`(?i)im lặng|trầm mặc|không nói gì|không quay (?:đầu|lại)|không ngoảnh lại|沉默了|没有说话|没有回头`)},
}

var (
	sentenceSplit = regexp.MustCompile(`[。！？.!?…\n]+`)
	openingTimeRe = regexp.MustCompile(`(?i)đêm|sáng sớm|bình minh|rạng sáng|trời sáng|tỉnh dậy|thức dậy|ban mai|夜|清晨|黎明|天亮|醒来|晨光|一整夜`)
	titlePrefixRe = regexp.MustCompile(`(?i)^#{0,2}\s*(?:chương\s*\d+|第[零〇一二三四五六七八九十百千万\d]+章)`)
	// phraseSegmentRe tách văn bản tiếng Việt thành đoạn không chứa dấu câu, để cụm từ không vắt qua câu.
	phraseSegmentRe = regexp.MustCompile(`[^\p{L}\p{M}' ]+`)
)

// shortEndingRunes — dòng cuối không vượt quá số ký tự này thì tính là "kết thúc ngắn".
const shortEndingRunes = 30

// Compute tính thống kê phong cách toàn tác phẩm; trả về nil nếu số chương chưa đủ.
func Compute(in Input) *Stats {
	n := len(in.Chapters)
	if n < minChapters {
		return nil
	}
	all := strings.Join(in.Chapters, "\n")

	s := &Stats{Chapters: n}
	for _, def := range patternDefs {
		total := len(def.re.FindAllStringIndex(all, -1))
		if total == 0 {
			continue
		}
		s.Patterns = append(s.Patterns, PatternStat{
			Name:       def.name,
			Total:      total,
			PerChapter: round1(float64(total) / float64(n)),
		})
	}
	s.TopPhrases = minePhrases(recentWindow(in.Chapters), in.Stopwords)
	s.RepeatedSentences = repeatedSentences(in.Chapters)
	s.Ending = endingShape(in.Chapters)
	s.OpeningTimeRate = openingTimeRate(in.Chapters)
	s.TitleFormats = titleFormats(in.Titles)
	return s
}

func recentWindow(chapters []string) []string {
	if len(chapters) <= phraseWindow {
		return chapters
	}
	return chapters[len(chapters)-phraseWindow:]
}

// minePhrases khai thác các cụm từ xuất hiện nhiều trong cửa sổ: 3–6 Hán tự (tiếng Trung)
// hoặc 3–6 âm tiết (tiếng Việt, ngăn bởi dấu cách).
// Lọc: có dấu câu/khoảng trắng, hư từ/đại từ ở đầu/cuối, trùng danh từ riêng;
// loại trùng: cụm nào là chuỗi con của cụm đã chọn thì bỏ.
func minePhrases(chapters []string, stopwords []string) []PhraseStat {
	text := strings.Join(chapters, "\n")
	runes := []rune(text)
	threshold := max(8, len(chapters)/2)

	counts := make(map[string]int)
	for size := 3; size <= 6; size++ {
		for i := 0; i+size <= len(runes); i++ {
			gram := runes[i : i+size]
			if !validGram(gram) {
				continue
			}
			counts[string(gram)]++
		}
	}
	wordCounts := mineWordGrams(text, nameSyllables(stopwords))

	stopGrams := stopwordBigrams(stopwords)
	type cand struct {
		text  string
		count int
	}
	var cands []cand
	for g, c := range counts {
		if c < threshold || hitStopword(g, stopGrams) {
			continue
		}
		cands = append(cands, cand{g, c})
	}
	for g, c := range wordCounts {
		if c >= threshold {
			cands = append(cands, cand{g, c})
		}
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].count != cands[j].count {
			return cands[i].count > cands[j].count
		}
		// Cùng tần suất thì ưu tiên cụm dài hơn (thông tin nhiều hơn), sau đó sắp xếp từ điển để ổn định
		if len(cands[i].text) != len(cands[j].text) {
			return len(cands[i].text) > len(cands[j].text)
		}
		return cands[i].text < cands[j].text
	})

	var out []PhraseStat
	for _, c := range cands {
		if len(out) >= 8 {
			break
		}
		dup := false
		for _, picked := range out {
			if strings.Contains(picked.Text, c.text) || strings.Contains(c.text, picked.Text) {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, PhraseStat{Text: c.text, Count: c.count})
		}
	}
	return out
}

// gramEdgeStop — n-gram có hư từ/đại từ ở đầu hoặc cuối không phải cụm từ phong cách, bỏ qua.
const gramEdgeStop = "的了着是在和与就也都还又把被他她它我你这那"

func validGram(gram []rune) bool {
	for _, r := range gram {
		if r < 0x4E00 || r > 0x9FFF { // chỉ chấp nhận đoạn thuần Hán tự
			return false
		}
	}
	if strings.ContainsRune(gramEdgeStop, gram[0]) || strings.ContainsRune(gramEdgeStop, gram[len(gram)-1]) {
		return false
	}
	return true
}

// wordEdgeStop — hư từ/đại từ tiếng Việt: cụm âm tiết bắt đầu hoặc kết thúc bằng chúng
// không phải cụm từ phong cách (tương đương gramEdgeStop của tiếng Trung).
var wordEdgeStop = toSet(strings.Fields(`và là của thì mà đã đang sẽ những các một này đó kia với cho để
	trong ngoài lại cũng vẫn còn rồi nhưng nên vì không có bị được ra vào lên xuống
	anh cô hắn nàng nó tôi ta y họ ông bà chàng gã mình em chị`))

// mineWordGrams đếm cụm 3–6 âm tiết Latin (tiếng Việt) trong từng đoạn không dấu câu.
// Cụm chứa âm tiết thuộc tên riêng (so khớp phân biệt hoa thường — tên luôn viết hoa) bị bỏ.
func mineWordGrams(text string, names map[string]struct{}) map[string]int {
	counts := make(map[string]int)
	for _, seg := range phraseSegmentRe.Split(text, -1) {
		words := strings.Fields(seg)
		for size := 3; size <= 6; size++ {
			for i := 0; i+size <= len(words); i++ {
				gram := words[i : i+size]
				if validWordGram(gram, names) {
					counts[strings.ToLower(strings.Join(gram, " "))]++
				}
			}
		}
	}
	return counts
}

func validWordGram(gram []string, names map[string]struct{}) bool {
	for _, w := range gram {
		if _, ok := names[w]; ok {
			return false
		}
		for _, r := range w {
			if r >= 0x4E00 && r <= 0x9FFF { // Hán tự đã được nhánh rune-gram xử lý
				return false
			}
		}
	}
	_, firstStop := wordEdgeStop[strings.ToLower(gram[0])]
	_, lastStop := wordEdgeStop[strings.ToLower(gram[len(gram)-1])]
	return !firstStop && !lastStop
}

// nameSyllables tách tên riêng thành các âm tiết viết hoa ("Lâm Phong" → Lâm, Phong).
func nameSyllables(stopwords []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, w := range stopwords {
		for _, syl := range strings.Fields(w) {
			if r := []rune(syl); len(r) > 0 && unicode.IsUpper(r[0]) {
				m[syl] = struct{}{}
			}
		}
	}
	return m
}

func toSet(words []string) map[string]struct{} {
	m := make(map[string]struct{}, len(words))
	for _, w := range words {
		m[w] = struct{}{}
	}
	return m
}

// stopwordBigrams tách danh từ riêng thành các mảnh 2 ký tự: tên người thường xuất hiện
// một phần trong văn ("Cửu Uyên chắp tay" chứa "Cửu Uyên"), khớp theo tên đầy đủ sẽ bỏ sót.
// Thà lọc nghiêm hơn — bớt một cụm từ thực tế không sao, còn tên người lọt vào danh sách
// cửa miệng mới là nhiễu.
func stopwordBigrams(stopwords []string) []string {
	var grams []string
	for _, w := range stopwords {
		runes := []rune(strings.TrimSpace(w))
		if len(runes) < 2 {
			continue
		}
		for i := 0; i+2 <= len(runes); i++ {
			grams = append(grams, string(runes[i:i+2]))
		}
	}
	return grams
}

func hitStopword(gram string, stopGrams []string) bool {
	for _, g := range stopGrams {
		if strings.Contains(gram, g) {
			return true
		}
	}
	return false
}

// repeatedSentences tìm các câu ≥12 ký tự lặp lại từng chữ xuyên ≥3 chương, lấy top 5 theo số lần.
func repeatedSentences(chapters []string) []SentenceStat {
	type rec struct {
		count    int
		chapters map[int]struct{}
	}
	seen := make(map[string]*rec)
	for ci, text := range chapters {
		for _, sent := range sentenceSplit.Split(text, -1) {
			// Bỏ dấu ngoặc kép bao quanh rồi gộp: cùng một câu thoại có/không có ngoặc mở không nên tính là hai câu khác
			sent = strings.Trim(strings.TrimSpace(sent), `"""''「」『』`)
			if len([]rune(sent)) < 12 {
				continue
			}
			r := seen[sent]
			if r == nil {
				r = &rec{chapters: make(map[int]struct{})}
				seen[sent] = r
			}
			r.count++
			r.chapters[ci] = struct{}{}
		}
	}

	var out []SentenceStat
	for sent, r := range seen {
		if len(r.chapters) < 3 {
			continue
		}
		out = append(out, SentenceStat{Text: truncateRunes(sent, 40), Chapters: len(r.chapters), Count: r.count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Text < out[j].Text
	})
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}

func endingShape(chapters []string) EndingStat {
	var lengths []int
	short := 0
	for _, text := range chapters {
		line := lastNonEmptyLine(text)
		if line == "" {
			continue
		}
		n := len([]rune(line))
		lengths = append(lengths, n)
		if n <= shortEndingRunes {
			short++
		}
	}
	if len(lengths) == 0 {
		return EndingStat{}
	}
	sort.Ints(lengths)
	return EndingStat{
		ShortRatio:  round2(float64(short) / float64(len(lengths))),
		MedianRunes: lengths[len(lengths)/2],
	}
}

func openingTimeRate(chapters []string) float64 {
	hit := 0
	for _, text := range chapters {
		if openingTimeRe.MatchString(firstParagraph(text)) {
			hit++
		}
	}
	return round2(float64(hit) / float64(len(chapters)))
}

func titleFormats(titles []string) *TitleStat {
	if len(titles) == 0 {
		return nil
	}
	t := &TitleStat{}
	for _, title := range titles {
		if strings.TrimSpace(title) == "" {
			continue
		}
		if titlePrefixRe.MatchString(title) {
			t.WithPrefix++
		} else {
			t.WithoutPrefix++
		}
	}
	// Chỉ báo cáo khi có dùng lẫn lộn; định dạng đồng nhất không phải vấn đề thực tế
	if t.WithPrefix == 0 || t.WithoutPrefix == 0 {
		return nil
	}
	return t
}

func lastNonEmptyLine(text string) string {
	lines := strings.Split(text, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return ""
}

// firstParagraph lấy dòng đầu tiên không rỗng và không phải tiêu đề Markdown
// (dòng đầu file chương thường là tiêu đề # ).
func firstParagraph(text string) string {
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return ""
}

func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

func round1(f float64) float64 { return float64(int(f*10+0.5)) / 10 }
func round2(f float64) float64 { return float64(int(f*100+0.5)) / 100 }
