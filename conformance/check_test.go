package conformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/luytbq/flowcast/layout"
)

type checkVector struct {
	From     string `json:"from"`
	Geometry struct {
		Lanes [][2]string `json:"lanes"`
		Items []struct {
			ID   string    `json:"id"`
			Lane int       `json:"lane"`
			Box  [4]string `json:"box"`
		} `json:"items"`
		Edges []struct {
			ID    string      `json:"id"`
			Src   string      `json:"src"`
			Dst   string      `json:"dst"`
			Pts   [][2]string `json:"pts"`
			Label []string    `json:"label"`
		} `json:"edges"`
	} `json:"geometry"`
	Findings [][2]string `json:"findings"`
}

func hexf(t *testing.T, s string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatalf("%q: %v", s, err)
	}
	return v
}

func assertFindings(t *testing.T, got []layout.Finding, want [][2]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("có %d phát hiện, cần %d:\n  Go %v\n  Py %v", len(got), len(want), got, want)
		return
	}
	for i := range got {
		if got[i].Level != want[i][0] || got[i].Msg != want[i][1] {
			t.Errorf("phát hiện %d: Go %s %q, Py %s %q", i, got[i].Level, got[i].Msg, want[i][0], want[i][1])
		}
	}
}

// TestTuKiemTrenHinhHocHong chạy tự kiểm trên hình học do các bản tham chiếu đã
// bị đột biến sinh ra.
//
// Engine đúng không sinh ra lỗi hình học, nên đầu ra của nó không kiểm được tự
// kiểm. Các hình học này thì hỏng đủ kiểu, và phủ cả tám loại phát hiện.
func TestTuKiemTrenHinhHocHong(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(Dir(), "check-vectors.json"))
	if err != nil {
		t.Fatalf("%v; chạy tools/check_vectors.py", err)
	}
	var vs []checkVector
	if err := json.Unmarshal(data, &vs); err != nil {
		t.Fatal(err)
	}
	findings := 0
	for i, v := range vs {
		var r layout.Result
		for _, l := range v.Geometry.Lanes {
			r.LaneX = append(r.LaneX, hexf(t, l[0]))
			r.LaneW = append(r.LaneW, hexf(t, l[1]))
		}
		for _, it := range v.Geometry.Items {
			r.Items = append(r.Items, layout.PlacedItem{ID: it.ID, Lane: it.Lane,
				X: hexf(t, it.Box[0]), Y: hexf(t, it.Box[1]), W: hexf(t, it.Box[2]), H: hexf(t, it.Box[3])})
		}
		for _, e := range v.Geometry.Edges {
			pe := layout.PlacedEdge{ID: e.ID, Src: e.Src, Dst: e.Dst}
			for _, p := range e.Pts {
				pe.Pts = append(pe.Pts, [2]float64{hexf(t, p[0]), hexf(t, p[1])})
			}
			if e.Label != nil {
				pe.Label = &[4]float64{hexf(t, e.Label[0]), hexf(t, e.Label[1]), hexf(t, e.Label[2]), hexf(t, e.Label[3])}
			}
			r.Edges = append(r.Edges, pe)
		}
		t.Run(strconv.Itoa(i)+" "+v.From, func(t *testing.T) {
			assertFindings(t, layout.Check(r), v.Findings)
		})
		findings += len(v.Findings)
	}
	t.Logf("đã so %d phát hiện trên %d hình học hỏng", findings, len(vs))
}

// TestChangCheck chạy tự kiểm trên đầu ra của chính engine Go, qua cả bộ case.
