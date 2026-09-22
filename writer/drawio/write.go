// Package drawio sinh file .drawio từ một sơ đồ đã xếp.
package drawio

import (
	"strings"

	"github.com/luytbq/flowcast/internal/etree"
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

func geo(cell *etree.Element, x, y, w, h float64) {
	cell.Add("mxGeometry", "x", f(x), "y", f(y), "width", f(w), "height", f(h), "as", "geometry")
}

// Write sinh nội dung một file .drawio.
func Write(r layout.Result, title string) string { return WriteMerged(r, title, nil, nil) }

// WriteMerged sinh file .drawio kèm những gì merge giữ lại từ file cũ: extras
// là các cell người dùng tự vẽ, nối vào cuối trang đầu; pages là các trang còn
// lại, nối nguyên vẹn sau trang đầu.
func WriteMerged(r layout.Result, title string, extras, pages []*etree.Element) string {
	ox, oy := r.Origin[0], r.Origin[1]
	if title == "" {
		title = "Flow"
	}
	name := []rune(title)
	if len(name) > 80 {
		name = name[:80]
	}
	mxfile := etree.New("mxfile", "host", "flowtable2drawio")
	diagram := mxfile.Add("diagram", "id", "flowtable", "name", string(name))
	model := diagram.Add("mxGraphModel", "grid", "1", "gridSize", "10", "guides", "1", "tooltips", "1",
		"connect", "1", "arrows", "1", "fold", "1", "page", "1", "pageScale", "1",
		"pageWidth", f(r.PoolW+float64(2*ox)), "pageHeight", f(r.PoolH+float64(2*oy)),
		"math", "0", "shadow", "0")
	root := model.Add("root")
	root.Add("mxCell", "id", "0")
	root.Add("mxCell", "id", "1", "parent", "0")

	// Toạ độ trong Result tính theo pool. Sơ đồ không có lane thì không có pool
	// làm cha: phần tử và dây nằm thẳng trên layer, nên cộng gốc pool vào.
	parentOf := func(it layout.PlacedItem) string { return r.Lanes[it.Lane].ID }
	edgeParent, dx, dy := "pool", 0.0, 0.0
	if r.NoLanes {
		parentOf = func(layout.PlacedItem) string { return "1" }
		edgeParent, dx, dy = "1", ox, oy
	}
	if !r.NoLanes {
		writePool(root, r, title, ox, oy)
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
		c := root.Add("mxCell", "id", it.ID, "value", htmlLines(it.Lines), "vertex", "1",
			"parent", parentOf(it), "style", style+font+mark)
		if r.NoLanes {
			geo(c, it.X+ox, it.Y+oy, it.W, it.H)
		} else {
			geo(c, it.X-r.LaneX[it.Lane], it.Y-float64(r.PoolHeader), it.W, it.H)
		}
	}
	for _, e := range r.Edges {
		style := "edgeStyle=orthogonalEdgeStyle;rounded=0;orthogonalLoop=1;jettySize=auto;html=1;" +
			"labelBackgroundColor=default;"
		switch {
		case e.Auto:
		case e.Kept:
			for _, kv := range e.Constraints {
				style += kv[0] + "=" + kv[1] + ";"
			}
		default:
			style += "exitX=" + f(e.ExitFrac[0]) + ";exitY=" + f(e.ExitFrac[1]) + ";exitDx=0;exitDy=0;" +
				"entryX=" + f(e.EntryFrac[0]) + ";entryY=" + f(e.EntryFrac[1]) + ";entryDx=0;entryDy=0;"
		}
		if e.Dashed {
			style += "dashed=1;"
		}
		if e.Highlight {
			style += highlightEdge
		}
		c := root.Add("mxCell", "id", e.ID, "value", htmlLines(e.Lines), "edge", "1", "parent", edgeParent,
			"source", e.Src, "target", e.Dst, "style", style+font+mark)
		g := c.Add("mxGeometry", "relative", "1", "as", "geometry")
		withLabel := e.Label != nil && !e.NoLabelPos && !e.Auto
		if withLabel {
			g.Set("x", f(e.LabelT))
		}
		var pts [][2]float64
		switch {
		case e.Auto:
		case e.Kept:
			pts = e.Waypoints
		case len(e.Pts) > 2:
			pts = e.Pts[1 : len(e.Pts)-1]
		}
		if len(pts) > 0 {
			arr := g.Add("Array", "as", "points")
			for _, p := range pts {
				arr.Add("mxPoint", "x", f(p[0]+dx), "y", f(p[1]+dy))
			}
		}
		if withLabel {
			g.Add("mxPoint", "x", f(e.LabelOff[0]), "y", f(e.LabelOff[1]), "as", "offset")
		}
	}
	root.Children = append(root.Children, extras...)
	mxfile.Children = append(mxfile.Children, pages...)
	etree.Indent(mxfile)
	return mxfile.String()
}

func writePool(root *etree.Element, r layout.Result, title string, ox, oy float64) {
	pool := root.Add("mxCell", "id", "pool", "value", htmlLines([]string{title}), "vertex", "1", "parent", "1",
		"style", "swimlane;html=1;childLayout=stackLayout;horizontalStack=1;resizeParent=1;"+
			"resizeParentMax=0;startSize="+itoa(r.PoolHeader)+";collapsible=0;fontStyle=1;"+font+mark)
	geo(pool, ox, oy, r.PoolW, r.PoolH)
	for i, lane := range r.Lanes {
		c := root.Add("mxCell", "id", lane.ID, "value", htmlLines(lane.Lines), "vertex", "1", "parent", "pool",
			"style", "swimlane;html=1;startSize="+itoa(r.LaneHeader)+";collapsible=0;"+font+mark)
		geo(c, r.LaneX[i], float64(r.PoolHeader), r.LaneW[i], r.PoolH-float64(r.PoolHeader))
	}
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
