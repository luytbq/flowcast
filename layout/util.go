package layout

import (
	"strings"

	"github.com/luytbq/flowcast/internal/unistr"
)

func splitStyle(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = unistr.Strip(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// gk là khóa lưới của một cột: lane trước, cột trong lane sau. So sánh theo thứ
// tự từ điển.
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

// fmax và fmin trả về số đứng trước khi hai số bằng nhau. math.Max và math.Min
// không dùng được, vì math.Max(-0, 0) luôn trả +0: kết quả khi đó phụ thuộc vào
// dấu của số không chứ không chỉ vào giá trị, và engine cần kết quả tất định.
func fmax(a, b float64) float64 {
	if b > a {
		return b
	}
	return a
}

func fmin(a, b float64) float64 {
	if b < a {
		return b
	}
	return a
}
