package source

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/luytbq/flowcast/model"
)

const (
	nsMain = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
	nsRel  = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
)

// xnode là một phần tử XML đã đọc, đủ cho những gì cần đọc trong file xlsx:
// tên có namespace, thuộc tính, chữ ngay bên trong trước phần tử con đầu tiên,
// và phần tử con.
type xnode struct {
	space, local string
	attrs        []xml.Attr
	text         string
	children     []*xnode
}

func parseXML(data []byte) (*xnode, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	var stack []*xnode
	var root *xnode
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &xnode{space: t.Name.Space, local: t.Name.Local, attrs: t.Attr}
			if len(stack) > 0 {
				p := stack[len(stack)-1]
				p.children = append(p.children, n)
			} else {
				root = n
			}
			stack = append(stack, n)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			// Chỉ giữ phần chữ đứng trước phần tử con đầu tiên; chữ xen giữa
			// các phần tử con không mang nội dung ô.
			if len(stack) > 0 && len(stack[len(stack)-1].children) == 0 {
				stack[len(stack)-1].text += string(t)
			}
		}
	}
	if root == nil {
		return nil, fmt.Errorf("không có phần tử gốc")
	}
	return root, nil
}

func (n *xnode) is(local string) bool { return n.space == nsMain && n.local == local }

// get đọc thuộc tính theo tên không namespace.
func (n *xnode) get(name string) (string, bool) {
	for _, a := range n.attrs {
		if a.Name.Space == "" && a.Name.Local == name {
			return a.Value, true
		}
	}
	return "", false
}

func (n *xnode) getNS(space, name string) string {
	for _, a := range n.attrs {
		if a.Name.Space == space && a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

// iter duyệt n và mọi hậu duệ theo thứ tự tài liệu, như Element.iter.
func (n *xnode) iter(local string, f func(*xnode)) {
	if n.is(local) {
		f(n)
	}
	for _, c := range n.children {
		c.iter(local, f)
	}
}

// find trả về con trực tiếp đầu tiên, như Element.find với một tên đơn.
func (n *xnode) find(local string) *xnode {
	for _, c := range n.children {
		if c.is(local) {
			return c
		}
	}
	return nil
}

func (n *xnode) textOf(local string) string {
	var b strings.Builder
	n.iter(local, func(t *xnode) { b.WriteString(t.text) })
	return b.String()
}

// colIndex đổi phần chữ đầu của một địa chỉ ô như "AB12" thành chỉ số cột từ 0.
func colIndex(ref string) int {
	n := 0
	for _, ch := range ref {
		if !isAlpha(ch) {
			break
		}
		n = n*26 + int(upperASCII(ch)-64)
	}
	return n - 1
}

func isAlpha(r rune) bool { return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') }

func upperASCII(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 32
	}
	return r
}

// numericZero nhận số như "7.0" mà Excel lưu cho một số nguyên.
var numericZero = regexp.MustCompile(`^-?\d+\.0+$`)

type sheetNote struct {
	issue model.Issue
	row   int
}

// xlsxGrid dựng lưới ô của một sheet, kèm cảnh báo về ô công thức rỗng, ô kiểu
// số ở cột id hoặc parent, hàng ẩn và ô gộp. Cảnh báo được lọc về vùng bảng sau.
func xlsxGrid(sheet *xnode, shared []string, name string) ([][]string, []model.Location, []sheetNote) {
	var notes []sheetNote
	rows := map[int]map[int]string{}
	note := func(cell string, row int, id, msg string) {
		notes = append(notes, sheetNote{model.Issue{Code: "source.xlsx_cell", Level: model.LevelWarning,
			Loc: model.Location{Kind: "table", Sheet: name, Cell: cell, Row: row}, ID: id, Msg: msg}, row})
	}
	sheet.iter("row", func(rowEl *xnode) {
		r := len(rows) + 1
		if v, ok := rowEl.get("r"); ok {
			if n, err := strconv.Atoi(v); err == nil {
				r = n
			}
		}
		cells := map[int]string{}
		rowEl.iter("c", func(c *xnode) {
			ref, _ := c.get("r")
			ci := len(cells)
			if ref != "" {
				ci = colIndex(ref)
			}
			t, hasT := c.get("t")
			v := c.find("v")
			val := ""
			switch {
			case t == "s":
				if v != nil && v.text != "" {
					if i, err := strconv.Atoi(v.text); err == nil && i >= 0 && i < len(shared) {
						val = shared[i]
					}
				}
			case t == "inlineStr":
				if is := c.find("is"); is != nil {
					val = is.textOf("t")
				}
			case v != nil:
				val = v.text
				if (!hasT || t == "n") && numericZero.MatchString(val) {
					val = val[:strings.IndexByte(val, '.')]
				}
			default:
				if c.find("f") != nil {
					note(ref, r, "", "ô công thức chưa có giá trị lưu sẵn, đọc thành rỗng")
				}
			}
			if val != "" && (!hasT || t == "n") && (ci == 0 || ci == 2) {
				note(ref, r, val, "ô kiểu số ở cột id/parent; định dạng cột là Text để id như 4.10 không bị đổi")
			}
			cells[ci] = val
		})
		rows[r] = cells
		if h, _ := rowEl.get("hidden"); h == "1" {
			notes = append(notes, sheetNote{model.Issue{Code: "source.xlsx_hidden", Level: model.LevelWarning,
				Loc: model.Location{Kind: "table", Sheet: name, Row: r}, Msg: "hàng đang bị ẩn nhưng vẫn được đọc"}, r})
		}
	})
	sheet.iter("mergeCell", func(m *xnode) {
		ref, _ := m.get("ref")
		first := strings.Split(ref, ":")[0]
		digits := regexp.MustCompile(`\D`).ReplaceAllString(first, "")
		r, _ := strconv.Atoi(digits)
		note(ref, r, "", "ô gộp: chỉ ô trên cùng bên trái giữ giá trị")
	})

	top := 0
	for r := range rows {
		top = max(top, r)
	}
	var grid [][]string
	var locs []model.Location
	for r := 1; r <= top; r++ {
		cells := rows[r]
		width := 0
		for i := range cells {
			width = max(width, i+1)
		}
		line := make([]string, width)
		for i := range line {
			line[i] = cells[i]
		}
		grid = append(grid, line)
		locs = append(locs, model.Location{Kind: "table", Sheet: name, Row: r})
	}
	return grid, locs, notes
}

// ParseXLSX đọc bảng từ sheet đầu tiên có hàng header, hoặc từ sheet chỉ định.
func ParseXLSX(data []byte, name, sheet string) (model.Table, error) {
	return parseXLSX(data, name, sheet, 0)
}

func parseXLSX(data []byte, name, sheet string, maxUnzipped int64) (model.Table, error) {
	var unzipped int64
	over := false
	t, err := readXLSX(data, name, sheet, func(rc io.Reader) ([]byte, error) {
		if maxUnzipped <= 0 {
			return io.ReadAll(rc)
		}
		b, err := io.ReadAll(io.LimitReader(rc, maxUnzipped-unzipped+1))
		unzipped += int64(len(b))
		if unzipped > maxUnzipped {
			over = true
			return nil, errors.New("vượt giới hạn giải nén")
		}
		return b, err
	})
	if over {
		return model.Table{}, model.Errf("limit.unzipped",
			"%s giải nén ra quá %d byte; file xlsx lớn bất thường hoặc hỏng", name, maxUnzipped)
	}
	return t, err
}

func readXLSX(data []byte, name, sheet string, readAll func(io.Reader) ([]byte, error)) (model.Table, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return model.Table{}, model.Errf("source.xlsx", "không đọc được %s: không phải file xlsx hợp lệ", name)
	}
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	read := func(p string) ([]byte, bool) {
		f, ok := files[p]
		if !ok {
			return nil, false
		}
		rc, err := f.Open()
		if err != nil {
			return nil, false
		}
		defer rc.Close()
		b, err := readAll(rc)
		return b, err == nil
	}
	parse := func(p string) (*xnode, error) {
		b, ok := read(p)
		if !ok {
			return nil, model.Errf("source.xlsx", "không đọc được %s: thiếu %s", name, p)
		}
		n, err := parseXML(b)
		if err != nil {
			return nil, model.Errf("source.xlsx", "không đọc được %s: %s: %v", name, p, err)
		}
		return n, nil
	}

	var shared []string
	if _, ok := files["xl/sharedStrings.xml"]; ok {
		sst, err := parse("xl/sharedStrings.xml")
		if err != nil {
			return model.Table{}, err
		}
		for _, si := range sst.children {
			shared = append(shared, si.textOf("t"))
		}
	}
	rels := map[string]string{}
	relsXML, err := parse("xl/_rels/workbook.xml.rels")
	if err != nil {
		return model.Table{}, err
	}
	for _, rel := range relsXML.children {
		id, _ := rel.get("Id")
		target, _ := rel.get("Target")
		rels[id] = target
	}
	wb, err := parse("xl/workbook.xml")
	if err != nil {
		return model.Table{}, err
	}
	type entry struct{ name, target string }
	var sheets []entry
	wb.iter("sheet", func(sh *xnode) {
		target := rels[sh.getNS(nsRel, "id")]
		if strings.HasPrefix(target, "/xl/") {
			target = target[1:]
		} else {
			target = "xl/" + strings.TrimLeft(target, "/")
		}
		n, _ := sh.get("name")
		sheets = append(sheets, entry{n, target})
	})
	if sheet != "" {
		var keep []entry
		for _, s := range sheets {
			if s.name == sheet {
				keep = append(keep, s)
			}
		}
		if len(keep) == 0 {
			return model.Table{}, model.Errf("source.xlsx_sheet", "%s không có sheet tên \"%s\"", name, sheet)
		}
		sheets = keep
	}
	for _, s := range sheets {
		b, ok := read(s.target)
		if !ok {
			continue
		}
		root, err := parseXML(b)
		if err != nil {
			return model.Table{}, model.Errf("source.xlsx", "không đọc được %s: %s: %v", name, s.target, err)
		}
		grid, locs, notes := xlsxGrid(root, shared, s.name)
		if _, _, found := findHeader(grid); !found {
			continue
		}
		title, rows, issues, hr0, err := tableFromGrid(grid, locs)
		if err != nil {
			return model.Table{}, err
		}
		// Chỉ giữ cảnh báo nằm trong vùng bảng: từ hàng header tới hàng cuối.
		hr := hr0 + 1
		for _, n := range notes {
			if hr <= n.row && n.row <= hr+len(rows) {
				issues = append(issues, n.issue)
			}
		}
		return model.Table{Title: title, Rows: rows, Issues: issues, Source: "xlsx sheet " + s.name}, nil
	}
	return model.Table{}, model.Errf("source.no_header", "không tìm thấy sheet nào có hàng header: %s",
		strings.Join(Header, " | "))
}
