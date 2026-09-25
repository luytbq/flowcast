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
		t.Fatalf("want a model.Error, got %v", err)
	}
	return e.Code
}

func TestExceedingLimitReturnsLimitCode(t *testing.T) {
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
				t.Errorf("%s: code %s, want %s", c.name, got, c.code)
			}
		}
	}
}

func TestWithinLimitsBuildsSameAsUncapped(t *testing.T) {
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
			t.Errorf("%s: limits changed the output", name)
		}
	}
}

func TestRowLimitStopsExactlyAtBoundary(t *testing.T) {
	src := caseSource(t, "05-merge-node.md")
	// Every data row starts with "| ", except the header row.
	n := strings.Count(string(src.Data), "\n| ") - 1
	if _, err := Build(src, Options{Limits: &Limits{MaxRows: n}}); err != nil {
		t.Errorf("exactly %d rows yet still blocked: %v", n, err)
	}
	if _, err := Build(src, Options{Limits: &Limits{MaxRows: n - 1}}); err == nil {
		t.Errorf("%d rows with limit %d yet not blocked", n, n-1)
	}
}
