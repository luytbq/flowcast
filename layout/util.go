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
