package conformance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luytbq/flowcast"
	"github.com/luytbq/flowcast/layout"
	"github.com/luytbq/flowcast/merge"
)

var dirCases = []string{"cases/05-merge-node.md", "cases/28-tracks-fan-in.md", "mermaid/01-dat-hang.mmd"}

// BT is the flip of TD, and RL is the flip of LR: node sizes do not change, so
// the layout is identical, only the orientation differs. LR truly differs from
// TD, because node width and height are swapped before layout.
func TestDirectionFlipMirrorsBaseDirection(t *testing.T) {
	for _, c := range dirCases {
		src := dirSource(t, c)
		for _, pair := range [][2]string{{layout.DirTD, layout.DirBT}, {layout.DirLR, layout.DirRL}} {
			a, err := flowcast.Build(src, flowcast.Options{Direction: pair[0]})
			if err != nil {
				t.Fatal(err)
			}
			b, err := flowcast.Build(src, flowcast.Options{Direction: pair[1]})
			if err != nil {
				t.Fatal(err)
			}
			if a.Stats != b.Stats {
				t.Errorf("%s %s and %s: stats %+v differ from %+v", c, pair[0], pair[1], a.Stats, b.Stats)
			}
			if len(a.Findings) != len(b.Findings) {
				t.Errorf("%s %s and %s: findings %d differ from %d", c, pair[0], pair[1], len(a.Findings), len(b.Findings))
			}
			for i := range a.Layout.Edges {
				if a.Layout.Edges[i].Case != b.Layout.Edges[i].Case {
					t.Errorf("%s: edge %s routed as case %c in %s but %c in %s", c, a.Layout.Edges[i].ID,
						a.Layout.Edges[i].Case, pair[0], b.Layout.Edges[i].Case, pair[1])
				}
			}
		}
	}
}

func dirSource(t *testing.T, c string) flowcast.Source {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(Dir(), c))
	if err != nil {
		t.Fatal(err)
	}
	return flowcast.Source{Name: filepath.Base(c), Data: data}
}

// Every direction must keep the element count, the edge count and the
// self-check finding count, and every forward edge must follow the direction.
func TestDirectionKeepsStructureAndOrientation(t *testing.T) {
	for _, c := range dirCases {
		src := dirSource(t, c)
		td, err := flowcast.Build(src, flowcast.Options{Direction: layout.DirTD})
		if err != nil {
			t.Fatal(err)
		}
		checkFlowDirection(t, c, layout.DirTD, *td.Layout)
		for _, dir := range []string{layout.DirBT, layout.DirLR, layout.DirRL} {
			r, err := flowcast.Build(src, flowcast.Options{Direction: dir})
			if err != nil {
				t.Fatalf("%s %s: %v", c, dir, err)
			}
			if len(r.Findings) != len(td.Findings) || len(r.Layout.Items) != len(td.Layout.Items) ||
				len(r.Layout.Edges) != len(td.Layout.Edges) {
				t.Errorf("%s %s: findings %d, elements %d, edges %d; TD has %d, %d, %d", c, dir,
					len(r.Findings), len(r.Layout.Items), len(r.Layout.Edges),
					len(td.Findings), len(td.Layout.Items), len(td.Layout.Edges))
			}
			checkFlowDirection(t, c, dir, *r.Layout)
		}
	}
}

// checkFlowDirection: every forward edge must follow the direction, measured on
// the centers of the source and target nodes.
func checkFlowDirection(t *testing.T, c, dir string, r layout.Result) {
	t.Helper()
	center := map[string][2]float64{}
	for _, it := range r.Items {
		center[it.ID] = [2]float64{it.X + it.W/2, it.Y + it.H/2}
	}
	for _, e := range r.Edges {
		if e.Back {
			continue
		}
		a, b := center[e.Src], center[e.Dst]
		var ok bool
		switch dir {
		case layout.DirTD:
			ok = b[1] >= a[1]
		case layout.DirBT:
			ok = b[1] <= a[1]
		case layout.DirLR:
			ok = b[0] >= a[0]
		case layout.DirRL:
			ok = b[0] <= a[0]
		}
		if !ok {
			t.Errorf("%s %s: edge %s runs against the direction: %v to %v", c, dir, e.ID, a, b)
		}
	}
}

func TestDirectionLRDrawsLanesAsHorizontalBands(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(Dir(), "cases", "05-merge-node.md"))
	if err != nil {
		t.Fatal(err)
	}
	src := flowcast.Source{Name: "a.md", Data: data}
	for dir, want := range map[string]bool{layout.DirTD: false, layout.DirBT: false, layout.DirLR: true, layout.DirRL: true} {
		r, err := flowcast.Build(src, flowcast.Options{Direction: dir})
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Contains(r.Text, "horizontal=0"); got != want {
			t.Errorf("%s: horizontal lanes %v, want %v", dir, got, want)
		}
	}
}

// The direction declared by the source, such as mermaid's flowchart LR line,
// must be used when the caller does not force another direction.
func TestDirectionTakenFromSource(t *testing.T) {
	src := dirSource(t, "mermaid/01-dat-hang.mmd")
	lr := flowcast.Source{Name: "lr.mmd", Data: []byte(strings.Replace(string(src.Data), "flowchart TD", "flowchart LR", 1))}
	td, err := flowcast.Build(src, flowcast.Options{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := flowcast.Build(lr, flowcast.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Stats.W <= got.Stats.H || got.Stats == td.Stats {
		t.Errorf("source declares LR but is still drawn as TD: %v then %v", td.Stats, got.Stats)
	}
	// When the caller forces a direction, the source direction gives way.
	forced, err := flowcast.Build(lr, flowcast.Options{Direction: layout.DirTD})
	if err != nil {
		t.Fatal(err)
	}
	if forced.Stats != td.Stats {
		t.Errorf("forced TD: %v, want %v", forced.Stats, td.Stats)
	}
}

func TestDirectionInvalidIsRejected(t *testing.T) {
	src := flowcast.Source{Name: "a.md", Data: []byte("| id | type | parent | content | metadata |\n|-|-|-|-|-|\n")}
	if _, err := flowcast.Build(src, flowcast.Options{Direction: "XY"}); err == nil {
		t.Error("direction XY must be rejected")
	}
}

func TestMergeRejectsDirectionOtherThanTD(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(Dir(), "cases", "05-merge-node.md"))
	if err != nil {
		t.Fatal(err)
	}
	src := flowcast.Source{Name: "a.md", Data: data}
	fresh, err := flowcast.Build(src, flowcast.Options{Direction: layout.DirLR})
	if err != nil {
		t.Fatal(err)
	}
	old, err := merge.Read([]byte(fresh.Text), "old.drawio")
	if err != nil {
		t.Fatal(err)
	}
	_, err = flowcast.Build(src, flowcast.Options{Direction: layout.DirLR, Previous: old})
	if err == nil || !strings.Contains(err.Error(), "merge does not support") {
		t.Errorf("error %v", err)
	}
}

// Goldens for the three new directions: regression guard, regenerate with
// -update after reviewing the images.
func TestDirectionMatchesGolden(t *testing.T) {
	for _, c := range dirCases {
		data, err := os.ReadFile(filepath.Join(Dir(), c))
		if err != nil {
			t.Fatal(err)
		}
		for _, dir := range []string{layout.DirBT, layout.DirLR, layout.DirRL} {
			r, err := flowcast.Build(flowcast.Source{Name: filepath.Base(c), Data: data},
				flowcast.Options{Direction: dir})
			if err != nil {
				t.Fatal(err)
			}
			name := strings.TrimSuffix(filepath.Base(c), filepath.Ext(c)) + "-" + dir + ".drawio"
			golden := filepath.Join(Dir(), "golden", "direction", name)
			if *update {
				if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(golden, []byte(r.Text), 0o644); err != nil {
					t.Fatal(err)
				}
				continue
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("%v; run go test ./conformance -run Direction -update, then review the images", err)
			}
			if r.Text != string(want) {
				t.Errorf("%s: differs from golden", name)
			}
		}
	}
}
