package drawio

import "strings"

// node là một phần tử XML, đủ cho những gì file draw.io cần: thẻ, thuộc tính
// theo thứ tự chèn, và phần tử con. Không có nội dung chữ.
//
// Viết riêng thay vì dùng encoding/xml vì đầu ra phải khớp từng byte với
// ElementTree của Python: thuộc tính giữ đúng thứ tự chèn, phần tử rỗng in
// thành "<tag ... />" với một khoảng trắng trước dấu gạch chéo, xuống dòng
// trong giá trị thuộc tính thành tham chiếu số, và thụt lề hai khoảng trắng.
type node struct {
	tag      string
	attrs    [][2]string
	children []*node
}

func el(tag string, attrs ...string) *node {
	n := &node{tag: tag}
	for i := 0; i+1 < len(attrs); i += 2 {
		n.attrs = append(n.attrs, [2]string{attrs[i], attrs[i+1]})
	}
	return n
}

func (n *node) add(tag string, attrs ...string) *node {
	c := el(tag, attrs...)
	n.children = append(n.children, c)
	return c
}

func (n *node) set(k, v string) {
	for i := range n.attrs {
		if n.attrs[i][0] == k {
			n.attrs[i][1] = v
			return
		}
	}
	n.attrs = append(n.attrs, [2]string{k, v})
}

// escapeAttr thoát giá trị thuộc tính theo đúng thứ tự của ElementTree.
func escapeAttr(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;",
		"\r", "&#13;", "\n", "&#10;", "\t", "&#09;")
	return r.Replace(s)
}

// render in cây theo cách ET.indent(space="  ") rồi ET.tostring(encoding="unicode")
// của Python cho ra: không có khai báo XML và không có dòng trống ở cuối.
func (n *node) render() string {
	var b strings.Builder
	n.write(&b, 0)
	return b.String()
}

func (n *node) write(b *strings.Builder, level int) {
	b.WriteString("<" + n.tag)
	for _, a := range n.attrs {
		b.WriteString(" " + a[0] + "=\"" + escapeAttr(a[1]) + "\"")
	}
	if len(n.children) == 0 {
		b.WriteString(" />")
		return
	}
	b.WriteString(">")
	inner := "\n" + strings.Repeat("  ", level+1)
	for _, c := range n.children {
		b.WriteString(inner)
		c.write(b, level+1)
	}
	b.WriteString("\n" + strings.Repeat("  ", level) + "</" + n.tag + ">")
}
