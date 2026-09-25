package render

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/luytbq/flowcast"
)

// TestRealDrawioDrawsComputedCoordinates exports SVGs with the real drawio for
// the goldens and then runs the render check: a correct diagram must yield 0
// mismatches. This test runs drawio, taking about a second per case, so it only
// runs when FLOWCAST_DRAWIO_TEST=1; tools/check.sh enables it when the machine
// has drawio.
func TestRealDrawioDrawsComputedCoordinates(t *testing.T) {
	if os.Getenv("FLOWCAST_DRAWIO_TEST") != "1" {
		t.Skip("set FLOWCAST_DRAWIO_TEST=1 to run with the real drawio")
	}
	exe := Bin()
	if exe == "" {
		t.Fatal("no drawio CLI")
	}
	for _, c := range []string{"cases/05-merge-node", "cases/16-route-general", "cases/18-labels-crowded",
		"cases/28-tracks-fan-in", "cases/47-place-attach-overflow", "flowchart/05-merge-node",
		"flowchart/11-labels-crowded", "flowchart/15-tracks-fan-in"} {
		src, err := os.ReadFile("../conformance/" + c + ".md")
		if err != nil {
			t.Fatal(err)
		}
		r, err := flowcast.Build(flowcast.Source{Name: filepath.Base(c) + ".md", Data: src}, flowcast.Options{})
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		in, out := filepath.Join(dir, "in.drawio"), filepath.Join(dir, "out.svg")
		if err := os.WriteFile(in, []byte(r.Text), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := Export(context.Background(), exe, in, out, "svg", 0); err != nil {
			t.Fatal(err)
		}
		svg, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		probs, err := Verify(*r.Layout, svg)
		if err != nil {
			t.Fatal(err)
		}
		if len(probs) > 0 {
			t.Errorf("%s: %q", c, probs)
		}
	}
}
