package layout_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/luytbq/flowcast"
	"github.com/luytbq/flowcast/layout"
)

// TestOptimizedCasesAreAFixedPoint: running the optimizer again on its own
// output changes nothing, for every valid case of the case suite.
func TestOptimizedCasesAreAFixedPoint(t *testing.T) {
	cfg := layout.DefaultConfig()
	cfg.Optimize = 10
	var paths []string
	for _, pat := range []string{"cases/*.md", "flowchart/*.md", "mermaid/*.mmd"} {
		ps, _ := filepath.Glob(filepath.Join("..", "conformance", pat))
		paths = append(paths, ps...)
	}
	if len(paths) == 0 {
		t.Fatal("no cases found")
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		r, err := flowcast.Build(flowcast.Source{Name: filepath.Base(p), Data: data}, flowcast.Options{Config: &cfg, Direction: layout.DirTD})
		if err != nil || r.HasErrors() {
			continue
		}
		// The layout of a top-down diagram is the optimizer's own space.
		if again := layout.Optimize(*r.Layout, cfg, nil); !reflect.DeepEqual(again, *r.Layout) {
			t.Errorf("%s: a second optimize still changed the layout", filepath.Base(p))
		}
	}
}
