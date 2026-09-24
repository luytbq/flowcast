package model

import (
	"fmt"
	"github.com/luytbq/flowcast/internal/unistr"
	"strings"
)

// Mức độ của một Issue.
const (
	LevelError   = "error"
	LevelWarning = "warning"
)

// Location nói phát hiện nằm ở đâu trong nguồn.
//
// Có cấu trúc chứ không phải chuỗi, để web trỏ đúng ô và đúng dòng. String là
// dạng CLI in ra.
type Location struct {
	Kind   string `json:"kind"`             // "table" hoặc "text"
	Row    int    `json:"row,omitempty"`    // table: số dòng trong file, 0 nghĩa là không xác định
	Sheet  string `json:"sheet,omitempty"`  // xlsx
	Cell   string `json:"cell,omitempty"`   // xlsx: địa chỉ ô, ví dụ B7, hoặc vùng ô gộp như A1:B2
	Column string `json:"column,omitempty"` // table: tên cột, khi phát hiện quy được về một ô
	Line   int    `json:"line,omitempty"`   // text: số dòng trong nguồn, dùng cho mermaid
}

// LineLoc là vị trí của một dòng bảng trong file văn bản.
func LineLoc(row int) Location { return Location{Kind: "table", Row: row} }

func (l Location) String() string {
	switch {
	case l.Sheet != "" && l.Cell != "":
		return l.Sheet + "!" + l.Cell
	case l.Sheet != "" && l.Row > 0:
		return fmt.Sprintf("%s dòng %d", l.Sheet, l.Row)
	case l.Sheet != "":
		return l.Sheet
	case l.Kind == "text" && l.Line > 0:
		return fmt.Sprintf("dòng %d", l.Line)
	case l.Row > 0:
		return fmt.Sprintf("dòng %d", l.Row)
	}
	return ""
}

// Issue là một phát hiện trả về cho caller.
//
// Code là mã máy ổn định, dành cho caller phân loại và cho web trả JSON. Msg
// là thông điệp tiếng Việt cho người đọc. Core không in Issue ra; caller tự
// trình bày.
type Issue struct {
	Code   string
	Level  string
	Loc    Location
	ID     string
	Msg    string
	Params map[string]string
}

// String là dạng CLI in ra cho một Issue.
func (i Issue) String() string {
	tag := ""
	if i.ID != "" {
		tag = " [" + i.ID + "]"
	}
	loc := i.Loc.String()
	if loc == "" {
		loc = "-"
	}
	return fmt.Sprintf("%-7s %s%s: %s", strings.ToUpper(i.Level), loc, tag, i.Msg)
}

func joinLines(lines []string) string { return strings.Join(lines, "\n") }
func trimSpace(s string) string       { return unistr.Strip(s) }

// Error là lỗi khiến việc đọc không thể tiếp tục và không quy được về một dòng
// cụ thể: không tìm thấy header, file hỏng, vượt giới hạn. Mọi thứ khác là
// Issue.
type Error struct {
	Code string
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

func Errf(code, format string, a ...any) error {
	return &Error{Code: code, Msg: fmt.Sprintf(format, a...)}
}
