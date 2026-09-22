// Package drawio sinh file .drawio từ một sơ đồ đã xếp.
package drawio

import (
	"strings"

	"github.com/luytbq/flowcast/layout"
	"github.com/luytbq/flowcast/num"
)

const (
	highlightNode = "fillColor=#dae8fc;strokeColor=#6c8ebf;"
	highlightText = "fillColor=#dae8fc;strokeColor=none;"
	highlightEdge = "strokeColor=#6c8ebf;strokeWidth=2;"
	font          = "fontFamily=Verdana;fontSize=12;"
	// mark đánh dấu cell do tool sinh. Khi sinh lại, merge dùng nó để tách cell
	// người dùng tự vẽ ra khỏi cell của tool.
	mark = "flowtable=1;"
)

// shapeStyle không bật whiteSpace=wrap: chữ đã được ngắt dòng sẵn bằng <br>, và
// để draw.io tự ngắt lại thì số dòng có thể khác kích thước node đã tính.
var shapeStyle = map[string]string{
	"task":      "rounded=0;html=1;",
	"condition": "rhombus;html=1;",
	"start":     "ellipse;html=1;",
	"end":       "shape=doubleEllipse;html=1;",
	"external":  "ellipse;html=1;dashed=1;",
	"db":        "shape=cylinder3;html=1;boundedLbl=1;backgroundOutline=1;size=8;",
	"text":      "text;html=1;align=left;verticalAlign=middle;spacingLeft=4;",
}

// htmlLines nối các dòng bằng <br> sau khi thoát ký tự html. Phép thoát này
// chồng lên phép thoát thuộc tính XML lúc ghi file, nên & trong nội dung thành
// &amp;amp; trong file: draw.io giải thuộc tính trước, rồi giải html.
func htmlLines(lines []string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = r.Replace(l)
	}
	return strings.Join(out, "<br>")
}

func f(v float64) string { return num.Fmt(v) }

func geo(cell *node, x, y, w, h float64) {
	cell.add("mxGeometry", "x", f(x), "y", f(y), "width", f(w), "height", f(h), "as", "geometry")
}

// Write sinh nội dung một file .drawio.
func Write(r layout.Result, title string) string {
	ox, oy := r.Origin[0], r.Origin[1]
	if title == "" {
		title = "Flow"
	}
	name := []rune(title)
	if len(name) > 80 {
		name = name[:80]
	}
	mxfile := el("mxfile", "host", "flowtable2drawio")
	diagram := mxfile.add("diagram", "id", "flowtable", "name", string(name))
	model := diagram.add("mxGraphModel", "grid", "1", "gridSize", "10", "guides", "1", "tooltips", "1",
		"connect", "1", "arrows", "1", "fold", "1", "page", "1", "pageScale", "1",
		"pageWidth", f(r.PoolW+float64(2*ox)), "pageHeight", f(r.PoolH+float64(2*oy)),
		"math", "0", "shadow", "0")
	root := model.add("root")
	root.add("mxCell", "id", "0")
	root.add("mxCell", "id", "1", "parent", "0")

	pool := root.add("mxCell", "id", "pool", "value", htmlLines([]string{title}), "vertex", "1", "parent", "1",
		"style", "swimlane;html=1;childLayout=stackLayout;horizontalStack=1;resizeParent=1;"+
			"resizeParentMax=0;startSize="+itoa(r.PoolHeader)+";collapsible=0;fontStyle=1;"+font+mark)
	geo(pool, ox, oy, r.PoolW, r.PoolH)
	for i, lane := range r.Lanes {
		c := root.add("mxCell", "id", lane.ID, "value", htmlLines(lane.Lines), "vertex", "1", "parent", "pool",
			"style", "swimlane;html=1;startSize="+itoa(r.LaneHeader)+";collapsible=0;"+font+mark)
		geo(c, r.LaneX[i], float64(r.PoolHeader), r.LaneW[i], r.PoolH-float64(r.PoolHeader))
	}
	for _, it := range r.Items {
		style := shapeStyle[it.Kind]
		if it.Highlight {
			if it.Kind == "text" {
				style += highlightText
			} else {
				style += highlightNode
			}
		}
		c := root.add("mxCell", "id", it.ID, "value", htmlLines(it.Lines), "vertex", "1",
			"parent", r.Lanes[it.Lane].ID, "style", style+font+mark)
		geo(c, it.X-r.LaneX[it.Lane], it.Y-float64(r.PoolHeader), it.W, it.H)
	}
	for _, e := range r.Edges {
		style := "edgeStyle=orthogonalEdgeStyle;rounded=0;orthogonalLoop=1;jettySize=auto;html=1;" +
			"labelBackgroundColor=default;" +
			"exitX=" + f(e.ExitFrac[0]) + ";exitY=" + f(e.ExitFrac[1]) + ";exitDx=0;exitDy=0;" +
			"entryX=" + f(e.EntryFrac[0]) + ";entryY=" + f(e.EntryFrac[1]) + ";entryDx=0;entryDy=0;"
		if e.Dashed {
			style += "dashed=1;"
		}
		if e.Highlight {
			style += highlightEdge
		}
		c := root.add("mxCell", "id", e.ID, "value", htmlLines(e.Lines), "edge", "1", "parent", "pool",
			"source", e.Src, "target", e.Dst, "style", style+font+mark)
		g := c.add("mxGeometry", "relative", "1", "as", "geometry")
		if e.Label != nil {
			g.set("x", f(e.LabelT))
		}
		if len(e.Pts) > 2 {
			arr := g.add("Array", "as", "points")
			for _, p := range e.Pts[1 : len(e.Pts)-1] {
				arr.add("mxPoint", "x", f(p[0]), "y", f(p[1]))
			}
		}
		if e.Label != nil {
			g.add("mxPoint", "x", f(e.LabelOff[0]), "y", f(e.LabelOff[1]), "as", "offset")
		}
	}
	return mxfile.render()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	if neg {
		d = append([]byte{'-'}, d...)
	}
	return string(d)
}
