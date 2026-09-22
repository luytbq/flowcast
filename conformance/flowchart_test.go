package conformance

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/luytbq/flowcast"
	"github.com/luytbq/flowcast/merge"
)

var update = flag.Bool("update", false, "ghi lại golden của flowchart từ bản Go")

// Flowchart và mermaid là hành vi bản tham chiếu không có, nên golden của chúng do bản Go
// sinh và là bộ chống hồi quy chứ không phải đáp án: đổi golden thì phải xem ảnh
// và đọc diff.
func flowcharts(t *testing.T) []string {
	t.Helper()
	var paths []string
	for _, pat := range []string{"flowchart/*.md", "mermaid/*.mmd"} {
		ps, err := filepath.Glob(filepath.Join(Dir(), pat))
		if err != nil || len(ps) == 0 {
			t.Fatalf("không có case %s: %v", pat, err)
		}
		paths = append(paths, ps...)
	}
	return paths
}

// goldenPath đặt golden của mỗi bộ vào thư mục con cùng tên với bộ đó.
func goldenPath(p string) string {
	set := filepath.Base(filepath.Dir(p))
	name := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	return filepath.Join(Dir(), "golden", set, name+".drawio")
}

func buildFile(t *testing.T, p string, opt flowcast.Options) flowcast.Result {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	r, err := flowcast.Build(flowcast.Source{Name: filepath.Base(p), Data: data}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if r.HasErrors() {
		t.Fatalf("%s: bảng có lỗi: %v", p, r.Issues)
	}
	return r
}

func TestFlowchartKhopGolden(t *testing.T) {
	for _, p := range flowcharts(t) {
		name := filepath.Base(p)
		r := buildFile(t, p, flowcast.Options{})
		for _, f := range r.Findings {
			if f.Level == "error" {
				t.Errorf("%s: tự kiểm báo lỗi: %s", name, f.Msg)
			}
		}
		if r.Layout.NoLanes && strings.Contains(r.Text, `id="pool"`) {
			t.Errorf("%s: sơ đồ không có lane mà vẫn vẽ pool", name)
		}
		golden := goldenPath(p)
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
			t.Fatalf("%v; chạy go test ./conformance -run Flowchart -update rồi xem ảnh", err)
		}
		if r.Text != string(want) {
			t.Errorf("%s: lệch golden", name)
		}
	}
}

func TestFlowchartMergeKhongDoiFileChuaSua(t *testing.T) {
	for _, p := range flowcharts(t) {
		fresh := buildFile(t, p, flowcast.Options{})
		old, err := merge.Read([]byte(fresh.Text), "cũ.drawio")
		if err != nil {
			t.Fatal(err)
		}
		merged := buildFile(t, p, flowcast.Options{Previous: old})
		if merged.Text != fresh.Text {
			t.Errorf("%s: merge trên file chưa sửa làm đổi file", filepath.Base(p))
		}
		for _, l := range merged.Merge.Lines() {
			if strings.Contains(l, "element mới") && !strings.HasSuffix(l, "(0): -") {
				t.Errorf("%s: %s", filepath.Base(p), l)
			}
		}
	}
}

func TestFlowchartMergeGiuNodeDaKeo(t *testing.T) {
	p := filepath.Join(Dir(), "flowchart", "05-merge-node.md")
	fresh := buildFile(t, p, flowcast.Options{})
	geo := regexp.MustCompile(`(<mxCell id="A-3"[^>]*>\s*<mxGeometry x=")([^"]+)(" y=")([^"]+)`)
	m := geo.FindStringSubmatch(fresh.Text)
	if m == nil {
		t.Fatal("không thấy geometry của A-3")
	}
	edited := geo.ReplaceAllString(fresh.Text, "${1}777${3}555")
	old, err := merge.Read([]byte(edited), "cũ.drawio")
	if err != nil {
		t.Fatal(err)
	}
	merged := buildFile(t, p, flowcast.Options{Previous: old})
	got := geo.FindStringSubmatch(merged.Text)
	if got == nil || got[2] != "777" || got[4] != "555" {
		t.Errorf("A-3 không giữ vị trí đã kéo: %v", got)
	}
}
