package conformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luytbq/flowcast/layout"
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

type sizeRow struct {
	Text  string   `json:"text"`
	Lines []string `json:"lines"`
	W     string   `json:"w"`
	H     string   `json:"h"`
}

type sizeKind struct {
	Kind string    `json:"kind"`
	Rows []sizeRow `json:"rows"`
}

type vectorFile struct {
	Size    float64        `json:"size"`
	LineH   float64        `json:"line_h"`
	Samples []sampleVector `json:"samples"`
	Sizes   []sizeKind     `json:"sizes"`
	Budgets map[string]int `json:"budgets"`
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
		t.Fatalf("không đọc được vector: %v", err)
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

// TestNganSachNgatDong chốt thẳng con số, không qua hành vi.
//
// Một hằng số lệch 2px gần như không lộ ra qua đầu ra: ranh giới ký tự phải rơi
// đúng vào khoảng lệch đó mới đổi số dòng, và với chữ rộng khoảng 7px thì cửa
// sổ 2px thường rỗng. Đã kiểm bằng mutation test.
func TestNganSachNgatDong(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(Dir(), "text-vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var vf vectorFile
	if err := json.Unmarshal(data, &vf); err != nil {
		t.Fatal(err)
	}
	if len(vf.Budgets) == 0 {
		t.Fatal("vector không có mục ngân sách")
	}
	cfg := layout.DefaultConfig()
	for kind, want := range vf.Budgets {
		if kind == "__label__" {
			if got := cfg.LabelWrap; got != want {
				t.Errorf("ngân sách nhãn cạnh = %d, cần %d", got, want)
			}
			continue
		}
		if got := layout.WrapBudget(cfg, kind); got != float64(want) {
			t.Errorf("ngân sách của %q = %v, cần %d", kind, got, want)
		}
	}
	t.Logf("đã so %d ngân sách", len(vf.Budgets))
}

// TestVectorKichThuoc chốt ngân sách ngắt dòng của từng loại phần tử.
//
// Ngân sách là hằng số trong SizeItem, và nó gần như vô hình với bộ case sơ đồ:
// Wrap thu hẹp về bề rộng nhỏ nhất vẫn giữ nguyên số dòng, nên lệch vài pixel
// hiếm khi đổi đầu ra. Thang bậc dài dần dưới đây vượt qua mọi ranh giới số
// dòng, nên chỗ lật dịch đi ngay khi ngân sách sai dù chỉ 2px.
func TestVectorKichThuoc(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(Dir(), "text-vectors.json"))
	if err != nil {
		t.Fatalf("không đọc được vector: %v", err)
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
	cfg := layout.DefaultConfig()

	checked, failed := 0, 0
	for _, k := range vf.Sizes {
		for _, r := range k.Rows {
			checked++
			lines, w, h := layout.SizeItem(tm, cfg, k.Kind, []string{r.Text})
			bad := num.Fmt(w) != r.W || num.Fmt(h) != r.H || len(lines) != len(r.Lines)
			if !bad {
				for i := range lines {
					if lines[i] != r.Lines[i] {
						bad = true
						break
					}
				}
			}
			if bad {
				failed++
				if failed <= 5 {
					t.Errorf("%s %q:\n  Go lines=%q w=%s h=%s\n  Py lines=%q w=%s h=%s",
						k.Kind, r.Text, lines, num.Fmt(w), num.Fmt(h), r.Lines, r.W, r.H)
				}
			}
		}
	}
	if failed > 5 {
		t.Errorf("... và %d vector lệch nữa", failed-5)
	}
	if checked == 0 {
		t.Fatal("không có vector kích thước nào")
	}
	t.Logf("đã so %d vector kích thước", checked)
}
