package source

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/luytbq/flowcast/model"
)

// ParseMarkdown đọc bảng đầu tiên có đủ năm cột header trong một tài liệu
// markdown.
//
// Tiêu đề lấy từ heading cấp một đầu tiên. Không có thì Title để rỗng và caller
// tự lùi về tên nguồn.
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
			"không tìm thấy bảng có header: %s", strings.Join(Header, " | "))
	}
	if start+1 >= len(lines) || !isSeparator(lines[start+1]) {
		return model.Table{}, model.Errf("source.no_separator",
			"dòng %d: thiếu dòng phân cách |---| sau header", start+2)
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
				Msg: fmt.Sprintf("dòng có %d cột, cần 5 (gạch đứng trong nội dung phải viết \\|)",
					len(cells)),
			})
			continue
		}
		meta := parseMeta(cells[4], loc, cells[0], &issues, unescape)
		rows = append(rows, model.Row{
			Idx:    len(rows),
			Loc:    loc,
			ID:     cells[0],
			Type:   strings.ToLower(cells[1]),
			Parent: cells[2],
			Lines:  mdLines(cells[3]),
			Meta:   meta,
		})
	}
	return model.Table{Title: title, Rows: rows, Issues: issues, Source: "markdown"}, nil
}

// h1Title nhận đúng heading cấp một: dấu thăng ở đầu dòng, rồi ít nhất một
// khoảng trắng, rồi nội dung. "##" và "#không-khoảng-trắng" đều không tính.
func h1Title(ln string) (string, bool) {
	if !strings.HasPrefix(ln, "#") {
		return "", false
	}
	rest := ln[1:]
	trimmed := strings.TrimLeftFunc(rest, unicode.IsSpace)
	if len(trimmed) == len(rest) {
		return "", false
	}
	t := strings.TrimRightFunc(trimmed, unicode.IsSpace)
	if t == "" {
		return "", false
	}
	return unescape(t), true
}

// isSeparator nhận dòng |---|---| ngay dưới header, kể cả dạng có dấu hai chấm
// canh lề.
func isSeparator(ln string) bool {
	s := strings.TrimSpace(ln)
	if !strings.HasPrefix(s, "|") {
		return false
	}
	rest := s[1:]
	if rest == "" {
		return false
	}
	for _, c := range rest {
		if !unicode.IsSpace(c) && c != ':' && c != '-' && c != '|' {
			return false
		}
	}
	return true
}

// mdLines tách nội dung một ô thành các dòng hiển thị.
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

// parseMeta đọc ô metadata dạng key=value ngăn bằng dấu chấm phẩy.
//
// Khóa không được gỡ escape, chỉ giá trị. Khóa trùng thì cái sau đè cái trước.
func parseMeta(cell string, loc model.Location, rid string, issues *[]model.Issue,
	unesc func(string) string) map[string]string {
	meta := map[string]string{}
	for _, part := range strings.Split(cell, ";") {
		part = strings.TrimSpace(part)
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
				Msg:   fmt.Sprintf("metadata sai cú pháp: %q (cần key=value)", part),
			})
			continue
		}
		meta[strings.TrimSpace(k)] = unesc(strings.TrimSpace(v))
	}
	return meta
}
