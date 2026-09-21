package conformance

import (
	"path/filepath"
	"testing"

	"github.com/luytbq/flowcast/layout"
	"github.com/luytbq/flowcast/num"
	"github.com/luytbq/flowcast/text"
)

var nodeTypes = map[string]bool{"start": true, "end": true, "task": true, "condition": true, "external": true}
var attachTypes = map[string]bool{"db": true, "text": true}

const restMarker = "Phần còn lại"

// isItem nói một dòng bảng có trở thành phần tử của sơ đồ hay không.
//
// Luật này thuộc về model chứ không thuộc text; nó nằm tạm ở đây cho tới khi
// bước 3 của lộ trình port dựng gói model.
func isItem(r Row) bool {
	if nodeTypes[r.Type] {
		return true
	}
	if !attachTypes[r.Type] {
		return false
	}
	isMarker := r.Type == "text" && trimJoin(r.Lines) == restMarker && r.Meta["attach"] == ""
	return !isMarker
}

func trimJoin(lines []string) string {
	s := ""
	for i, l := range lines {
		if i > 0 {
			s += "\n"
		}
		s += l
	}
	return trimSpace(s)
}

func loadMeasure(t *testing.T) *text.Measure {
	t.Helper()
	m, err := text.LoadMetrics(filepath.Join("..", "data", "verdana.json"))
	if err != nil {
		t.Fatalf("không đọc được bảng số đo: %v", err)
	}
	return text.NewMeasure(m)
}

// TestChangText chốt gói text và hàm SizeItem: ngắt dòng và kích thước hộp phải
// khớp bản tham chiếu trên toàn bộ bộ đối chiếu.
func TestChangText(t *testing.T) {
	dumps, err := Load(Dir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := layout.DefaultConfig()
	checked := 0
	for _, d := range dumps {
		if d.Text == nil {
			continue // bảng có lỗi, không có layout nào được dựng
		}
		t.Run(d.Name, func(t *testing.T) {
			tm := loadMeasure(t)
			for _, r := range d.Table.Rows {
				switch {
				case r.Type == "edge":
					want, ok := d.Text.Edges[r.ID]
					if !ok {
						t.Fatalf("%s: dump không có cạnh này", r.ID)
					}
					lines, lw, lh := layout.SizeLabel(tm, cfg, r.Lines)
					assertLines(t, r.ID, lines, want.Lines)
					assertNum(t, r.ID+".lw", lw, want.LW)
					assertNum(t, r.ID+".lh", lh, want.LH)
				case isItem(r):
					want, ok := d.Text.Items[r.ID]
					if !ok {
						t.Fatalf("%s: dump không có phần tử này", r.ID)
					}
					lines, w, h := layout.SizeItem(tm, cfg, r.Type, r.Lines)
					assertLines(t, r.ID, lines, want.Lines)
					assertNum(t, r.ID+".w", w, want.W)
					assertNum(t, r.ID+".h", h, want.H)
				}
			}
		})
		checked += len(d.Text.Items) + len(d.Text.Edges)
	}
	if checked == 0 {
		t.Fatal("không so được phần tử nào; bộ đối chiếu rỗng?")
	}
	t.Logf("đã so %d phần tử và cạnh trên %d case", checked, len(dumps))
}

func assertLines(t *testing.T, id string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: ngắt thành %d dòng %q, cần %d dòng %q", id, len(got), got, len(want), want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s: dòng %d = %q, cần %q", id, i, got[i], want[i])
		}
	}
}

func assertNum(t *testing.T, what string, got float64, want string) {
	t.Helper()
	if s := num.Fmt(got); s != want {
		t.Errorf("%s = %s, cần %s", what, s, want)
	}
}
