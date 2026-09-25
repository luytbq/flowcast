package layout

import (
	"reflect"
	"testing"
)

// jogResult is one lane with a source S at the top right, a blocker B, and a
// target T below it, joined by a wire that detours through a gutter track when
// it could go straight down under S.
func jogResult() Result {
	return Result{
		PoolW: 600, PoolH: 400, PoolHeader: 30, LaneHeader: 30,
		LaneX: []float64{0}, LaneW: []float64{600},
		Items: []PlacedItem{
			{ID: "S", X: 400, Y: 80, W: 100, H: 40},
			{ID: "B", X: 100, Y: 160, W: 100, H: 40},
			{ID: "T", X: 100, Y: 300, W: 100, H: 40},
		},
		Edges: []PlacedEdge{{ID: "E", Src: "S", Dst: "T",
			Pts: [][2]float64{{450, 120}, {450, 140}, {300, 140}, {300, 260}, {150, 260}, {150, 300}}}},
	}
}

func TestOptimizeStraightensGutterDetour(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Optimize = 5
	got := Optimize(jogResult(), cfg, nil).Edges[0].Pts
	want := [][2]float64{{450, 120}, {450, 260}, {150, 260}, {150, 300}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("path after optimize %v, want %v", got, want)
	}
}

func TestOptimizeOffChangesNothing(t *testing.T) {
	r := jogResult()
	if got := Optimize(r, DefaultConfig(), nil); !reflect.DeepEqual(got, r) {
		t.Errorf("optimize = 0 still changed the result")
	}
}

func TestOptimizeNeverRoutesThroughAnItem(t *testing.T) {
	r := jogResult()
	// An item right under the source blocks the straight path down.
	r.Items = append(r.Items, PlacedItem{ID: "X", X: 420, Y: 200, W: 60, H: 30})
	cfg := DefaultConfig()
	cfg.Optimize = 5
	got := Optimize(r, cfg, nil).Edges[0].Pts
	if !reflect.DeepEqual(got, r.Edges[0].Pts) {
		t.Errorf("wire pulled through an item: %v", got)
	}
}

func TestOptimizeReanchorsLabelInPlace(t *testing.T) {
	r := jogResult()
	lb := [4]float64{160, 262, 200, 280}
	r.Edges[0].Label = &lb
	r.Edges[0].LabelOff = [2]float64{30, 11}
	cfg := DefaultConfig()
	cfg.Optimize = 5
	e := Optimize(r, cfg, nil).Edges[0]
	if *e.Label != lb {
		t.Fatalf("label box moved: %v", *e.Label)
	}
	// The label centre (180, 271) projects onto the horizontal run y = 260 at x = 180.
	if e.LabelOff != [2]float64{0, 11} {
		t.Errorf("label offset %v, want [0 11]", e.LabelOff)
	}
}

// TestOptimizeRejectsStraighteningThatAddsACrossing: removing two bends is worth
// less than one new crossing, so the detour stays.
func TestOptimizeRejectsStraighteningThatAddsACrossing(t *testing.T) {
	r := jogResult()
	r.Edges = append(r.Edges, PlacedEdge{ID: "F", Src: "P", Dst: "Q",
		Pts: [][2]float64{{420, 200}, {520, 200}}})
	cfg := DefaultConfig()
	cfg.Optimize = 5
	got := Optimize(r, cfg, nil).Edges[0].Pts
	if !reflect.DeepEqual(got, r.Edges[0].Pts) {
		t.Errorf("straightened across another wire: %v", got)
	}
}
