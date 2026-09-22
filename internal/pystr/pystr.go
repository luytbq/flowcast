// Package pystr mang đúng ngữ nghĩa khoảng trắng và chữ thường của str trong
// Python, thứ bản tham chiếu dùng khắp nơi để cắt và so chuỗi.
//
// Không dùng thẳng strings.TrimSpace, strings.Fields hay strings.ToLower: đã đo
// trên toàn bộ không gian Unicode thì Python coi thêm U+001C tới U+001F là
// khoảng trắng mà Go không coi, và "İ".lower() của Python ra hai ký tự còn Go ra
// một.
//
// Giới hạn đã biết: Python 3.14 dùng Unicode 16, còn bảng Unicode của Go 1.25 là
// Unicode 15. Ký tự mới có ở Unicode 16 có thể đổi chữ thường khác nhau. Chữ Hy
// Lạp Σ ở cuối từ không được đổi thành ς như str.lower() của Python làm; chỗ duy
// nhất bị ảnh hưởng là thông điệp lỗi của một type viết bằng chữ Hy Lạp.
package pystr

import (
	"strings"
	"unicode"
)

// IsSpace đúng như str.isspace() của Python cho một ký tự, và như \s của re
// trên chuỗi str.
func IsSpace(r rune) bool { return unicode.IsSpace(r) || (r >= 0x1c && r <= 0x1f) }

// Strip, LStrip, RStrip như str.strip(), str.lstrip(), str.rstrip() không đối số.
func Strip(s string) string  { return strings.TrimFunc(s, IsSpace) }
func LStrip(s string) string { return strings.TrimLeftFunc(s, IsSpace) }
func RStrip(s string) string { return strings.TrimRightFunc(s, IsSpace) }

// Fields như str.split() không đối số.
func Fields(s string) []string { return strings.FieldsFunc(s, IsSpace) }

// Lower như str.lower(). U+0130 thành "i" cộng dấu chấm trên kết hợp, theo
// SpecialCasing của Unicode, đúng như Python.
func Lower(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == 0x130 {
			b.WriteString("i\u0307")
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
