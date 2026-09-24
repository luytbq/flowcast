// Package source đọc các định dạng đầu vào và quy chúng về một model.Table.
//
// Nhận bytes chứ không nhận đường dẫn: core không chạm filesystem, nên cùng một
// đường code phục vụ được CLI đọc file và dịch vụ web nhận upload.
package source

import (
	"github.com/luytbq/flowcast/internal/unistr"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// Header là năm tên cột bắt buộc. Số cột là một cam kết tương thích, xem
// docs/adr/0001.
var Header = []string{"id", "type", "parent", "content", "metadata"}

// mdEscapable là tập ký tự mà markdown cho phép đặt gạch chéo ngược phía trước.
const mdEscapable = "\\`*_{}[]()#+-.!|<>~"

// splitLines cắt dòng theo mọi ranh giới dòng, không chỉ theo xuống dòng kiểu
// Unix: LF, CR, CRLF, VT, FF, các ký tự phân tách U+001C tới U+001E, NEL, và
// dấu phân dòng, phân đoạn của Unicode. File soạn trên Windows dùng CRLF, và
// vài nguồn dán vào có dấu phân đoạn của Unicode.
func splitLines(s string) []string {
	var out []string
	start, i := 0, 0
	r := []rune(s)
	for i < len(r) {
		c := r[i]
		n := 1
		switch c {
		case 0x0d:
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
	s := unistr.Strip(line)
	s = strings.TrimPrefix(s, "|")
	if strings.HasSuffix(s, "|") && !strings.HasSuffix(s, "\\|") {
		s = s[:len(s)-1]
	}
	var cells []string
	var cur strings.Builder
	prevEscape := false
	for _, c := range s {
		if c == '|' && !prevEscape {
			cells = append(cells, unistr.Strip(cur.String()))
			cur.Reset()
			prevEscape = false
			continue
		}
		cur.WriteRune(c)
		prevEscape = c == '\\' && !prevEscape
	}
	return append(cells, unistr.Strip(cur.String()))
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
		if r[i] == 0x5c && i+1 < len(r) && strings.ContainsRune(mdEscapable, r[i+1]) {
			b.WriteRune(r[i+1])
			i++
			continue
		}
		b.WriteRune(r[i])
	}
	return norm.NFC.String(b.String())
}

func normalize(s string) string { return norm.NFC.String(s) }

// splitBR cắt một ô tại các thẻ xuống dòng của html, như BR_RE.split của bản
// tham chiếu với biểu thức <br\s*/?> không phân biệt hoa thường.
func splitBR(cell string) []string {
	r := []rune(cell)
	var out []string
	start := 0
	for i := 0; i < len(r); {
		if n := matchBR(r[i:]); n > 0 {
			out = append(out, string(r[start:i]))
			i += n
			start = i
			continue
		}
		i++
	}
	return append(out, string(r[start:]))
}

// matchBR trả về số ký tự của thẻ br ở đầu r, hoặc 0 nếu không có.
//
// So từng ký tự thay vì hạ chữ thường cả chuỗi rồi dò vị trí: hạ chữ thường đổi
// được độ dài chuỗi, như İ thành i cộng dấu chấm, nên vị trí dò trên chuỗi đã hạ
// không cắt đúng chuỗi gốc. Không phân biệt hoa thường chỉ áp cho b và r, và
// chỉ khớp đúng B và R, không khớp ký tự Unicode nào khác.
// Tham lam không lùi là đủ: khoảng trắng, dấu gạch chéo và dấu lớn hơn không lẫn
// vào nhau nên không có cách lùi nào khớp được khi cách tham lam không khớp.
func matchBR(r []rune) int {
	if len(r) < 4 || r[0] != '<' || (r[1] != 'b' && r[1] != 'B') || (r[2] != 'r' && r[2] != 'R') {
		return 0
	}
	i := 3
	for i < len(r) && unistr.IsSpace(r[i]) {
		i++
	}
	if i < len(r) && r[i] == '/' {
		i++
	}
	if i < len(r) && r[i] == '>' {
		return i + 1
	}
	return 0
}

func trimLeftSpace(s string) string {
	return unistr.LStrip(s)
}

func lower(cells []string) []string {
	out := make([]string, len(cells))
	for i, c := range cells {
		out[i] = unistr.Lower(c)
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
