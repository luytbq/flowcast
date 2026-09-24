package conformance

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/luytbq/flowcast"
	"github.com/luytbq/flowcast/merge"
	"github.com/luytbq/flowcast/model"
)

var update = flag.Bool("update", false, "ghi lại golden từ kết quả hiện tại")

// sets là các bộ case. Mỗi case có golden riêng trong golden/<bộ>/: file .drawio
// khi bảng dựng được, và file .report.txt ghi mọi phát hiện. Golden là bộ chống
// hồi quy: đổi golden thì phải xem ảnh và đọc diff.
var sets = []string{"cases/*.md", "cases/*.csv", "cases/*.xlsx", "flowchart/*.md", "mermaid/*.mmd"}

func casePaths(t *testing.T) []string {
	t.Helper()
	var paths []string
	for _, pat := range sets {
		ps, err := filepath.Glob(filepath.Join(Dir(), pat))
		if err != nil || len(ps) == 0 {
			t.Fatalf("không có case %s: %v", pat, err)
		}
		paths = append(paths, ps...)
	}
	return paths
}

// goldenPath đặt golden của mỗi bộ vào thư mục con cùng tên với bộ đó.
func goldenPath(p, ext string) string {
	set := filepath.Base(filepath.Dir(p))
	name := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	return filepath.Join(Dir(), "golden", set, name+ext)
}

func build(p string, opt flowcast.Options) (flowcast.Result, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return flowcast.Result{}, err
	}
	return flowcast.Build(flowcast.Source{Name: filepath.Base(p), Data: data}, opt)
}

// buildValid dựng một case mà test cần bảng hợp lệ; case sai cố ý trả ok false.
func buildValid(t *testing.T, p string, opt flowcast.Options) (flowcast.Result, bool) {
	t.Helper()
	r, err := build(p, opt)
	if err != nil || r.HasErrors() {
		return r, false
	}
	return r, true
}

// report ghi mọi phát hiện của một lần dựng thành văn bản, mỗi phát hiện một
// dòng: issue của bảng, cảnh báo của engine, rồi kết quả tự kiểm.
func report(r flowcast.Result, err error) string {
	var b strings.Builder
	if err != nil {
		code := ""
		if me, ok := err.(*model.Error); ok {
			code = me.Code
		}
		fmt.Fprintf(&b, "lỗi %s: %s\n", code, err)
		return b.String()
	}
	for _, i := range r.Issues {
		fmt.Fprintf(&b, "issue %s %s\n", i.Code, i)
	}
	for _, w := range r.Warnings {
		fmt.Fprintf(&b, "engine %s: %s\n", w.Code, w.Msg)
	}
	for _, f := range r.Findings {
		fmt.Fprintf(&b, "tự kiểm %s %s: %s\n", f.Level, f.Code, f.Msg)
	}
	if r.Text != "" {
		s := r.Stats
		fmt.Fprintf(&b, "sơ đồ: %d lane, %d phần tử, %d cạnh\n", s.Lanes, s.Items, s.Edges)
	}
	return b.String()
}

func checkGolden(t *testing.T, path, got string) {
	t.Helper()
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v; chạy go test ./conformance -update rồi xem ảnh và diff", err)
	}
	if got != string(want) {
		t.Errorf("%s: lệch golden ở byte %d: %s", path, firstDiff(got, string(want)), context(got, string(want)))
	}
}

func TestCaseKhopGolden(t *testing.T) {
	for _, p := range casePaths(t) {
		r, err := build(p, flowcast.Options{})
		checkGolden(t, goldenPath(p, ".report.txt"), report(r, err))
		drawio := goldenPath(p, ".drawio")
		if err == nil && r.Text != "" {
			checkGolden(t, drawio, r.Text)
		} else if _, statErr := os.Stat(drawio); statErr == nil && !*update {
			t.Errorf("%s: bảng không dựng được nhưng golden vẫn có .drawio", filepath.Base(p))
		}
	}
}

// TestCaseHopLeKhongCoLoiTuKiem: engine đúng không bao giờ sinh ra hình học mà
// tự kiểm bác.
func TestCaseHopLeKhongCoLoiTuKiem(t *testing.T) {
	for _, p := range casePaths(t) {
		r, ok := buildValid(t, p, flowcast.Options{})
		if !ok {
			continue
		}
		for _, f := range r.Findings {
			if f.Level == "error" {
				t.Errorf("%s: tự kiểm báo lỗi: %s", filepath.Base(p), f.Msg)
			}
		}
		if r.Layout.NoLanes && strings.Contains(r.Text, `id="pool"`) {
			t.Errorf("%s: sơ đồ không có lane mà vẫn vẽ pool", filepath.Base(p))
		}
	}
}

func TestMergeKhongDoiFileChuaSua(t *testing.T) {
	for _, p := range casePaths(t) {
		fresh, ok := buildValid(t, p, flowcast.Options{})
		if !ok {
			continue
		}
		old, err := merge.Read([]byte(fresh.Text), "cũ.drawio")
		if err != nil {
			t.Fatal(err)
		}
		merged, _ := buildValid(t, p, flowcast.Options{Previous: old})
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

func TestMergeGiuNodeDaKeo(t *testing.T) {
	p := filepath.Join(Dir(), "flowchart", "05-merge-node.md")
	fresh, _ := buildValid(t, p, flowcast.Options{})
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
	merged, _ := buildValid(t, p, flowcast.Options{Previous: old})
	got := geo.FindStringSubmatch(merged.Text)
	if got == nil || got[2] != "777" || got[4] != "555" {
		t.Errorf("A-3 không giữ vị trí đã kéo: %v", got)
	}
}

// TestConditionVaoTuDinhRaBaMat: dây vào một condition luôn nối vào đỉnh, dây
// ra chỉ đi ở mặt trái, phải hoặc đáy.
func TestConditionVaoTuDinhRaBaMat(t *testing.T) {
	checked := 0
	for _, p := range casePaths(t) {
		r, ok := buildValid(t, p, flowcast.Options{})
		if !ok {
			continue
		}
		kind := map[string]string{}
		for _, it := range r.Layout.Items {
			kind[it.ID] = it.Kind
		}
		for _, e := range r.Layout.Edges {
			if kind[e.Dst] == "condition" && e.EntryFrac != [2]float64{0.5, 0} {
				t.Errorf("%s: %s vào condition %s tại %v, không phải đỉnh", filepath.Base(p), e.ID, e.Dst, e.EntryFrac)
			}
			if kind[e.Src] == "condition" && e.ExitSide != 'L' && e.ExitSide != 'R' && e.ExitSide != 'B' {
				t.Errorf("%s: %s ra khỏi condition %s ở mặt %c", filepath.Base(p), e.ID, e.Src, e.ExitSide)
			}
			if kind[e.Dst] == "condition" || kind[e.Src] == "condition" {
				checked++
			}
		}
	}
	if checked == 0 {
		t.Fatal("không có dây nào nối condition để kiểm")
	}
}

func firstDiff(a, b string) int {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

func context(got, want string) string {
	i := firstDiff(got, want)
	lo := max(0, i-60)
	cut := func(s string) string { return s[lo:min(len(s), i+60)] }
	return "\n  có     " + cut(got) + "\n  golden " + cut(want)
}
