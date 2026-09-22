package text

import (
	"github.com/luytbq/flowcast/internal/pystr"
	"math"
	"strings"
)

// breakChars là các ký tự mà sau chúng được phép ngắt một từ dài.
//
// Bản tham chiếu viết luật này bằng lookbehind: (?<=[/._,(){}=&?-])|(?<=::).
// RE2 không có lookbehind, nên chỗ ngắt được tìm thẳng thay vì bằng regexp.
const breakChars = "/._,(){}=&?-"

const (
	DefaultSize  = 12.0
	DefaultLineH = 15.0
)

// Measure ngắt dòng và đo hộp chữ.
type Measure struct {
	m     *Metrics
	Size  float64
	LineH float64
	hard  bool
}

func NewMeasure(m *Metrics) *Measure {
	return &Measure{m: m, Size: DefaultSize, LineH: DefaultLineH}
}

func (t *Measure) W(s string) float64 { return t.m.Width(s, t.Size) }

// Hard cho biết lần ngắt dòng gần nhất có phải cắt giữa một từ hay không.
//
// Đây là cờ tạm, đặt lại theo từng dòng bên trong Wrap. Đọc nó một lần sau khi
// ngắt nhiều dòng là đọc dòng cuối cùng, không phải cả cụm.
func (t *Measure) Hard() bool { return t.hard }

// Box trả về bề rộng dòng dài nhất và tổng bề cao.
func (t *Measure) Box(lines []string) (w, h float64) {
	for _, l := range lines {
		if x := t.W(l); x > w {
			w = x
		}
	}
	return w, float64(float64(len(lines)) * t.LineH)
}

// breakAfter nói có được ngắt ngay trước vị trí i hay không. Gọi với 1 <= i <= len(r).
func breakAfter(r []rune, i int) bool {
	if strings.ContainsRune(breakChars, r[i-1]) {
		return true
	}
	return i >= 2 && r[i-1] == ':' && r[i-2] == ':'
}

// breakSplit cắt một từ tại mọi chỗ được phép ngắt, bỏ mảnh rỗng.
func breakSplit(word string) []string {
	r := []rune(word)
	var out []string
	start := 0
	for i := 1; i <= len(r); i++ {
		if breakAfter(r, i) {
			if i > start {
				out = append(out, string(r[start:i]))
			}
			start = i
		}
	}
	if start < len(r) {
		out = append(out, string(r[start:]))
	}
	return out
}

// splitLong cắt một từ rộng hơn maxw, ưu tiên chỗ được phép ngắt; hết cách thì
// cắt cứng giữa từ và bật cờ hard.
func (t *Measure) splitLong(word string, maxw float64) []string {
	chunks := breakSplit(word)
	var out []string
	cur := ""
	for _, c := range chunks {
		if t.W(cur+c) <= maxw {
			cur += c
			continue
		}
		if cur != "" {
			out = append(out, cur)
		}
		cur = ""
		for t.W(c) > maxw {
			t.hard = true
			rc := []rune(c)
			k := len(rc)
			for k > 1 && t.W(string(rc[:k])) > maxw {
				k--
			}
			out = append(out, string(rc[:k]))
			c = string(rc[k:])
		}
		cur = c
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func (t *Measure) wrapLine(line string, maxw float64) []string {
	line = pystr.Strip(line)
	if t.W(line) <= maxw {
		return []string{line}
	}
	var out []string
	cur := ""
	for _, word := range pystr.Fields(line) {
		cand := word
		if cur != "" {
			cand = cur + " " + word
		}
		if t.W(cand) <= maxw {
			cur = cand
			continue
		}
		if cur != "" {
			out = append(out, cur)
		}
		if t.W(word) <= maxw {
			cur = word
			continue
		}
		parts := t.splitLong(word, maxw)
		out = append(out, parts[:len(parts)-1]...)
		cur = parts[len(parts)-1]
	}
	return append(out, cur)
}

// Wrap ngắt dòng ở bề rộng maxw, rồi thu hẹp tới mức nhỏ nhất vẫn giữ nguyên số
// dòng, để các dòng dài xấp xỉ nhau thay vì một dòng đầy và một dòng cụt.
func (t *Measure) Wrap(lines []string, maxw float64) []string {
	var out []string
	for _, line := range lines {
		t.hard = false
		first := t.wrapLine(line, maxw)
		if len(first) == 1 || t.hard {
			out = append(out, first...)
			continue
		}
		lo, hi := 1, int(math.Ceil(maxw))
		for lo < hi {
			mid := (lo + hi) / 2
			t.hard = false
			if len(t.wrapLine(line, float64(mid))) <= len(first) && !t.hard {
				hi = mid
			} else {
				lo = mid + 1
			}
		}
		out = append(out, t.wrapLine(line, float64(hi))...)
	}
	return out
}
