// Package source reads the input formats and reduces them to a model.Table.
//
// It takes bytes rather than paths: the core never touches the filesystem, so
// the same code path serves both the CLI reading files and the web service
// receiving uploads.
package source

import (
	"github.com/luytbq/flowcast/internal/unistr"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// Header is the five required column names. The column count is a
// compatibility commitment, see docs/adr/0001.
var Header = []string{"id", "type", "parent", "content", "metadata"}

// mdEscapable is the set of characters markdown allows a backslash in front of.
const mdEscapable = "\\`*_{}[]()#+-.!|<>~"

// splitLines splits on every line boundary, not only Unix line breaks: LF, CR,
// CRLF, VT, FF, the separators U+001C to U+001E, NEL, and the Unicode line and
// paragraph separators. Files written on Windows use CRLF, and some pasted
// sources contain Unicode paragraph separators.
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

// splitCells splits a markdown table line into cells.
//
// Only a pipe not preceded by a backslash separates columns, so content written
// as \| keeps its pipe without breaking the table.
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

// unescape removes markdown backslashes and then normalizes to NFC.
//
// Normalization is required, not cosmetic: "Xử lý" in decomposed form measures
// 42.43px instead of 30.75px, which changes the layout outright, and macOS often
// produces the decomposed form when copying Vietnamese text. See docs/adr/0005.
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

// splitBR splits a cell at html line break tags, like BR_RE.split of the
// reference implementation with the case-insensitive expression <br\s*/?>.
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

// matchBR returns the number of characters of the br tag at the start of r, or
// 0 if there is none.
//
// It compares character by character instead of lowercasing the whole string
// and searching: lowercasing can change the string length, as İ becomes i plus a
// dot, so positions found in the lowercased string do not cut the original
// correctly. Case-insensitivity applies only to b and r, and matches only B and
// R, no other Unicode character.
// Greedy matching without backtracking is enough: whitespace, the slash and the
// greater-than sign never overlap, so no backtracking could match where the
// greedy match fails.
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
