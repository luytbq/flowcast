// Package etree đọc và ghi XML dưới dạng cây phần tử.
//
// Merge chép nguyên các cell người dùng tự vẽ từ file .drawio cũ sang file mới.
// Để một file đi qua merge nhiều lần mà không trôi, cây XML được đọc và ghi theo
// các quy tắc cố định sau:
//
//   - Text là chữ đứng trước phần tử con đầu tiên, Tail là chữ đứng sau phần tử
//     đó. Chú thích và chỉ thị xử lý bị bỏ, và chữ hai bên chúng dính liền lại.
//   - Tab, xuống dòng và CR viết thẳng trong giá trị thuộc tính thành khoảng
//     trắng, như đặc tả XML quy định; viết bằng tham chiếu số thì được giữ.
//     encoding/xml không làm việc này, và sau khi giải mã thì không còn phân biệt
//     được hai cách viết, nên việc chuẩn hóa chạy trên byte thô trước.
//   - Thuộc tính giữ thứ tự trong tài liệu. Phần tử không có con và không có chữ
//     ghi thành "<tag ... />" với một khoảng trắng trước dấu gạch chéo.
//
// Không xử lý namespace: file .drawio không dùng tới.
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

// Element là một phần tử XML.
type Element struct {
	Tag      string
	Attrs    [][2]string // theo thứ tự trong tài liệu
	Text     string
	Tail     string
	Children []*Element
}

// New dựng một phần tử với các cặp khóa và giá trị thuộc tính.
func New(tag string, kv ...string) *Element {
	e := &Element{Tag: tag}
	for i := 0; i+1 < len(kv); i += 2 {
		e.Attrs = append(e.Attrs, [2]string{kv[i], kv[i+1]})
	}
	return e
}

// Add thêm một phần tử con, như ET.SubElement.
func (e *Element) Add(tag string, kv ...string) *Element {
	c := New(tag, kv...)
	e.Children = append(e.Children, c)
	return c
}

// Get đọc một thuộc tính.
func (e *Element) Get(k string) (string, bool) {
	for _, a := range e.Attrs {
		if a[0] == k {
			return a[1], true
		}
	}
	return "", false
}

// Set gán một thuộc tính. Thuộc tính mới được thêm vào cuối, thuộc tính đã có
// giữ nguyên vị trí.
func (e *Element) Set(k, v string) {
	for i := range e.Attrs {
		if e.Attrs[i][0] == k {
			e.Attrs[i][1] = v
			return
		}
	}
	e.Attrs = append(e.Attrs, [2]string{k, v})
}

// Del xóa một thuộc tính.
func (e *Element) Del(k string) {
	for i := range e.Attrs {
		if e.Attrs[i][0] == k {
			e.Attrs = append(e.Attrs[:i], e.Attrs[i+1:]...)
			return
		}
	}
}

// Find trả về con trực tiếp đầu tiên mang thẻ tag.
func (e *Element) Find(tag string) *Element {
	for _, c := range e.Children {
		if c.Tag == tag {
			return c
		}
	}
	return nil
}

// FindAll trả về mọi con trực tiếp mang thẻ tag.
func (e *Element) FindAll(tag string) []*Element {
	var out []*Element
	for _, c := range e.Children {
		if c.Tag == tag {
			out = append(out, c)
		}
	}
	return out
}

// Iter duyệt e và mọi hậu duệ mang thẻ tag theo thứ tự tài liệu, như
// Element.iter. tag rỗng nghĩa là mọi phần tử.
func (e *Element) Iter(tag string, f func(*Element)) {
	if tag == "" || e.Tag == tag {
		f(e)
	}
	for _, c := range e.Children {
		c.Iter(tag, f)
	}
}

// Remove bỏ một con trực tiếp.
func (e *Element) Remove(c *Element) {
	for i, x := range e.Children {
		if x == c {
			e.Children = append(e.Children[:i], e.Children[i+1:]...)
			return
		}
	}
}

// Copy chép sâu, như copy.deepcopy.
func (e *Element) Copy() *Element {
	c := &Element{Tag: e.Tag, Text: e.Text, Tail: e.Tail, Attrs: append([][2]string(nil), e.Attrs...)}
	for _, ch := range e.Children {
		c.Children = append(c.Children, ch.Copy())
	}
	return c
}

// normalizeAttrs thay tab, xuống dòng và CR viết thẳng trong giá trị thuộc tính
// bằng khoảng trắng, bỏ qua chú thích, CDATA và chỉ thị xử lý.
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
				// Xuống dòng được chuẩn hóa trước, nên CRLF chỉ thành một khoảng trắng.
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

// Parse đọc một tài liệu XML, như ET.fromstring.
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
				continue // chữ ngoài phần tử gốc không thuộc về cây
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

// String ghi cây như ET.tostring(e, encoding="unicode"): không có khai báo XML,
// và có cả Tail của chính e.
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

// Indent thụt lề như ET.indent(tree, space="  "): chỉ ghi đè Text và Tail khi
// chúng rỗng hoặc toàn khoảng trắng, nên chữ thật của người dùng được giữ.
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
