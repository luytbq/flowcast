// Package unistr định nghĩa khoảng trắng và chữ thường mà flowcast dùng khắp
// nơi để cắt và so chuỗi.
//
// Khoảng trắng gồm mọi ký tự unicode.IsSpace cộng thêm bốn ký tự phân tách
// U+001C tới U+001F. Bốn ký tự đó hay lọt vào ô khi dán từ bảng tính, và không
// ai muốn một ô chỉ chứa chúng bị coi là có chữ. Chữ thường theo SpecialCasing
// của Unicode, nên "İ" thành "i" cộng dấu chấm trên kết hợp. Không dùng thẳng
// strings.TrimSpace, strings.Fields hay strings.ToLower vì chúng khác ở đúng
// những chỗ đó.
//
// Bảng Unicode là bảng của phiên bản Go đang dùng. Chữ Hy Lạp Σ ở cuối từ không
// được đổi thành ς; chỗ duy nhất bị ảnh hưởng là thông điệp lỗi của một type
// viết bằng chữ Hy Lạp.
package unistr

import (
	"strings"
	"unicode"
)

// IsSpace nói r có phải khoảng trắng không.
func IsSpace(r rune) bool { return unicode.IsSpace(r) || (r >= 0x1c && r <= 0x1f) }

// Strip, LStrip, RStrip cắt khoảng trắng ở hai đầu, đầu trái, đầu phải.
func Strip(s string) string  { return strings.TrimFunc(s, IsSpace) }
func LStrip(s string) string { return strings.TrimLeftFunc(s, IsSpace) }
func RStrip(s string) string { return strings.TrimRightFunc(s, IsSpace) }

// Fields tách chuỗi theo khoảng trắng, bỏ các đoạn rỗng.
func Fields(s string) []string { return strings.FieldsFunc(s, IsSpace) }

// Lower đổi chữ thường. U+0130 thành "i" cộng dấu chấm trên kết hợp, theo
// SpecialCasing của Unicode.
func Lower(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == 0x130 {
			b.WriteString("i̇")
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
