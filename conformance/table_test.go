package conformance

import (
	"os"
	"testing"

	"github.com/luytbq/flowcast/source"
)

// TestChangTable chốt gói model và source/markdown: bảng sau khi parse phải
// khớp bản tham chiếu trên toàn bộ bộ đối chiếu, kể cả những bảng có lỗi.
func TestChangTable(t *testing.T) {
	dumps, err := Load(Dir())
	if err != nil {
		t.Fatal(err)
	}
	rows := 0
	for _, d := range dumps {
		t.Run(d.Name, func(t *testing.T) {
			data, err := os.ReadFile(CaseFile(Dir(), d.Name))
			if err != nil {
				t.Fatal(err)
			}
			got, err := source.Parse(source.Source{Data: data, Name: d.Name + ".md"})
			if err != nil {
				t.Fatalf("parse lỗi: %v", err)
			}
			if got.Title != d.Table.Title {
				t.Errorf("tiêu đề = %q, cần %q", got.Title, d.Table.Title)
			}
			if got.Source != d.Table.Source {
				t.Errorf("nguồn = %q, cần %q", got.Source, d.Table.Source)
			}
			if len(got.Rows) != len(d.Table.Rows) {
				t.Fatalf("đọc được %d dòng, cần %d", len(got.Rows), len(d.Table.Rows))
			}
			for i, w := range d.Table.Rows {
				g := got.Rows[i]
				if g.Idx != w.Idx || g.Loc.String() != w.Loc || g.ID != w.ID ||
					g.Type != w.Type || g.Parent != w.Parent {
					t.Errorf("dòng %d: Go {idx:%d loc:%q id:%q type:%q parent:%q}, cần {idx:%d loc:%q id:%q type:%q parent:%q}",
						i, g.Idx, g.Loc.String(), g.ID, g.Type, g.Parent,
						w.Idx, w.Loc, w.ID, w.Type, w.Parent)
					continue
				}
				assertLines(t, w.ID+".lines", g.Lines, w.Lines)
				assertMeta(t, w.ID, g.Meta, w.Meta)
				assertLines(t, w.ID+".meta_order", g.MetaKeys, w.MetaOrder)
			}
			assertIssues(t, got.Issues, d.Table.Issues)
		})
		rows += len(d.Table.Rows)
	}
	if rows == 0 {
		t.Fatal("không so được dòng nào")
	}
	t.Logf("đã so %d dòng trên %d case", rows, len(dumps))
}

func assertMeta(t *testing.T, id string, got, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s.meta = %v, cần %v", id, got, want)
		return
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s.meta[%q] = %q, cần %q", id, k, got[k], v)
		}
	}
}
