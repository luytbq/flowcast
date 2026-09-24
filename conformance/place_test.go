package conformance

import (
	"strconv"
	"testing"

	"github.com/luytbq/flowcast/layout"
)

// TestChangPlace chốt gói layout ở pha xếp chỗ: thứ tự topo, lane, hàng, cột
// của từng phần tử, và cảnh báo phát sinh trong pha này.
func TestChangPlace(t *testing.T) {
	dumps, err := Load(Dir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := layout.DefaultConfig()
	checked := 0
	for _, d := range dumps {
		if d.Place == nil {
			continue // bảng có lỗi, không có layout nào được dựng
		}
		t.Run(d.Name, func(t *testing.T) {
			tbl, err := ParseCase(Dir(), d.Name)
			if err != nil {
				t.Fatal(err)
			}
			l := layout.New(tbl.Rows, cfg, loadMeasure(t))
			l.Place()
			w := d.Place

			var lanes []string
			for _, r := range l.Lanes {
				lanes = append(lanes, r.ID)
			}
			assertLines(t, "lanes", lanes, w.Lanes)
			assertLines(t, "topo", l.TopoOrder, w.Topo)
			assertLines(t, "warnings", warningMsgs(l.Warnings), w.Warnings)
			if l.NRows != w.NRows {
				t.Errorf("nrows = %d, cần %d", l.NRows, w.NRows)
			}
			for lane, want := range w.Cols {
				i, _ := strconv.Atoi(lane)
				if got := l.Cols[i]; !equalInts(got, want) {
					t.Errorf("cols[lane %s] = %v, cần %v", lane, got, want)
				}
			}
			for id, want := range w.Items {
				it := l.Item(id)
				if it == nil {
					t.Errorf("%s: Go không có phần tử này", id)
					continue
				}
				if it.Lane != want.Lane || it.Row != want.Row || it.Col != want.Col || it.Attach != want.Attach {
					t.Errorf("%s: Go lane=%d row=%d col=%d attach=%q, cần lane=%d row=%d col=%d attach=%q",
						id, it.Lane, it.Row, it.Col, it.Attach, want.Lane, want.Row, want.Col, want.Attach)
				}
			}
			if len(l.ItemOrder) != len(w.Items) {
				t.Errorf("Go có %d phần tử, cần %d", len(l.ItemOrder), len(w.Items))
			}
		})
		checked += len(d.Place.Items)
	}
	if checked == 0 {
		t.Fatal("không so được phần tử nào")
	}
	t.Logf("đã so vị trí %d phần tử", checked)
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func warningMsgs(ws []layout.Warning) []string {
	out := make([]string, len(ws))
	for i, w := range ws {
		out[i] = w.Msg
	}
	return out
}
