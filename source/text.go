// Package source đọc các định dạng đầu vào và quy chúng về một model.Table.
//
// Nhận bytes chứ không nhận đường dẫn: core không chạm filesystem, nên cùng một
// đường code phục vụ được CLI đọc file và dịch vụ web nhận upload.
package source

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Header là năm tên cột bắt buộc. Số cột là một cam kết tương thích, xem
// docs/adr/0001.
var Header = []string{"id", "type", "parent", "content", "metadata"}

// mdEscapable là tập ký tự mà markdown cho phép đặt gạch chéo ngược phía trước.
const mdEscapable = "\\`*_{}[]()#+-.!|<>~"

// splitLines cắt dòng theo đúng tập ranh giới mà str.splitlines của Python
// dùng, không chỉ theo xuống dòng kiểu Unix.
//
// Tách riêng vì đây là chỗ dễ lệch giữa hai bản: file soạn trên Windows dùng
// CRLF, và vài nguồn dán vào có dấu phân đoạn của Unicode.
func splitLines(s string) []string {
	var out []string
	start, i := 0, 0
	r := []rune(s)
	for i < len(r) {
		c := r[i]
		n := 1
		switch c {
		case '\r':
			if i+1 < len(r) && r[i+1] == '\n' {
				n = 2
			}
		case '\n', '\v', '\f', 0x1c, 0x1d, 0x1e, 0x85, 0x2028, 0x2029:
		default:
			i++
			continue
		}
		out = append(out, string(r[start:i]))
		i += n
		start = i
	}
	if start < len(r) {
		out = append(out, string(r[start:]))
	}
	return out
}

// splitCells tách một dòng bảng markdown thành các ô.
//
// Chỉ gạch đứng không có gạch chéo ngược phía trước mới là dấu ngăn cột, nên
// nội dung viết \| giữ được gạch đứng mà không vỡ bảng.
func splitCells(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	if strings.HasSuffix(s, "|") && !strings.HasSuffix(s, "\\|") {
		s = s[:len(s)-1]
	}
	var cells []string
	var cur strings.Builder
	prevEscape := false
	for _, c := range s {
		if c == '|' && !prevEscape {
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
			prevEscape = false
			continue
		}
		cur.WriteRune(c)
		prevEscape = c == '\\' && !prevEscape
	}
	return append(cells, strings.TrimSpace(cur.String()))
}

// unescape gỡ gạch chéo ngược của markdown rồi chuẩn hóa về NFC.
//
// Chuẩn hóa là bắt buộc chứ không phải làm cho đẹp: "Xử lý" ở dạng tách dấu đo
// ra 42.43px thay vì 30.75px, tức bố cục đổi hẳn, và macOS thường sinh dạng
// tách dấu khi chép chữ tiếng Việt. Xem docs/adr/0005.
func unescape(s string) string {
	var b strings.Builder
	r := []rune(s)
	for i := 0; i < len(r); i++ {
		if r[i] == '\\' && i+1 < len(r) && strings.ContainsRune(mdEscapable, r[i+1]) {
			b.WriteRune(r[i+1])
			i++
			continue
		}
		b.WriteRune(r[i])
	}
	return norm.NFC.String(b.String())
}

func normalize(s string) string { return norm.NFC.String(s) }

// splitBR cắt một ô tại các thẻ xuống dòng của html, không phân biệt hoa thường.
func splitBR(cell string) []string {
	var out []string
	rest := cell
	for {
		i, n := findBR(rest)
		if i < 0 {
			return append(out, rest)
		}
		out = append(out, rest[:i])
		rest = rest[i+n:]
	}
}

// findBR trả về vị trí và độ dài của thẻ <br>, <br/> hoặc <br />; -1 nếu không có.
func findBR(s string) (int, int) {
	low := strings.ToLower(s)
	for i := 0; i+3 <= len(low); i++ {
		if low[i:i+3] != "<br" {
			continue
		}
		j := i + 3
		for j < len(low) && (low[j] == ' ' || low[j] == '\t' || low[j] == '\n' || low[j] == '\r') {
			j++
		}
		if j < len(low) && low[j] == '/' {
			j++
		}
		if j < len(low) && low[j] == '>' {
			return i, j + 1 - i
		}
	}
	return -1, 0
}

func trimLeftSpace(s string) string {
	return strings.TrimLeftFunc(s, unicode.IsSpace)
}

func lower(cells []string) []string {
	out := make([]string, len(cells))
	for i, c := range cells {
		out[i] = strings.ToLower(c)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
