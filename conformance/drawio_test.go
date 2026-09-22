package conformance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/luytbq/flowcast/layout"
	"github.com/luytbq/flowcast/source"
	"github.com/luytbq/flowcast/validate"
	"github.com/luytbq/flowcast/writer/drawio"
)

// TestFileDrawioKhopTungByte là cổng cuối của cả chuỗi: từ bảng đầu vào tới file
// .drawio, bản Go phải ra đúng từng byte như bản tham chiếu.
func TestFileDrawioKhopTungByte(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join(Dir(), "golden", "*.drawio"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("không có golden .drawio: %v", err)
	}
	for _, p := range paths {
		name := filepath.Base(p)
		name = name[:len(name)-len(".drawio")]
		t.Run(name, func(t *testing.T) {
			want, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(CaseFile(Dir(), name))
			if err != nil {
				t.Fatal(err)
			}
			tbl, err := source.Parse(source.Source{Data: data, Name: name + ".md"})
			if err != nil {
				t.Fatal(err)
			}
			for _, i := range append(tbl.Issues, validate.Validate(tbl.Rows)...) {
				if i.Level == "error" {
					t.Fatalf("bảng có lỗi nhưng golden vẫn có .drawio: %s", i)
				}
			}
			l := layout.New(tbl.Rows, layout.DefaultConfig(), loadMeasure(t)).Run()
			got := drawio.Write(l.Result(), tbl.Title)
			if got != string(want) {
				t.Errorf("lệch ở byte %d: %s", firstDiff(got, string(want)), context(got, string(want)))
			}
		})
	}
	t.Logf("đã so %d file .drawio", len(paths))
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
	return "\n  Go " + cut(got) + "\n  Py " + cut(want)
}
