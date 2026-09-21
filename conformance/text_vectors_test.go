package conformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luytbq/flowcast/num"
	"github.com/luytbq/flowcast/text"
)

type widthVector struct {
	MaxW  int      `json:"maxw"`
	Lines []string `json:"lines"`
	Hard  bool     `json:"hard"`
	W     string   `json:"w"`
	H     string   `json:"h"`
}

type sampleVector struct {
	Text   string        `json:"text"`
	Widths []widthVector `json:"widths"`
}

type vectorFile struct {
	Size    float64        `json:"size"`
	LineH   float64        `json:"line_h"`
	Samples []sampleVector `json:"samples"`
}

// TestVectorNgatDong quét bề rộng từng pixel một.
//
// Cổng dựa vào bộ case sơ đồ quá lỏng cho module này: Wrap thu hẹp về bề rộng
// nhỏ nhất vẫn giữ nguyên số dòng, nên lệch vài pixel ở ngân sách thường không
// đổi đầu ra. Quét từng pixel thì mọi sai lệch dù một pixel đều lật ít nhất một
// dòng trong bảng.
func TestVectorNgatDong(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(Dir(), "text-vectors.json"))
	if err != nil {
		t.Fatalf("không đọc được vector: %v; chạy tools/text_vectors.py", err)
	}
	var vf vectorFile
	if err := json.Unmarshal(data, &vf); err != nil {
		t.Fatal(err)
	}
	m, err := text.LoadMetrics(filepath.Join("..", "data", "verdana.json"))
	if err != nil {
		t.Fatal(err)
	}
	tm := text.NewMeasure(m)
	if tm.Size != vf.Size || tm.LineH != vf.LineH {
		t.Fatalf("cỡ chữ hoặc bề cao dòng lệch: Go %v/%v, vector %v/%v",
			tm.Size, tm.LineH, vf.Size, vf.LineH)
	}

	checked, failed := 0, 0
	for _, s := range vf.Samples {
		lines := strings.Split(s.Text, "\n")
		for _, v := range s.Widths {
			checked++
			got := tm.Wrap(lines, float64(v.MaxW))
			gw, gh := tm.Box(got)
			bad := tm.Hard() != v.Hard || num.Fmt(gw) != v.W || num.Fmt(gh) != v.H || len(got) != len(v.Lines)
			if !bad {
				for i := range got {
					if got[i] != v.Lines[i] {
						bad = true
						break
					}
				}
			}
			if bad {
				failed++
				if failed <= 5 {
					t.Errorf("%q tại maxw=%d:\n  Go   lines=%q hard=%v w=%s h=%s\n  Py   lines=%q hard=%v w=%s h=%s",
						s.Text, v.MaxW, got, tm.Hard(), num.Fmt(gw), num.Fmt(gh),
						v.Lines, v.Hard, v.W, v.H)
				}
			}
		}
	}
	if failed > 5 {
		t.Errorf("... và %d vector lệch nữa", failed-5)
	}
	t.Logf("đã so %d vector", checked)
}
