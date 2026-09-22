package layout

import "strings"

func splitStyle(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// gk là khóa lưới của một cột: lane trước, cột trong lane sau. So sánh theo thứ
// tự từ điển, đúng như so tuple (lane, col) trong bản tham chiếu.
type gk struct{ lane, col int }

func (a gk) less(b gk) bool { return a.lane < b.lane || (a.lane == b.lane && a.col < b.col) }

// cell là một ô của lưới.
type cell struct{ lane, col, row int }

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

// pyMax và pyMin mang đúng ngữ nghĩa max và min của Python: khi hai số bằng
// nhau thì trả về số đứng trước. math.Max và math.Min không dùng được, vì
// math.Max(-0, 0) luôn trả +0, và dấu của số không lọt được vào toạ độ, nơi
// num.Fmt in nó ra thành "-0".
func pyMax(a, b float64) float64 {
	if b > a {
		return b
	}
	return a
}

func pyMin(a, b float64) float64 {
	if b < a {
		return b
	}
	return a
}
