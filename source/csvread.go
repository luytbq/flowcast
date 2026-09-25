package source

import (
	"fmt"
	"strings"
)

// fieldLimit is the maximum length of one cell, so a broken file missing a
// closing quote does not swallow the whole file into one cell.
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

// readCSV reads csv with delimiter d, the way spreadsheets export it: double
// quotes enclose a cell, two adjacent quotes inside a cell are one quote, and
// there is no escape character.
//
// encoding/csv is not used because it handles two corners differently from what
// spreadsheet users expect: "a"b yields a"b instead of ab, and a blank line is
// skipped instead of yielding an empty row, while an empty row is what ends the
// table. The behavior is pinned by conformance/csv-vectors.json.
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
				return nil // blank line: one empty row
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
				// Line ends inside quotes: the cell continues on the next line.
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
				// Not strict: text after the closing quote is appended to the cell.
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

// splitKeepEnds splits lines, keeping the line break at the end of each line,
// and accepts LF, CR and CRLF as boundaries.
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
