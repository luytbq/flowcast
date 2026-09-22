package conformance

import (
	"math"
	"strconv"
	"testing"
)

// bitsEqual so một số Go với giá trị float.hex() của Python, từng bit.
func bitsEqual(t *testing.T, got float64, hex string) bool {
	t.Helper()
	want, err := strconv.ParseFloat(hex, 64)
	if err != nil {
		t.Fatalf("giá trị %q trong dump không đọc được: %v", hex, err)
	}
	return math.Float64bits(got) == math.Float64bits(want)
}

// TestChangGeometry chốt pha hình học và nhãn tới từng bit.
//
// So từng bit chứ không so hai chữ số thập phân, vì pha này ra quyết định bằng
// so sánh số thực. Lệch một đơn vị ở bit cuối có thể chưa đổi gì trên bộ case
// này mà vẫn lật một quyết định trên bảng khác.
func TestChangGeometry(t *testing.T) {
	dumps, err := Load(Dir())
	if err != nil {
		t.Fatal(err)
	}
	values := 0
	for _, d := range dumps {
		if d.Geom == nil {
			continue
		}
		t.Run(d.Name, func(t *testing.T) {
			pr := buildLayout(t, d.Name)
			pr.Place()
			pr.Route()
			l := buildLayout(t, d.Name).Run()
			assertLines(t, "warnings", l.Warnings[len(pr.Warnings):], d.Geom.Warnings)
			w := d.Geom.Exact
			check := func(what string, got float64, hex string) {
				values++
				if !bitsEqual(t, got, hex) {
					want, _ := strconv.ParseFloat(hex, 64)
					t.Errorf("%s = %v (%x), Python %v (%x)", what, got, math.Float64bits(got), want, math.Float64bits(want))
				}
			}
			check("pool.w", l.PoolW, w.Pool[0])
			check("pool.h", l.PoolH, w.Pool[1])
			if len(l.LaneX) != len(w.Lanes) {
				t.Fatalf("Go có %d lane, cần %d", len(l.LaneX), len(w.Lanes))
			}
			for i, lw := range w.Lanes {
				check("lane.x", l.LaneX[i], lw[0])
				check("lane.w", l.LaneW[i], lw[1])
			}
			for id, v := range w.Items {
				it := l.Item(id)
				check(id+".x", it.X, v[0])
				check(id+".y", it.Y, v[1])
				check(id+".w", it.W, v[2])
				check(id+".h", it.H, v[3])
			}
			for _, e := range l.Edges {
				we, ok := w.Edges[e.ID]
				if !ok {
					t.Errorf("%s: dump không có cạnh này", e.ID)
					continue
				}
				if len(e.Pts) != len(we.Points) {
					t.Errorf("%s: %d điểm, cần %d: Go %v", e.ID, len(e.Pts), len(we.Points), e.Pts)
					continue
				}
				for i, p := range we.Points {
					check(e.ID+".pts.x", e.Pts[i][0], p[0])
					check(e.ID+".pts.y", e.Pts[i][1], p[1])
				}
				if (e.Label == nil) != (we.Label == nil) {
					t.Errorf("%s: có nhãn Go=%v, Python=%v", e.ID, e.Label != nil, we.Label != nil)
					continue
				}
				if e.Label != nil {
					for i, v := range we.Label {
						check(e.ID+".label", e.Label[i], v)
					}
				}
				check(e.ID+".label_t", e.LabelT, we.LabelT)
				check(e.ID+".label_off.x", e.LabelOff[0], we.LabelOff[0])
				check(e.ID+".label_off.y", e.LabelOff[1], we.LabelOff[1])
			}
		})
	}
	if values == 0 {
		t.Fatal("không so được giá trị nào")
	}
	t.Logf("đã so %d số thực từng bit", values)
}
