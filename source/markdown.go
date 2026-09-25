package source

import (
	"fmt"
	"github.com/luytbq/flowcast/internal/unistr"
	"strings"

	"github.com/luytbq/flowcast/model"
)

// ParseMarkdown reads the first table with all five header columns in a
// markdown document.
//
// The title comes from the first level-one heading. Without one, Title is left
// empty and the caller falls back to the source name.
func ParseMarkdown(data []byte) (model.Table, error) {
	lines := splitLines(string(data))

	title := ""
	for _, ln := range lines {
		if t, ok := h1Title(ln); ok {
			title = t
			break
		}
	}

	start := -1
	for i, ln := range lines {
		if strings.HasPrefix(trimLeftSpace(ln), "|") && equalStrings(lower(splitCells(ln)), Header) {
			start = i
			break
		}
	}
	if start < 0 {
		return model.Table{}, model.Errf("source.no_header",
			"no table found with header: %s", strings.Join(Header, " | "))
	}
	if start+1 >= len(lines) || !isSeparator(lines[start+1]) {
		return model.Table{}, model.Errf("source.no_separator",
			"line %d: missing |---| separator line after header", start+2)
	}

	var rows []model.Row
	var issues []model.Issue
	for i := start + 2; i < len(lines) && strings.HasPrefix(trimLeftSpace(lines[i]), "|"); i++ {
		loc := model.LineLoc(i + 1)
		cells := splitCells(lines[i])
		if len(cells) != 5 {
			issues = append(issues, model.Issue{
				Code:  "table.cell_count",
				Level: model.LevelError,
				Loc:   loc,
				Msg: fmt.Sprintf("row has %d columns, needs 5 (a pipe in content must be written \\|)",
					len(cells)),
			})
			continue
		}
		meta, metaKeys := parseMeta(cells[4], loc, cells[0], &issues, unescape)
		rows = append(rows, model.Row{
			Idx:      len(rows),
			Loc:      loc,
			ID:       cells[0],
			Type:     unistr.Lower(cells[1]),
			Parent:   cells[2],
			Lines:    mdLines(cells[3]),
			Meta:     meta,
			MetaKeys: metaKeys,
		})
	}
	return model.Table{Title: title, Rows: rows, Issues: issues, Source: "markdown"}, nil
}

// h1Title accepts exactly a level-one heading per the reference implementation's
// expression ^#\s+(.+?)\s*$: a hash at the start of the line, at least one space,
// then content. "##" and "#no-space" do not count.
//
// A line made of the hash and two or more spaces still matches, with the title
// being exactly the last space: \s+ backtracks one character so the group (.+?)
// has something to match.
func h1Title(ln string) (string, bool) {
	if !strings.HasPrefix(ln, "#") {
		return "", false
	}
	rest := []rune(ln[1:])
	if len(rest) < 2 || !unistr.IsSpace(rest[0]) {
		return "", false
	}
	i := 0
	for i < len(rest) && unistr.IsSpace(rest[i]) {
		i++
	}
	if i == len(rest) {
		return unescape(string(rest[len(rest)-1:])), true
	}
	return unescape(unistr.RStrip(string(rest[i:]))), true
}

// isSeparator accepts the |---|---| line right below the header, that is a line
// matching ^\s*\|[\s:\-|]+\|?\s*$. Only leading whitespace is trimmed:
// trailing whitespace belongs to the valid character set, so "| " is also a
// separator line.
func isSeparator(ln string) bool {
	s := unistr.LStrip(ln)
	if !strings.HasPrefix(s, "|") || len(s) == 1 {
		return false
	}
	for _, c := range s[1:] {
		if !unistr.IsSpace(c) && c != ':' && c != '-' && c != '|' {
			return false
		}
	}
	return true
}

// mdLines splits the content of a cell into display lines.
func mdLines(cell string) []string {
	if cell == "" {
		return []string{""}
	}
	parts := splitBR(cell)
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = unescape(p)
	}
	return out
}

// parseMeta reads a metadata cell of key=value pairs separated by semicolons.
//
// Keys are not unescaped, only values. For a repeated key the later one
// overrides the earlier.
func parseMeta(cell string, loc model.Location, rid string, issues *[]model.Issue,
	unesc func(string) string) (map[string]string, []string) {
	meta := map[string]string{}
	var order []string
	for _, part := range strings.Split(cell, ";") {
		part = unistr.Strip(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			*issues = append(*issues, model.Issue{
				Code:  "schema.meta_syntax",
				Level: model.LevelError,
				Loc:   loc,
				ID:    rid,
				Msg:   fmt.Sprintf("invalid metadata syntax: \"%s\" (expected key=value)", part),
			})
			continue
		}
		key := unistr.Strip(k)
		// A repeated key overrides the value but keeps the position of its first occurrence.
		if _, seen := meta[key]; !seen {
			order = append(order, key)
		}
		meta[key] = unesc(unistr.Strip(v))
	}
	return meta, order
}
