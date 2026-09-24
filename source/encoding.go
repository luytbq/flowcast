package source

import (
	"strings"
	"unicode/utf8"
)

// csvEncodings là các bảng mã thử lần lượt khi không chỉ định. utf-8-sig đứng
// trước nên mọi file utf-8 hợp lệ đều được báo là utf-8-sig, kể cả khi không có
// BOM. cp1252 đứng cuối vì nó giải mã được gần như mọi chuỗi byte.
var csvEncodings = []string{"utf-8-sig", "utf-8", "cp1252"}

// encodingAliases chuẩn hóa tên bảng mã người dùng gõ vào --encoding về tên mà
// decode hiểu. Chỉ nhận bốn bảng mã, đủ cho file csv xuất từ bảng tính; tên
// khác bị báo là không hỗ trợ.
var encodingAliases = map[string]string{
	"utf-8": "utf-8", "utf8": "utf-8", "u8": "utf-8",
	"utf-8-sig": "utf-8-sig", "utf8-sig": "utf-8-sig",
	"cp1252": "cp1252", "windows-1252": "cp1252", "1252": "cp1252",
	"latin-1": "latin-1", "latin1": "latin-1", "iso-8859-1": "latin-1", "iso8859-1": "latin-1", "l1": "latin-1",
}

func normalizeEncoding(name string) (string, bool) {
	n := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "_", "-")
	canon, ok := encodingAliases[n]
	return canon, ok
}

// decode giải mã bytes theo một bảng mã, và không thay ký tự lỗi: một byte
// không giải mã được là cả file không giải mã được. ok là false khi giải mã không
// được hoặc không biết bảng mã.
func decode(raw []byte, enc string) (string, bool) {
	canon, ok := normalizeEncoding(enc)
	if !ok {
		return "", false
	}
	switch canon {
	case "utf-8-sig":
		if !utf8.Valid(raw) {
			return "", false
		}
		return strings.TrimPrefix(string(raw), "\ufeff"), true
	case "utf-8":
		if !utf8.Valid(raw) {
			return "", false
		}
		return string(raw), true
	case "latin-1":
		var b strings.Builder
		for _, c := range raw {
			b.WriteRune(rune(c))
		}
		return b.String(), true
	}
	var b strings.Builder
	for _, c := range raw {
		switch {
		case c < 0x80 || c >= 0xA0:
			b.WriteRune(rune(c))
		default:
			r, ok := cp1252High[c]
			if !ok {
				return "", false
			}
			b.WriteRune(r)
		}
	}
	return b.String(), true
}
