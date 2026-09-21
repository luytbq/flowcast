// Package conformance đọc bộ đối chiếu và so bản Go với bản tham chiếu Python.
//
// Mỗi chặng dump vừa là đầu ra cần khớp của một module, vừa là đầu vào của
// module sau. Nhờ vậy port được từng module riêng lẻ: module text lấy chặng
// table làm đầu vào và so với chặng text, không cần parser Go nào tồn tại.
package conformance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Row là một dòng bảng như chặng table của dump ghi lại.
type Row struct {
	Idx       int               `json:"idx"`
	Loc       string            `json:"loc"`
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Parent    string            `json:"parent"`
	Lines     []string          `json:"lines"`
	Meta      map[string]string `json:"meta"`
	MetaOrder []string          `json:"meta_order"`
}

type DumpIssue struct {
	Level string `json:"level"`
	Loc   string `json:"loc"`
	ID    string `json:"id"`
	Msg   string `json:"msg"`
}

type TableStage struct {
	Title  string      `json:"title"`
	Source string      `json:"source"`
	Issues []DumpIssue `json:"issues"`
	Rows   []Row       `json:"rows"`
}

// TextBox là kích thước một phần tử trong chặng text. Số là chuỗi vì dump ghi
// chúng qua num.Fmt, để dump và file .drawio chuẩn hóa số theo cùng một quy tắc.
type TextBox struct {
	Kind  string   `json:"kind"`
	Lines []string `json:"lines"`
	W     string   `json:"w"`
	H     string   `json:"h"`
}

type TextLabel struct {
	Lines []string `json:"lines"`
	LW    string   `json:"lw"`
	LH    string   `json:"lh"`
}

type TextStage struct {
	Items map[string]TextBox   `json:"items"`
	Edges map[string]TextLabel `json:"edges"`
}

type PlaceItem struct {
	Lane   int    `json:"lane"`
	Row    int    `json:"row"`
	Col    int    `json:"col"`
	Attach string `json:"attach"`
}

type PlaceStage struct {
	Lanes    []string             `json:"lanes"`
	Topo     []string             `json:"topo"`
	NRows    int                  `json:"nrows"`
	Cols     map[string][]int     `json:"cols"`
	Warnings []string             `json:"warnings"`
	Items    map[string]PlaceItem `json:"items"`
}

// Dump là một file conformance/dumps/<case>.json.
type Dump struct {
	Name   string      // tên case, không có đuôi
	Table  TableStage  `json:"table"`
	Text   *TextStage  `json:"text"`
	Issues []DumpIssue `json:"issues"`
	Place  *PlaceStage `json:"place"`
}

// CaseFile trả về đường dẫn bảng đầu vào của một case.
func CaseFile(dir, name string) string { return filepath.Join(dir, "cases", name+".md") }

// Dir trả về thư mục bộ đối chiếu, tính từ vị trí gói này.
func Dir() string { return "." }

// Load đọc mọi dump, sắp theo tên case.
func Load(dir string) ([]Dump, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "dumps", "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("không có dump nào trong %s/dumps; chạy conformance/generate.py", dir)
	}
	out := make([]Dump, 0, len(paths))
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var d Dump
		if err := json.Unmarshal(data, &d); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		d.Name = strings.TrimSuffix(filepath.Base(p), ".json")
		out = append(out, d)
	}
	return out, nil
}

func trimSpace(s string) string {
	return strings.TrimSpace(s)
}
