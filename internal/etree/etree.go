// Package etree reads and writes XML as an element tree.
//
// Merge copies the cells the user drew by hand verbatim from the old .drawio file
// into the new one. So that a file can go through merge many times without
// drifting, the XML tree is read and written by these fixed rules:
//
//   - Text is the text before the first child element, Tail is the text after
//     that element. Comments and processing instructions are dropped, and the
//     text on either side of them is joined.
//   - Tab, newline and CR written literally in attribute values become spaces,
//     as the XML specification requires; when written as numeric references they
//     are kept. encoding/xml does not do this, and after decoding the two forms
//     can no longer be told apart, so normalization runs on the raw bytes first.
//   - Attributes keep their document order. An element with no children and no
//     text is written as "<tag ... />" with a space before the slash.
//
// Namespaces are not handled: .drawio files do not use them.
package etree

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/luytbq/flowcast/internal/unistr"
)

// Element is an XML element.
type Element struct {
	Tag      string
	Attrs    [][2]string // in document order
	Text     string
	Tail     string
	Children []*Element
}

// New builds an element with attribute key and value pairs.
func New(tag string, kv ...string) *Element {
	e := &Element{Tag: tag}
	for i := 0; i+1 < len(kv); i += 2 {
		e.Attrs = append(e.Attrs, [2]string{kv[i], kv[i+1]})
	}
	return e
}

// Add appends a child element, like ET.SubElement.
func (e *Element) Add(tag string, kv ...string) *Element {
	c := New(tag, kv...)
	e.Children = append(e.Children, c)
	return c
}

// Get reads an attribute.
func (e *Element) Get(k string) (string, bool) {
	for _, a := range e.Attrs {
		if a[0] == k {
			return a[1], true
		}
	}
	return "", false
}

// Set assigns an attribute. A new attribute is appended at the end, an existing
// one keeps its position.
func (e *Element) Set(k, v string) {
	for i := range e.Attrs {
		if e.Attrs[i][0] == k {
			e.Attrs[i][1] = v
			return
		}
	}
	e.Attrs = append(e.Attrs, [2]string{k, v})
}

// Del removes an attribute.
func (e *Element) Del(k string) {
	for i := range e.Attrs {
		if e.Attrs[i][0] == k {
			e.Attrs = append(e.Attrs[:i], e.Attrs[i+1:]...)
			return
		}
	}
}

// Find returns the first direct child with tag tag.
func (e *Element) Find(tag string) *Element {
	for _, c := range e.Children {
		if c.Tag == tag {
			return c
		}
	}
	return nil
}

// FindAll returns every direct child with tag tag.
func (e *Element) FindAll(tag string) []*Element {
	var out []*Element
	for _, c := range e.Children {
		if c.Tag == tag {
			out = append(out, c)
		}
	}
	return out
}

// Iter walks e and every descendant with tag tag in document order, like
// Element.iter. An empty tag means every element.
func (e *Element) Iter(tag string, f func(*Element)) {
	if tag == "" || e.Tag == tag {
		f(e)
	}
	for _, c := range e.Children {
		c.Iter(tag, f)
	}
}

// Remove removes a direct child.
func (e *Element) Remove(c *Element) {
	for i, x := range e.Children {
		if x == c {
			e.Children = append(e.Children[:i], e.Children[i+1:]...)
			return
		}
	}
}

// Copy makes a deep copy, like copy.deepcopy.
func (e *Element) Copy() *Element {
	c := &Element{Tag: e.Tag, Text: e.Text, Tail: e.Tail, Attrs: append([][2]string(nil), e.Attrs...)}
	for _, ch := range e.Children {
		c.Children = append(c.Children, ch.Copy())
	}
	return c
}

// normalizeAttrs replaces tab, newline and CR written literally in attribute
// values with spaces, skipping comments, CDATA and processing instructions.
func normalizeAttrs(data []byte) []byte {
	var out bytes.Buffer
	n := len(data)
	skip := func(i int, open, close string) (int, bool) {
		if !bytes.HasPrefix(data[i:], []byte(open)) {
			return i, false
		}
		j := bytes.Index(data[i+len(open):], []byte(close))
		if j < 0 {
			out.Write(data[i:])
			return n, true
		}
		end := i + len(open) + j + len(close)
		out.Write(data[i:end])
		return end, true
	}
	for i := 0; i < n; {
		if j, ok := skip(i, "<!--", "-->"); ok {
			i = j
			continue
		}
		if j, ok := skip(i, "<![CDATA[", "]]>"); ok {
			i = j
			continue
		}
		if j, ok := skip(i, "<?", "?>"); ok {
			i = j
			continue
		}
		if data[i] != '<' {
			out.WriteByte(data[i])
			i++
			continue
		}
		var quote byte
		for i < n {
			c := data[i]
			i++
			if quote == 0 {
				out.WriteByte(c)
				if c == '"' || c == '\'' {
					quote = c
				} else if c == '>' {
					break
				}
				continue
			}
			switch c {
			case quote:
				quote = 0
				out.WriteByte(c)
			case '\r':
				// Line breaks are normalized first, so CRLF becomes a single space.
				out.WriteByte(' ')
				if i < n && data[i] == '\n' {
					i++
				}
			case '\n', '\t':
				out.WriteByte(' ')
			default:
				out.WriteByte(c)
			}
		}
	}
	return out.Bytes()
}

// Parse reads an XML document, like ET.fromstring.
func Parse(data []byte) (*Element, error) {
	d := xml.NewDecoder(bytes.NewReader(normalizeAttrs(data)))
	var stack []*Element
	var root *Element
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
			e := &Element{Tag: t.Name.Local}
			for _, a := range t.Attr {
				e.Attrs = append(e.Attrs, [2]string{a.Name.Local, a.Value})
			}
			if len(stack) > 0 {
				p := stack[len(stack)-1]
				p.Children = append(p.Children, e)
			} else if root == nil {
				root = e
			}
			stack = append(stack, e)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) == 0 {
				continue // text outside the root element does not belong to the tree
			}
			cur := stack[len(stack)-1]
			if len(cur.Children) == 0 {
				cur.Text += string(t)
			} else {
				cur.Children[len(cur.Children)-1].Tail += string(t)
			}
		}
	}
	if root == nil {
		return nil, fmt.Errorf("no element found")
	}
	return root, nil
}

var (
	attrEsc = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;",
		"\r", "&#13;", "\n", "&#10;", "\t", "&#09;")
	textEsc = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
)

// String writes the tree like ET.tostring(e, encoding="unicode"): no XML
// declaration, and including e's own Tail.
func (e *Element) String() string {
	var b strings.Builder
	e.write(&b)
	return b.String()
}

func (e *Element) write(b *strings.Builder) {
	b.WriteString("<" + e.Tag)
	for _, a := range e.Attrs {
		b.WriteString(" " + a[0] + "=\"" + attrEsc.Replace(a[1]) + "\"")
	}
	if e.Text != "" || len(e.Children) > 0 {
		b.WriteString(">" + textEsc.Replace(e.Text))
		for _, c := range e.Children {
			c.write(b)
		}
		b.WriteString("</" + e.Tag + ">")
	} else {
		b.WriteString(" />")
	}
	b.WriteString(textEsc.Replace(e.Tail))
}

// Indent indents like ET.indent(tree, space="  "): it only overwrites Text and
// Tail when they are empty or all whitespace, so the user's real text is kept.
func Indent(root *Element) {
	if len(root.Children) == 0 {
		return
	}
	levels := []string{"\n"}
	var rec func(e *Element, level int)
	rec = func(e *Element, level int) {
		if level+1 >= len(levels) {
			levels = append(levels, levels[level]+"  ")
		}
		child := levels[level+1]
		if unistr.Strip(e.Text) == "" {
			e.Text = child
		}
		for _, c := range e.Children {
			if len(c.Children) > 0 {
				rec(c, level+1)
			}
			if unistr.Strip(c.Tail) == "" {
				c.Tail = child
			}
		}
		if last := e.Children[len(e.Children)-1]; unistr.Strip(last.Tail) == "" {
			last.Tail = levels[level]
		}
	}
	rec(root, 0)
}
