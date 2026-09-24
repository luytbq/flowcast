package layout

import (
	"fmt"
	"math"
	"sort"
)

// Finding là một phát hiện của tự kiểm hình học. Code là mã máy ổn định để
// caller phân loại; Msg là thông điệp cho người đọc.
type Finding struct {
	Level string // "error" hoặc "warning"
	Code  string
	Msg   string
}

func shrink(b box, d float64) box { return box{b[0] + d, b[1] + d, b[2] - d, b[3] - d} }

// collinearOverlap là bề dài phần hai đoạn nằm chồng lên nhau trên cùng một
// đường ngang hoặc dọc. Hai đoạn cách nhau dưới nửa điểm ảnh coi như cùng
// đường, vì khi vẽ ra mắt không phân biệt được.
func collinearOverlap(a1, b1, a2, b2 [2]float64) float64 {
	if math.Abs(a1[1]-b1[1]) < 0.01 && math.Abs(a2[1]-b2[1]) < 0.01 && math.Abs(a1[1]-a2[1]) < 0.5 {
		lo := pyMax(pyMin(a1[0], b1[0]), pyMin(a2[0], b2[0]))
		hi := pyMin(pyMax(a1[0], b1[0]), pyMax(a2[0], b2[0]))
		return hi - lo
	}
	if math.Abs(a1[0]-b1[0]) < 0.01 && math.Abs(a2[0]-b2[0]) < 0.01 && math.Abs(a1[0]-a2[0]) < 0.5 {
		lo := pyMax(pyMin(a1[1], b1[1]), pyMin(a2[1], b2[1]))
		hi := pyMin(pyMax(a1[1], b1[1]), pyMax(a2[1], b2[1]))
		return hi - lo
	}
	return 0
}

// Check tìm lỗi hình học trong một sơ đồ đã xếp: node chồng node, node tràn ra
// ngoài lane, đoạn dây xiên, dây cắt qua node, hai dây khác nguồn khác đích
// chồng lên nhau, và nhãn đè lên node, nhãn khác hay dây khác.
//
// Engine đúng không bao giờ sinh ra các lỗi này; Check là lưới an toàn cho
// chính engine. Kết quả đã bỏ trùng và giữ thứ tự phát hiện đầu tiên.
func Check(r Result) []Finding {
	var issues []Finding
	add := func(level, code, format string, a ...any) {
		issues = append(issues, Finding{level, code, fmt.Sprintf(format, a...)})
	}

	boxes := map[string]box{}
	for _, it := range r.Items {
		boxes[it.ID] = it.box()
	}
	ids := make([]string, 0, len(r.Items))
	for _, it := range r.Items {
		ids = append(ids, it.ID)
	}
	sort.Strings(ids)
	for i, a := range ids {
		for _, b := range ids[i+1:] {
			if area(boxes[a], boxes[b]) > 0 {
				add("error", "check.node-overlap", "%s và %s chồng lên nhau", a, b)
			}
		}
	}
	for _, it := range r.Items {
		lx, lw := r.LaneX[it.Lane], r.LaneW[it.Lane]
		if it.X < lx || it.X+it.W > lx+lw {
			add("error", "check.outside-lane", "%s tràn ra ngoài lane", it.ID)
		}
	}

	type wire struct {
		edge int
		a, b [2]float64
	}
	var all []wire
	for ei, e := range r.Edges {
		pts := e.Pts
		for k := 0; k+1 < len(pts); k++ {
			a, b := pts[k], pts[k+1]
			if math.Abs(a[0]-b[0]) > 0.01 && math.Abs(a[1]-b[1]) > 0.01 {
				add("error", "check.oblique-segment", "%s: đoạn %d không vuông góc", e.ID, k)
			}
			for _, it := range r.Items {
				// Đoạn đầu và đoạn cuối được chạm vào chính node nguồn và node
				// đích, vì chúng xuất phát và kết thúc ở mép node đó.
				if (it.ID == e.Src || it.ID == e.Dst) && (k == 0 || k == len(pts)-2) {
					continue
				}
				if segHits(a, b, shrink(boxes[it.ID], 1)) {
					add("error", "check.edge-crosses-node", "%s: dây cắt qua %s", e.ID, it.ID)
				}
			}
			all = append(all, wire{ei, a, b})
		}
	}
	for i, w1 := range all {
		for _, w2 := range all[i+1:] {
			e1, e2 := r.Edges[w1.edge], r.Edges[w2.edge]
			// Dây cùng nguồn hoặc cùng đích được phép chồng: chúng gộp thành
			// một đường chung.
			if w1.edge == w2.edge || e1.Dst == e2.Dst || e1.Src == e2.Src {
				continue
			}
			if collinearOverlap(w1.a, w1.b, w2.a, w2.b) > 1 {
				add("error", "check.edge-overlap", "%s và %s chồng dây", e1.ID, e2.ID)
			}
		}
	}

	type label struct {
		edge int
		b    box
	}
	var labels []label
	for ei, e := range r.Edges {
		if e.Label != nil {
			labels = append(labels, label{ei, *e.Label})
		}
	}
	for i, lb := range labels {
		id := r.Edges[lb.edge].ID
		for _, it := range r.Items {
			if area(lb.b, boxes[it.ID]) > 0 {
				add("warning", "check.label-over-node", "nhãn %s đè lên %s", id, it.ID)
			}
		}
		for _, lb2 := range labels[i+1:] {
			if area(lb.b, lb2.b) > 0 {
				add("warning", "check.label-over-label", "nhãn %s đè nhãn %s", id, r.Edges[lb2.edge].ID)
			}
		}
		for _, w := range all {
			if w.edge != lb.edge && segHits(w.a, w.b, lb.b) {
				add("warning", "check.label-over-edge", "nhãn %s đè dây %s", id, r.Edges[w.edge].ID)
			}
		}
	}

	seen := map[Finding]bool{}
	var out []Finding
	for _, f := range issues {
		if !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	return out
}
