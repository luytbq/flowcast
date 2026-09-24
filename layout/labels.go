package layout

import (
	"math"
	"strings"

	"github.com/luytbq/flowcast/num"
)

type box = [4]float64 // x1, y1, x2, y2

func boxOf(it *Item) box { return box{it.X, it.Y, it.X + it.W, it.Y + it.H} }

// area là diện tích phần chồng nhau của hai hộp.
func area(a, b box) float64 {
	w := pyMin(a[2], b[2]) - pyMax(a[0], b[0])
	h := pyMin(a[3], b[3]) - pyMax(a[1], b[1])
	if w > 0 && h > 0 {
		return float64(w * h)
	}
	return 0
}

// segHits nói đoạn thẳng từ a tới b có cắt vào trong hộp hay không. Chỉ chạm
// mép thì không tính.
func segHits(a, b [2]float64, bx box) bool {
	x1, x2 := pyMin(a[0], b[0]), pyMax(a[0], b[0])
	y1, y2 := pyMin(a[1], b[1]), pyMax(a[1], b[1])
	return x1 < bx[2] && x2 > bx[0] && y1 < bx[3] && y2 > bx[1]
}

func pathLen(pts [][2]float64) float64 {
	total := 0.0
	for i := 1; i < len(pts); i++ {
		total += math.Abs(pts[i][0]-pts[i-1][0]) + math.Abs(pts[i][1]-pts[i-1][1])
	}
	return total
}

type candidate struct {
	b      box
	dist   float64    // khoảng cách dọc đường từ nguồn tới điểm neo
	anchor [2]float64 // điểm trên đường mà nhãn bám vào
}

// labelCandidates liệt kê các chỗ có thể đặt nhãn, theo thứ tự ưu tiên: gần
// nguồn trước, và trên mỗi đoạn đủ dài thì thử sát đầu đoạn rồi giữa đoạn, mỗi
// chỗ thử cả hai bên đường.
func labelCandidates(e *Edge) []candidate {
	w, h := e.LW, e.LH
	acc := 0.0
	var out []candidate
	for i := 1; i < len(e.Pts); i++ {
		a, b := e.Pts[i-1], e.Pts[i]
		L := math.Abs(b[0]-a[0]) + math.Abs(b[1]-a[1])
		if L >= 12 {
			if math.Abs(a[1]-b[1]) < 0.01 {
				d := -1.0
				if b[0] > a[0] {
					d = 1
				}
				for _, cx := range []float64{a[0] + float64(d*(6+w/2)), (a[0] + b[0]) / 2} {
					ax := pyMin(pyMax(cx, pyMin(a[0], b[0])), pyMax(a[0], b[0]))
					for _, cy := range []float64{a[1] - 3 - h/2, a[1] + 3 + h/2} {
						out = append(out, candidate{
							box{cx - w/2, cy - h/2, cx + w/2, cy + h/2},
							acc + math.Abs(ax-a[0]), [2]float64{ax, a[1]},
						})
					}
				}
			} else {
				d := -1.0
				if b[1] > a[1] {
					d = 1
				}
				for _, cy := range []float64{a[1] + float64(d*(6+h/2)), (a[1] + b[1]) / 2} {
					ay := pyMin(pyMax(cy, pyMin(a[1], b[1])), pyMax(a[1], b[1]))
					for _, cx := range []float64{a[0] + 5 + w/2, a[0] - 5 - w/2} {
						out = append(out, candidate{
							box{cx - w/2, cy - h/2, cx + w/2, cy + h/2},
							acc + math.Abs(ay-a[1]), [2]float64{a[0], ay},
						})
					}
				}
			}
		}
		acc += L
	}
	return out
}

// placeLabels đặt nhãn cho từng cạnh vào ứng viên có chi phí nhỏ nhất.
//
// Chi phí cộng diện tích chồng lên node, lên nhãn đã đặt và lên header, cộng
// một khoản cố định cho mỗi dây khác cắt qua. Ứng viên đầu tiên có chi phí
// bằng không được nhận ngay. Thứ tự cộng giữ đúng bản tham chiếu, vì các tổng
// này được so sánh với nhau và cộng số thực không có tính kết hợp.
func (l *Layout) placeLabels() {
	type named struct {
		id string
		b  box
	}
	var boxes []named
	for _, it := range l.ItemOrder {
		boxes = append(boxes, named{it.ID, boxOf(it)})
	}
	boxes = append(boxes, named{"header", box{-1e6, -1e6, 1e6, float64(l.Cfg.PoolHeader + l.Cfg.LaneHeader)}})

	type wire struct {
		id   string // rỗng với đường phân cách lane
		a, b [2]float64
	}
	var wires []wire
	for _, x := range l.LaneX[1:] {
		wires = append(wires, wire{"", [2]float64{x, 0}, [2]float64{x, l.PoolH}})
	}
	for _, e := range l.Edges {
		for i := 1; i < len(e.Pts); i++ {
			wires = append(wires, wire{e.ID, e.Pts[i-1], e.Pts[i]})
		}
	}

	var placed []box
	for _, e := range l.Edges {
		if len(e.Lines) == 0 {
			continue
		}
		var best *candidate
		bestCost := 0.0
		for _, c := range labelCandidates(e) {
			cost := 0.0
			for _, nb := range boxes {
				cost += float64(area(c.b, nb.b) * 4)
			}
			for _, pb := range placed {
				cost += float64(area(c.b, pb) * 4)
			}
			for _, w := range wires {
				if w.id != e.ID && segHits(w.a, w.b, c.b) {
					cost += 100
				}
			}
			if best == nil || cost < bestCost {
				cc := c
				best, bestCost = &cc, cost
			}
			if cost == 0 {
				break
			}
		}
		if best == nil {
			l.warn("layout.label-no-room", e.ID, "%s: đường quá ngắn, không có chỗ đặt nhãn", e.ID)
			continue
		}
		if bestCost > 0 {
			l.warn("layout.label-crowded", e.ID, "%s: nhãn \"%s\" không tìm được chỗ trống hoàn toàn",
				e.ID, strings.Join(e.Lines, " "))
		}
		placed = append(placed, best.b)
		b := best.b
		e.Label = &b
		if total := pathLen(e.Pts); total != 0 {
			e.LabelT = num.Round(float64(2*best.dist)/total-1, 4)
		} else {
			e.LabelT = 0
		}
		cx, cy := (b[0]+b[2])/2, (b[1]+b[3])/2
		e.LabelOff = [2]float64{num.Round(cx-best.anchor[0], 2), num.Round(cy-best.anchor[1], 2)}
	}
}

// Run chạy đủ các pha theo đúng thứ tự.
//
// Hình học tính hai lượt. Lượt đầu chưa biết phần ngang của các cạnh B và C
// dài bao nhiêu, tức chưa biết nhãn của chúng có vừa không; lượt hai chừa thêm
// chỗ cho những nhãn không vừa rồi tính lại.
func (l *Layout) Run() *Layout {
	l.Place()
	l.Route()
	lzG, lzC := l.labelReservations(nil)
	l.computeGeometry(lzG, lzC)
	l.resolvePaths()
	lzG, lzC = l.labelReservations(l.horizontalSpans())
	l.computeGeometry(lzG, lzC)
	l.resolvePaths()
	l.placeLabels()
	return l
}
