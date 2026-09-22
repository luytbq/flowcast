package render

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/luytbq/flowcast"
)

// TestDrawioThatVeDungToaDo xuất SVG bằng drawio thật cho các golden rồi kiểm
// render: sơ đồ đúng phải cho 0 điểm lệch. Test này chạy drawio, mất khoảng một
// giây mỗi case, nên chỉ chạy khi FLOWCAST_DRAWIO_TEST=1; tools/check.sh bật nó
// khi máy có drawio.
func TestDrawioThatVeDungToaDo(t *testing.T) {
	if os.Getenv("FLOWCAST_DRAWIO_TEST") != "1" {
		t.Skip("đặt FLOWCAST_DRAWIO_TEST=1 để chạy với drawio thật")
	}
	exe := Bin()
	if exe == "" {
		t.Fatal("không có drawio CLI")
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
