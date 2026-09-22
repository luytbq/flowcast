package conformance

import (
	"fmt"
	"os"
	"testing"

	"github.com/luytbq/flowcast/layout"
	"github.com/luytbq/flowcast/num"
	"github.com/luytbq/flowcast/source"
)

func buildLayout(t *testing.T, name string) *layout.Layout {
	t.Helper()
	data, err := os.ReadFile(CaseFile(Dir(), name))
	if err != nil {
		t.Fatal(err)
	}
	tbl, err := source.Parse(source.Source{Data: data, Name: name + ".md"})
	if err != nil {
		t.Fatal(err)
	}
	return layout.New(tbl.Rows, layout.DefaultConfig(), loadMeasure(t))
}

func refString(r layout.Ref, index map[*layout.Seg]int) string {
	switch r.Kind {
	case layout.RefSrc:
		return "src"
	case layout.RefCol:
		return fmt.Sprintf("col:%d:%d", r.Lane, r.Col)
	}
	return fmt.Sprintf("seg:%d", index[r.Seg])
}

func pairString(p [2]float64) []string { return []string{num.Fmt(p[0]), num.Fmt(p[1])} }

// TestChangRoute chốt gói layout ở pha đi dây: kiểu đi dây và cổng của từng
// cạnh, các đoạn dây theo đúng thứ tự được thêm vào, track của từng đoạn, và
// cách pha hình học sẽ dựng toạ độ.
func TestChangRoute(t *testing.T) {
	dumps, err := Load(Dir())
	if err != nil {
		t.Fatal(err)
	}
	edges, segs := 0, 0
	for _, d := range dumps {
		if d.Route == nil {
			continue
		}
		t.Run(d.Name, func(t *testing.T) {
			l := buildLayout(t, d.Name)
			l.Place()
			before := len(l.Warnings)
			l.Route()
			w := d.Route
			assertLines(t, "warnings", l.Warnings[before:], w.Warnings)

			if len(l.Segs) != len(w.Segs) {
				t.Fatalf("Go sinh %d đoạn dây, cần %d", len(l.Segs), len(w.Segs))
			}
			index := map[*layout.Seg]int{}
			for i, s := range l.Segs {
				index[s] = i
				ws := w.Segs[i]
				key := ""
				if ws.Key != nil {
					key = *ws.Key
				}
				var stubs [][]int
				for _, st := range s.Stubs {
					stubs = append(stubs, []int{st.Pos, st.Dir})
				}
				if s.Res.String() != ws.Res || s.Lo != ws.Lo || s.Hi != ws.Hi || s.Key != key ||
					s.Track != ws.Track || fmt.Sprint(stubs) != fmt.Sprint(ws.Stubs) {
					t.Errorf("đoạn %d:\n  Go res=%s lo=%d hi=%d key=%q track=%d stubs=%v\n  Py res=%s lo=%d hi=%d key=%q track=%d stubs=%v",
						i, s.Res, s.Lo, s.Hi, s.Key, s.Track, stubs, ws.Res, ws.Lo, ws.Hi, key, ws.Track, ws.Stubs)
				}
			}
			for _, e := range l.Edges {
				we, ok := w.Edges[e.ID]
				if !ok {
					t.Errorf("%s: dump không có cạnh này", e.ID)
					continue
				}
				got := fmt.Sprintf("case=%c exit=%c entry=%c exit_frac=%v entry_frac=%v back=%v",
					e.Case, e.ExitSide, e.EntrySide, pairString(e.ExitFrac), pairString(e.EntryFrac), e.Back)
				want := fmt.Sprintf("case=%s exit=%s entry=%s exit_frac=%v entry_frac=%v back=%v",
					we.Case, we.Exit, we.Entry, we.ExitFrac, we.EntryFrac, we.Back)
				if got != want {
					t.Errorf("%s:\n  Go %s\n  Py %s", e.ID, got, want)
				}
				var sym [][]string
				for _, p := range e.Sym {
					sym = append(sym, []string{refString(p.X, index), refString(p.Y, index)})
				}
				if fmt.Sprint(sym) != fmt.Sprint(we.Sym) {
					t.Errorf("%s.sym:\n  Go %v\n  Py %v", e.ID, sym, we.Sym)
				}
			}
		})
		edges += len(d.Route.Edges)
		segs += len(d.Route.Segs)
	}
	if edges == 0 {
		t.Fatal("không so được cạnh nào")
	}
	t.Logf("đã so %d cạnh và %d đoạn dây", edges, segs)
}
