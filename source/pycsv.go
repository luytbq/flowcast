package source

import (
	"fmt"
	"strings"
)

// fieldLimit là giới hạn độ dài một ô của csv.reader trong Python.
const fieldLimit = 131072

type csvState int

const (
	startRecord csvState = iota
	startField
	inField
	inQuotedField
	quoteInQuotedField
	eatCRNL
)

// readCSV đọc csv đúng như csv.reader(io.StringIO(text, newline=”),
// delimiter=d) của Python với dialect mặc định.
//
// Không dùng encoding/csv: nó hiểu dấu nháy khác Python ở nhiều góc. Chẳng hạn
// "a"b ra ab trong Python nhưng ra a"b trong Go, và dòng trống ra một hàng rỗng
// trong Python nhưng bị bỏ qua trong Go. Đây là máy trạng thái của _csv.c, chỉ
// giữ các nhánh dùng tới khi không có escapechar, doublequote bật và strict tắt.
// Đã kiểm trên conformance/csv-vectors.json.
func readCSV(text string, d rune) ([][]string, error) {
	var (
		records [][]string
		fields  []string
		field   []rune
		state   = startRecord
	)
	save := func() {
		fields = append(fields, string(field))
		field = field[:0]
	}
	add := func(c rune) error {
		if len(field) >= fieldLimit {
			return fmt.Errorf("field larger than field limit (%d)", fieldLimit)
		}
		field = append(field, c)
		return nil
	}
	const eol = -1
	process := func(c rune) error {
		newline := c == '\n' || c == '\r'
		switch state {
		case startRecord:
			if c == eol {
				return nil // dòng trống: một hàng rỗng
			}
			if newline {
				state = eatCRNL
				return nil
			}
			state = startField
			fallthrough
		case startField:
			switch {
			case newline || c == eol:
				save()
				state = map[bool]csvState{true: startRecord, false: eatCRNL}[c == eol]
			case c == '"':
				state = inQuotedField
			case c == d:
				save()
			default:
				state = inField
				return add(c)
			}
		case inField:
			switch {
			case newline || c == eol:
				save()
				state = map[bool]csvState{true: startRecord, false: eatCRNL}[c == eol]
			case c == d:
				save()
				state = startField
			default:
				return add(c)
			}
		case inQuotedField:
			switch {
			case c == eol:
				// Dòng hết khi đang trong dấu nháy: ô tiếp tục sang dòng sau.
			case c == '"':
				state = quoteInQuotedField
			default:
				return add(c)
			}
		case quoteInQuotedField:
			switch {
			case c == '"':
				state = inQuotedField
				return add(c)
			case c == d:
				save()
				state = startField
			case newline || c == eol:
				save()
				state = map[bool]csvState{true: startRecord, false: eatCRNL}[c == eol]
			default:
				// Không strict: chữ sau dấu nháy đóng được nối vào ô.
				state = inField
				return add(c)
			}
		case eatCRNL:
			switch {
			case newline:
			case c == eol:
				state = startRecord
			default:
				return fmt.Errorf("new-line character seen in unquoted field - do you need to open the file with newline=''?")
			}
		}
		return nil
	}

	lines := splitKeepEnds(text)
	li := 0
	for {
		fields, field, state = nil, field[:0], startRecord
		for {
			if li >= len(lines) {
				if len(field) != 0 || state == inQuotedField {
					save()
					return append(records, fields), nil
				}
				return records, nil
			}
			for _, c := range lines[li] {
				if err := process(c); err != nil {
					return nil, err
				}
			}
			li++
			if err := process(eol); err != nil {
				return nil, err
			}
			if state == startRecord {
				break
			}
		}
		if fields == nil {
			fields = []string{}
		}
		records = append(records, fields)
	}
}

// splitKeepEnds cắt dòng như khi lặp một io.StringIO(text, newline=”) của
// Python: giữ nguyên ký tự xuống dòng ở cuối mỗi dòng, và nhận cả LF, CR lẫn
// CRLF làm ranh giới.
func splitKeepEnds(s string) []string {
	var out []string
	for len(s) > 0 {
		i := strings.IndexAny(s, "\r\n")
		if i < 0 {
			out = append(out, s)
			break
		}
		end := i + 1
		if s[i] == '\r' && end < len(s) && s[end] == '\n' {
			end++
		}
		out = append(out, s[:end])
		s = s[end:]
	}
	return out
}
