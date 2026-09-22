package flowcast

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/luytbq/flowcast/model"
)

func caseSource(t *testing.T, name string) Source {
	t.Helper()
	data, err := os.ReadFile("conformance/cases/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return Source{Name: name, Data: data}
}

func limitCode(t *testing.T, err error) string {
	t.Helper()
	var e *model.Error
	if !errors.As(err, &e) {
		t.Fatalf("mong lỗi model.Error, nhận %v", err)
	}
	return e.Code
}

func TestVuotGioiHanTraVeLoiMaLimit(t *testing.T) {
	md := caseSource(t, "05-merge-node.md")
	xlsx := caseSource(t, "xlsx-01-shared-strings.xlsx")
	cases := []struct {
		name string
		src  Source
		lim  Limits
		code string
	}{
		{"bytes", md, Limits{MaxBytes: 100}, "limit.bytes"},
		{"rows", md, Limits{MaxRows: 5}, "limit.rows"},
		{"edges", md, Limits{MaxEdges: 2}, "limit.edges"},
		{"unzipped", xlsx, Limits{MaxUnzipped: 200}, "limit.unzipped"},
		{"timeout", md, Limits{Timeout: time.Nanosecond}, "limit.timeout"},
	}
	for _, c := range cases {
		for _, fn := range []func(Source, Options) (Result, error){Build, Check} {
			_, err := fn(c.src, Options{Limits: &c.lim})
			if got := limitCode(t, err); got != c.code {
				t.Errorf("%s: mã %s, mong %s", c.name, got, c.code)
			}
		}
	}
}

func TestDungDuoiGioiHanThiDungNhuKhongChan(t *testing.T) {
	for _, name := range []string{"05-merge-node.md", "xlsx-01-shared-strings.xlsx"} {
		src := caseSource(t, name)
		free, err := Build(src, Options{})
		if err != nil {
			t.Fatal(err)
		}
		lim := WebLimits
		limited, err := Build(src, Options{Limits: &lim})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if free.Text != limited.Text || free.Text == "" {
			t.Errorf("%s: giới hạn làm đổi đầu ra", name)
		}
	}
}

func TestGioiHanDungDungBienCuaSoDong(t *testing.T) {
	src := caseSource(t, "05-merge-node.md")
	// Mỗi dòng dữ liệu bắt đầu bằng "| ", trừ dòng header.
	n := strings.Count(string(src.Data), "\n| ") - 1
	if _, err := Build(src, Options{Limits: &Limits{MaxRows: n}}); err != nil {
		t.Errorf("đúng %d dòng mà vẫn bị chặn: %v", n, err)
	}
	if _, err := Build(src, Options{Limits: &Limits{MaxRows: n - 1}}); err == nil {
		t.Errorf("%d dòng với giới hạn %d mà không bị chặn", n, n-1)
	}
}
