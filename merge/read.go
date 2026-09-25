// Package merge regenerates a diagram while keeping the manual edits the user
// made in the old .drawio file: node positions, lane widths, wire waypoints, and
// added freehand cells.
//
// Merge is only for the CLI, where the old file sits right next to the table.
// The web service does not accept an old file, so it does not use this package.
package merge

import (
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/luytbq/flowcast/internal/etree"
	"github.com/luytbq/flowcast/internal/unistr"
)

// Old is the first page of the old .drawio file, along with the remaining pages
// kept as is.
type Old struct {
	ids   []string // in order of first appearance
	cells map[string]*oldCell
	Pages []*etree.Element
}

// oldCell is a cell of the first page. elem is the element directly under root:
// the mxCell itself, or the wrapping object element when the cell has custom
// attributes.
type oldCell struct {
	id    string
	elem  *etree.Element
	cell  *etree.Element
	order int
}

func (c *oldCell) attr(k string) string {
	v, _ := c.cell.Get(k)
	return v
}

func (c *oldCell) parent() string   { return c.attr("parent") }
func (c *oldCell) vertex() bool     { return c.attr("vertex") == "1" }
func (c *oldCell) edge() bool       { return c.attr("edge") == "1" }
func (c *oldCell) styleRaw() string { return c.attr("style") }
func (c *oldCell) style() *style    { return parseStyle(c.styleRaw()) }
func (c *oldCell) geo() *etree.Element {
	return c.cell.Find("mxGeometry")
}

func (c *oldCell) points() [][2]float64 {
	g := c.geo()
	if g == nil {
		return nil
	}
	for _, arr := range g.FindAll("Array") {
		if v, _ := arr.Get("as"); v == "points" {
			var out [][2]float64
			for _, p := range arr.FindAll("mxPoint") {
				out = append(out, [2]float64{attrf(p, "x"), attrf(p, "y")})
			}
			return out
		}
	}
	return nil
}

// style is the style of a cell: keys in order of first appearance, and a key
// without an equals sign carries an empty value with set false.
type style struct {
	keys []string
	vals map[string]styleVal
}

type styleVal struct {
	v   string
	set bool
}

func parseStyle(raw string) *style {
	st := &style{vals: map[string]styleVal{}}
	for _, part := range splitSemi(raw) {
		if part == "" {
			continue
		}
		k, v, found := cut(part, '=')
		if _, ok := st.vals[k]; !ok {
			st.keys = append(st.keys, k)
		}
		st.vals[k] = styleVal{v, found}
	}
	return st
}

func (st *style) has(k string) bool { _, ok := st.vals[k]; return ok }

func splitSemi(s string) []string {
	var out []string
	for {
		i := bytes.IndexByte([]byte(s), ';')
		if i < 0 {
			return append(out, s)
		}
		out = append(out, s[:i])
		s = s[i+1:]
	}
}

func cut(s string, sep byte) (string, string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}

// attrf reads a numeric attribute, like float(el.get(key, 0)): missing or
// unreadable means 0.
func attrf(el *etree.Element, key string) float64 {
	if el == nil {
		return 0
	}
	s, ok := el.Get(key)
	if !ok {
		return 0
	}
	v, ok := parseFloat(s)
	if !ok {
		return 0
	}
	return v
}

// ErrRead is an error that prevents merging. The message already includes the
// file name.
type ErrRead struct{ Msg string }

func (e *ErrRead) Error() string { return e.Msg }

// Read reads the old .drawio file. name is only used in error messages.
//
// The first page may be compressed by draw.io: base64 of raw deflate data,
// containing URL-encoded XML.
func Read(data []byte, name string) (*Old, error) {
	root, err := etree.Parse(data)
	if err != nil {
		return nil, &ErrRead{fmt.Sprintf("cannot read %s: %v", name, err)}
	}
	old := &Old{cells: map[string]*oldCell{}}
	model := root
	if root.Tag != "mxGraphModel" {
		pages := root.FindAll("diagram")
		if len(pages) == 0 {
			return nil, &ErrRead{name + " has no pages"}
		}
		model = pages[0].Find("mxGraphModel")
		if model == nil {
			model, err = decodeDiagram(unistr.Strip(pages[0].Text))
			if err != nil {
				return nil, &ErrRead{fmt.Sprintf("cannot decompress the first page of %s: %v", name, err)}
			}
		}
		old.Pages = pages[1:]
	}
	groot := model.Find("root")
	if groot == nil {
		return nil, &ErrRead{name + " has no mxGraphModel/root"}
	}
	for i, el := range groot.Children {
		cell := el
		if el.Tag != "mxCell" {
			if cell = el.Find("mxCell"); cell == nil {
				continue
			}
		}
		cid, _ := el.Get("id")
		if cid == "" {
			continue
		}
		// duplicate id: the later cell replaces the earlier one but keeps the earlier
		// one's slot.
		if _, ok := old.cells[cid]; !ok {
			old.ids = append(old.ids, cid)
		}
		old.cells[cid] = &oldCell{cid, el, cell, i}
	}
	return old, nil
}

func decodeDiagram(text string) (*etree.Element, error) {
	raw, err := b64decode(text)
	if err != nil {
		return nil, err
	}
	xml, err := io.ReadAll(flate.NewReader(bytes.NewReader(raw)))
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(xml) {
		return nil, fmt.Errorf("'utf-8' codec can't decode")
	}
	return etree.Parse([]byte(unquote(string(xml))))
}
