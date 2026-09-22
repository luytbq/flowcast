package merge

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/luytbq/flowcast/internal/pystr"
)

// pyFloat đọc số như float(s) của Python. Khác strconv.ParseFloat ở chỗ nhận
// khoảng trắng hai đầu, kể cả khoảng trắng Unicode, và chữ số Unicode, nhận dấu gạch dưới giữa hai chữ số,
// và không nhận số viết theo hệ mười sáu.
func pyFloat(s string) (float64, bool) {
	var b strings.Builder
	for _, r := range s {
		// Ký tự ASCII giữ nguyên, nên U+001C tới U+001F không phải khoảng trắng ở
		// đây dù str.isspace nhận chúng.
		switch {
		case r < utf8.RuneSelf:
			b.WriteRune(r)
		case pystr.IsSpace(r):
			b.WriteByte(' ')
		default:
			d, ok := decimalValue(r)
			if !ok {
				return 0, false
			}
			b.WriteByte(byte('0' + d))
		}
	}
	t := strings.Trim(b.String(), " \t\n\v\f\r")
	body := strings.TrimLeft(t, "+-")
	if len(t)-len(body) > 1 {
		return 0, false
	}
	switch strings.ToLower(body) {
	case "inf", "infinity":
		if strings.HasPrefix(t, "-") {
			return math.Inf(-1), true
		}
		return math.Inf(1), true
	case "nan":
		return math.NaN(), true
	}
	clean, ok := decimalLiteral(body)
	if !ok {
		return 0, false
	}
	v, err := strconv.ParseFloat(t[:len(t)-len(body)]+clean, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0, false
	}
	return v, true
}

// decimalLiteral kiểm cú pháp digits [. digits] [e [sign] digits] của Python,
// trong đó phần nguyên hoặc phần lẻ được vắng một, và trả về chuỗi đã bỏ dấu
// gạch dưới.
func decimalLiteral(s string) (string, bool) {
	var out []byte
	i := 0
	digits := func() int {
		n := 0
		for i < len(s) {
			c := s[i]
			if c >= '0' && c <= '9' {
				out = append(out, c)
				n++
				i++
				continue
			}
			if c == '_' && n > 0 && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
				i++
				continue
			}
			break
		}
		return n
	}
	n := digits()
	if i < len(s) && s[i] == '.' {
		out = append(out, '.')
		i++
		n += digits()
	}
	if n == 0 {
		return "", false
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		out = append(out, 'e')
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			out = append(out, s[i])
			i++
		}
		if digits() == 0 {
			return "", false
		}
	}
	return string(out), i == len(s)
}

// decimalValue trả về giá trị của một chữ số thập phân Unicode. Các chữ số loại
// Nd luôn đi thành bộ mười liền nhau bắt đầu từ số không, nên giá trị là vị trí
// trong khoảng tính theo mô-đun mười.
func decimalValue(r rune) (int, bool) {
	for _, rg := range unicode.Nd.R16 {
		if rg.Stride == 1 && rune(rg.Lo) <= r && r <= rune(rg.Hi) {
			return int(r-rune(rg.Lo)) % 10, true
		}
	}
	for _, rg := range unicode.Nd.R32 {
		if rg.Stride == 1 && rune(rg.Lo) <= r && r <= rune(rg.Hi) {
			return int(r-rune(rg.Lo)) % 10, true
		}
	}
	return 0, false
}

// b64decode giải base64 như base64.b64decode của Python ở chế độ không chặt:
// bỏ qua ký tự ngoài bảng chữ, và dừng ở dấu đệm đủ cho nhóm bốn ký tự.
func b64decode(s string) ([]byte, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var out []byte
	quad, pads, left, chars := 0, 0, 0, 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '=' {
			if quad >= 2 {
				pads++
				if quad+pads >= 4 {
					return out, nil
				}
			}
			continue
		}
		v := strings.IndexByte(alphabet, c)
		if v < 0 {
			continue
		}
		chars++
		pads = 0
		switch quad {
		case 0:
			left = v
			quad = 1
		case 1:
			out = append(out, byte(left<<2|v>>4))
			left = v & 0xf
			quad = 2
		case 2:
			out = append(out, byte(left<<4|v>>2))
			left = v & 0x3
			quad = 3
		case 3:
			out = append(out, byte(left<<6|v))
			quad = 0
		}
	}
	switch quad {
	case 0:
		return out, nil
	case 1:
		return nil, errors.New("Invalid base64-encoded string: number of data characters (" +
			strconv.Itoa(chars) + ") cannot be 1 more than a multiple of 4")
	}
	return nil, errors.New("Incorrect padding")
}

// unquote giải %XX như urllib.parse.unquote: chỉ trong các đoạn ASCII, và byte
// giải ra được đọc thành UTF-8 với phép thay U+FFFD của Python.
func unquote(s string) string {
	if !strings.Contains(s, "%") {
		return s
	}
	var b strings.Builder
	for len(s) > 0 {
		n := 0
		for n < len(s) && s[n] < utf8.RuneSelf {
			n++
		}
		if n == 0 {
			_, w := utf8.DecodeRuneInString(s)
			b.WriteString(s[:w])
			s = s[w:]
			continue
		}
		b.WriteString(decodeReplace(unquoteBytes(s[:n])))
		s = s[n:]
	}
	return b.String()
}

func unhex(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

func unquoteBytes(s string) []byte {
	var out []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			hi, ok1 := unhex(s[i+1])
			lo, ok2 := unhex(s[i+2])
			if ok1 && ok2 {
				out = append(out, hi<<4|lo)
				i += 2
				continue
			}
		}
		out = append(out, s[i])
	}
	return out
}

// decodeReplace đọc UTF-8 như bytes.decode("utf-8", "replace") của Python: mỗi
// đoạn con hợp lệ dài nhất của một chuỗi byte hỏng thành đúng một U+FFFD. Phép
// chuyển string của Go thay từng byte, nên một chuỗi ba byte bị cắt cụt ra hai
// U+FFFD thay vì một.
func decodeReplace(p []byte) string {
	var b strings.Builder
	for i := 0; i < len(p); {
		c := p[i]
		if c < 0x80 {
			b.WriteByte(c)
			i++
			continue
		}
		need, lo, hi := 0, byte(0x80), byte(0xbf)
		switch {
		case c >= 0xc2 && c <= 0xdf:
			need = 1
		case c == 0xe0:
			need, lo = 2, 0xa0
		case c == 0xed:
			need, hi = 2, 0x9f
		case c >= 0xe1 && c <= 0xef:
			need = 2
		case c == 0xf0:
			need, lo = 3, 0x90
		case c == 0xf4:
			need, hi = 3, 0x8f
		case c >= 0xf1 && c <= 0xf3:
			need = 3
		}
		j := i + 1
		for k := 0; k < need && j < len(p); k++ {
			if p[j] < lo || p[j] > hi {
				break
			}
			lo, hi = 0x80, 0xbf
			j++
		}
		if need > 0 && j-i == need+1 {
			b.Write(p[i:j])
		} else {
			b.WriteRune(utf8.RuneError)
		}
		i = j
	}
	return b.String()
}
