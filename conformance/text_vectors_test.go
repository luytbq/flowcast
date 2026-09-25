package conformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luytbq/flowcast/layout"
	"github.com/luytbq/flowcast/num"
	"github.com/luytbq/flowcast/text"
)

type widthVector struct {
	MaxW  int      `json:"maxw"`
	Lines []string `json:"lines"`
	Hard  bool     `json:"hard"`
	W     string   `json:"w"`
	H     string   `json:"h"`
}

type sampleVector struct {
	Text   string        `json:"text"`
	Widths []widthVector `json:"widths"`
}

type sizeRow struct {
	Text  string   `json:"text"`
	Lines []string `json:"lines"`
	W     string   `json:"w"`
	H     string   `json:"h"`
}

type sizeKind struct {
	Kind string    `json:"kind"`
	Rows []sizeRow `json:"rows"`
}

type vectorFile struct {
	Size    float64        `json:"size"`
	LineH   float64        `json:"line_h"`
	Samples []sampleVector `json:"samples"`
	Sizes   []sizeKind     `json:"sizes"`
	Budgets map[string]int `json:"budgets"`
}

// TestWrapVectors sweeps the width one pixel at a time.
//
// A gate based on the diagram case suite is too loose for this module: Wrap
// shrinks to the smallest width that keeps the same line count, so a budget off
// by a few pixels usually does not change the output. Sweeping pixel by pixel
// makes any deviation, even of one pixel, flip at least one row in the table.
func TestWrapVectors(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(Dir(), "text-vectors.json"))
	if err != nil {
		t.Fatalf("cannot read vectors: %v", err)
	}
	var vf vectorFile
	if err := json.Unmarshal(data, &vf); err != nil {
		t.Fatal(err)
	}
	m, err := text.LoadMetrics(filepath.Join("..", "data", "verdana.json"))
	if err != nil {
		t.Fatal(err)
	}
	tm := text.NewMeasure(m)
	if tm.Size != vf.Size || tm.LineH != vf.LineH {
		t.Fatalf("font size or line height mismatch: Go %v/%v, vector %v/%v",
			tm.Size, tm.LineH, vf.Size, vf.LineH)
	}

	checked, failed := 0, 0
	for _, s := range vf.Samples {
		lines := strings.Split(s.Text, "\n")
		for _, v := range s.Widths {
			checked++
			got := tm.Wrap(lines, float64(v.MaxW))
			gw, gh := tm.Box(got)
			bad := tm.Hard() != v.Hard || num.Fmt(gw) != v.W || num.Fmt(gh) != v.H || len(got) != len(v.Lines)
			if !bad {
				for i := range got {
					if got[i] != v.Lines[i] {
						bad = true
						break
					}
				}
			}
			if bad {
				failed++
				if failed <= 5 {
					t.Errorf("%q at maxw=%d:\n  Go   lines=%q hard=%v w=%s h=%s\n  Py   lines=%q hard=%v w=%s h=%s",
						s.Text, v.MaxW, got, tm.Hard(), num.Fmt(gw), num.Fmt(gh),
						v.Lines, v.Hard, v.W, v.H)
				}
			}
		}
	}
	if failed > 5 {
		t.Errorf("... and %d more mismatched vectors", failed-5)
	}
	t.Logf("compared %d vectors", checked)
}

// TestWrapBudgets pins the numbers directly, not through behavior.
//
// A constant off by 2px hardly shows in the output: a character boundary has to
// fall exactly inside that gap to change the line count, and with characters
// about 7px wide a 2px window is usually empty. Verified by mutation testing.
func TestWrapBudgets(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(Dir(), "text-vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var vf vectorFile
	if err := json.Unmarshal(data, &vf); err != nil {
		t.Fatal(err)
	}
	if len(vf.Budgets) == 0 {
		t.Fatal("vectors have no budget entries")
	}
	cfg := layout.DefaultConfig()
	for kind, want := range vf.Budgets {
		if kind == "__label__" {
			if got := cfg.LabelWrap; got != want {
				t.Errorf("edge label budget = %d, want %d", got, want)
			}
			continue
		}
		if got := layout.WrapBudget(cfg, kind); got != float64(want) {
			t.Errorf("budget of %q = %v, want %d", kind, got, want)
		}
	}
	t.Logf("compared %d budgets", len(vf.Budgets))
}

// TestSizeVectors pins the wrap budget of each element kind.
//
// The budget is a constant in SizeItem, and it is nearly invisible to the
// diagram case suite: Wrap shrinks to the smallest width that keeps the same
// line count, so a few pixels off rarely changes the output. The ladder of
// increasing lengths below crosses every line-count boundary, so the flip point
// moves as soon as the budget is off by even 2px.
func TestSizeVectors(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(Dir(), "text-vectors.json"))
	if err != nil {
		t.Fatalf("cannot read vectors: %v", err)
	}
	var vf vectorFile
	if err := json.Unmarshal(data, &vf); err != nil {
		t.Fatal(err)
	}
	m, err := text.LoadMetrics(filepath.Join("..", "data", "verdana.json"))
	if err != nil {
		t.Fatal(err)
	}
	tm := text.NewMeasure(m)
	cfg := layout.DefaultConfig()

	checked, failed := 0, 0
	for _, k := range vf.Sizes {
		for _, r := range k.Rows {
			checked++
			lines, w, h := layout.SizeItem(tm, cfg, k.Kind, []string{r.Text})
			bad := num.Fmt(w) != r.W || num.Fmt(h) != r.H || len(lines) != len(r.Lines)
			if !bad {
				for i := range lines {
					if lines[i] != r.Lines[i] {
						bad = true
						break
					}
				}
			}
			if bad {
				failed++
				if failed <= 5 {
					t.Errorf("%s %q:\n  Go lines=%q w=%s h=%s\n  Py lines=%q w=%s h=%s",
						k.Kind, r.Text, lines, num.Fmt(w), num.Fmt(h), r.Lines, r.W, r.H)
				}
			}
		}
	}
	if failed > 5 {
		t.Errorf("... and %d more mismatched vectors", failed-5)
	}
	if checked == 0 {
		t.Fatal("no size vectors")
	}
	t.Logf("compared %d size vectors", checked)
}
