package source

import (
	"fmt"
	"strings"

	"github.com/luytbq/flowcast/internal/unistr"
	"github.com/luytbq/flowcast/model"
)

// findHeader tìm hàng header đầu tiên: năm ô liền nhau mang đúng năm tên cột,
// bắt đầu ở cột bất kỳ. Nhờ vậy csv và xlsx được phép có cột thừa bên trái.
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

// plainLines tách nội dung một ô csv hoặc xlsx thành các dòng: xuống dòng thật
// và thẻ br đều là xuống dòng, và không xử lý escape kiểu markdown.
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

// tableFromGrid đọc bảng nằm dưới hàng header, kết thúc ở hàng đầu tiên có cả
// năm ô đều rỗng. Tiêu đề là ô không rỗng đầu tiên phía trên header.
//
// locs là vị trí của từng hàng trong lưới, để phát hiện trỏ đúng dòng của file.
func tableFromGrid(grid [][]string, locs []model.Location) (title string, rows []model.Row,
	issues []model.Issue, headerRow int, err error) {
	hr, hc, ok := findHeader(grid)
	if !ok {
		return "", nil, nil, 0, model.Errf("source.no_header", "không tìm thấy hàng header: %s",
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
				Loc: locs[r], ID: cells[0], Msg: `nội dung còn escape kiểu markdown (\|); csv/xlsx không cần escape`})
		}
		meta, keys := parseMeta(cells[4], locs[r], cells[0], &issues, normalize)
		rows = append(rows, model.Row{Idx: len(rows), Loc: locs[r], ID: cells[0], Type: unistr.Lower(cells[1]),
			Parent: cells[2], Lines: plainLines(cells[3]), Meta: meta, MetaKeys: keys})
	}
	return title, rows, issues, hr, nil
}

var delimiterNames = map[string]string{",": "dấu phẩy", ";": "dấu chấm phẩy", "\t": "tab"}

// ParseCSV đọc bảng từ một file csv.
//
// Không chỉ định thì bảng mã thử lần lượt utf-8-sig, utf-8, cp1252, còn dấu phân
// cách thử dấu phẩy, dấu chấm phẩy, tab, và lấy cái đầu tiên cho ra hàng header.
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
		return model.Table{}, model.Errf("source.decode", "không giải mã được %s; thử --encoding", name)
	}
	delims := []string{",", ";", "\t"}
	if delimiter != "" {
		delims = []string{delimiter}
	}
	var grid [][]string
	for _, d := range delims {
		r := []rune(d)
		if len(r) != 1 {
			return model.Table{}, model.Errf("source.delimiter", "dấu phân cách phải là đúng một ký tự, nhận %q", d)
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
			"không tìm thấy hàng header trong csv; kiểm tra dấu phân cách, hoặc dùng --delimiter")
	}
	locs := make([]model.Location, len(grid))
	for i := range grid {
		locs[i] = model.LineLoc(i + 1)
	}
	title, rows, issues, _, err := tableFromGrid(grid, locs)
	if err != nil {
		return model.Table{}, err
	}
	// So nguyên văn chứ không so tên đã chuẩn hóa: người dùng tự ghi
	// --encoding CP1252 là đã biết mình chọn gì, nên không cần cảnh báo.
	if used == "cp1252" {
		issues = append(issues, model.Issue{Code: "source.cp1252", Level: model.LevelWarning,
			Msg: "file không phải utf-8, đã đọc theo cp1252; chữ tiếng Việt có thể đã hỏng từ trước"})
	}
	shown, ok := delimiterNames[delimiter]
	if !ok {
		shown = delimiter
	}
	return model.Table{Title: title, Rows: rows, Issues: issues,
		Source: fmt.Sprintf("csv (%s, %s)", shown, used)}, nil
}
