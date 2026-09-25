package source

import (
	"fmt"
	"strings"

	"github.com/luytbq/flowcast/internal/unistr"
	"github.com/luytbq/flowcast/model"
)

// findHeader finds the first header row: five adjacent cells holding exactly the
// five column names, starting at any column. This lets csv and xlsx have extra
// columns on the left.
func findHeader(grid [][]string) (row, col int, ok bool) {
	for r, cells := range grid {
		low := make([]string, len(cells))
		for i, c := range cells {
			low[i] = unistr.Lower(unistr.Strip(c))
		}
		for c0 := 0; c0 < max(1, len(low)-4); c0++ {
			if c0+5 <= len(low) && equalStrings(low[c0:c0+5], Header) {
				return r, c0, true
			}
		}
	}
	return 0, 0, false
}

// plainLines splits the content of a csv or xlsx cell into lines: both real line
// breaks and br tags break lines, and markdown-style escapes are not processed.
func plainLines(cell string) []string {
	if cell == "" {
		return []string{""}
	}
	var parts []string
	for _, chunk := range splitBR(cell) {
		chunk = strings.ReplaceAll(strings.ReplaceAll(chunk, "\r\n", "\n"), "\r", "\n")
		parts = append(parts, strings.Split(chunk, "\n")...)
	}
	for i, p := range parts {
		parts[i] = normalize(unistr.Strip(p))
	}
	return parts
}

// tableFromGrid reads the table below the header row, ending at the first row
// whose five cells are all empty. The title is the first non-empty cell above
// the header.
//
// locs is the location of each grid row, so findings point at the right line of
// the file.
func tableFromGrid(grid [][]string, locs []model.Location) (title string, rows []model.Row,
	issues []model.Issue, headerRow int, err error) {
	hr, hc, ok := findHeader(grid)
	if !ok {
		return "", nil, nil, 0, model.Errf("source.no_header", "header row not found: %s",
			strings.Join(Header, " | "))
	}
	for r := 0; r < hr && title == ""; r++ {
		for _, c := range grid[r] {
			if s := unistr.Strip(c); s != "" {
				title = normalize(s)
				break
			}
		}
	}
	for r := hr + 1; r < len(grid); r++ {
		row := grid[r]
		var cells [5]string
		empty := true
		for i := range cells {
			if hc+i < len(row) {
				cells[i] = unistr.Strip(row[hc+i])
			}
			empty = empty && cells[i] == ""
		}
		if empty {
			break
		}
		if strings.Contains(cells[3], `\|`) || strings.Contains(cells[4], `\|`) {
			issues = append(issues, model.Issue{Code: "table.markdown_escape", Level: model.LevelWarning,
				Loc: locs[r], ID: cells[0], Msg: `content still has a markdown escape (\|); csv/xlsx needs no escaping`})
		}
		meta, keys := parseMeta(cells[4], locs[r], cells[0], &issues, normalize)
		rows = append(rows, model.Row{Idx: len(rows), Loc: locs[r], ID: cells[0], Type: unistr.Lower(cells[1]),
			Parent: cells[2], Lines: plainLines(cells[3]), Meta: meta, MetaKeys: keys})
	}
	return title, rows, issues, hr, nil
}

var delimiterNames = map[string]string{",": "comma", ";": "semicolon", "\t": "tab"}

// ParseCSV reads the table from a csv file.
//
// When not specified, the encodings tried in turn are utf-8-sig, utf-8, cp1252,
// and the delimiters tried are comma, semicolon, tab; the first one that yields a
// header row wins.
func ParseCSV(data []byte, name, delimiter, encoding string) (model.Table, error) {
	encs := csvEncodings
	if encoding != "" {
		encs = []string{encoding}
	}
	text, used := "", ""
	for _, enc := range encs {
		if t, ok := decode(data, enc); ok {
			text, used = t, enc
			break
		}
	}
	if used == "" {
		return model.Table{}, model.Errf("source.decode", "cannot decode %s; try --encoding", name)
	}
	delims := []string{",", ";", "\t"}
	if delimiter != "" {
		delims = []string{delimiter}
	}
	var grid [][]string
	for _, d := range delims {
		r := []rune(d)
		if len(r) != 1 {
			return model.Table{}, model.Errf("source.delimiter", "delimiter must be exactly one character, got %q", d)
		}
		cand, err := readCSV(text, r[0])
		if err != nil {
			return model.Table{}, model.Errf("source.csv", "%s", err)
		}
		if _, _, ok := findHeader(cand); ok {
			grid, delimiter = cand, d
			break
		}
	}
	if grid == nil {
		return model.Table{}, model.Errf("source.no_header",
			"header row not found in csv; check the delimiter, or use --delimiter")
	}
	locs := make([]model.Location, len(grid))
	for i := range grid {
		locs[i] = model.LineLoc(i + 1)
	}
	title, rows, issues, _, err := tableFromGrid(grid, locs)
	if err != nil {
		return model.Table{}, err
	}
	// Compare the literal name, not the normalized one: a user who writes
	// --encoding CP1252 knows what they chose, so no warning is needed.
	if used == "cp1252" {
		issues = append(issues, model.Issue{Code: "source.cp1252", Level: model.LevelWarning,
			Msg: "file is not utf-8, read as cp1252; Vietnamese text may already have been corrupted"})
	}
	shown, ok := delimiterNames[delimiter]
	if !ok {
		shown = delimiter
	}
	return model.Table{Title: title, Rows: rows, Issues: issues,
		Source: fmt.Sprintf("csv (%s, %s)", shown, used)}, nil
}
