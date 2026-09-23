package conformance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luytbq/flowcast"
	"github.com/luytbq/flowcast/layout"
	"github.com/luytbq/flowcast/merge"
)

var dirCases = []string{"cases/05-merge-node.md", "cases/28-tracks-fan-in.md", "mermaid/01-dat-hang.mmd"}

// BT là ảnh lật của TD, và RL là ảnh lật của LR: kích thước node không đổi nên
// cách xếp giống hệt, chỉ khác chiều. LR thì khác TD thật sự, vì bề rộng và bề
// cao của node được hoán đổi trước khi xếp.
func TestHuongLatLaAnhLatCuaHuongGoc(t *testing.T) {
	for _, c := range dirCases {
		src := dirSource(t, c)
		for _, pair := range [][2]string{{layout.DirTD, layout.DirBT}, {layout.DirLR, layout.DirRL}} {
			a, err := flowcast.Build(src, flowcast.Options{Direction: pair[0]})
			if err != nil {
				t.Fatal(err)
			}
			b, err := flowcast.Build(src, flowcast.Options{Direction: pair[1]})
			if err != nil {
				t.Fatal(err)
			}
			if a.Stats != b.Stats {
				t.Errorf("%s %s và %s: số liệu %+v khác %+v", c, pair[0], pair[1], a.Stats, b.Stats)
			}
			if len(a.Findings) != len(b.Findings) {
				t.Errorf("%s %s và %s: phát hiện %d khác %d", c, pair[0], pair[1], len(a.Findings), len(b.Findings))
			}
			for i := range a.Layout.Edges {
				if a.Layout.Edges[i].Case != b.Layout.Edges[i].Case {
					t.Errorf("%s: cạnh %s đi dây kiểu %c ở %s nhưng %c ở %s", c, a.Layout.Edges[i].ID,
						a.Layout.Edges[i].Case, pair[0], b.Layout.Edges[i].Case, pair[1])
				}
			}
		}
	}
}

func dirSource(t *testing.T, c string) flowcast.Source {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(Dir(), c))
	if err != nil {
		t.Fatal(err)
	}
	return flowcast.Source{Name: filepath.Base(c), Data: data}
}

// Mọi hướng phải giữ nguyên số phần tử, số cạnh và số phát hiện tự kiểm, và
// mỗi cạnh xuôi phải đi đúng chiều của hướng.
func TestHuongGiuNguyenCauTrucVaDungChieu(t *testing.T) {
	for _, c := range dirCases {
		src := dirSource(t, c)
		td, err := flowcast.Build(src, flowcast.Options{Direction: layout.DirTD})
		if err != nil {
			t.Fatal(err)
		}
		checkFlowDirection(t, c, layout.DirTD, *td.Layout)
		for _, dir := range []string{layout.DirBT, layout.DirLR, layout.DirRL} {
			r, err := flowcast.Build(src, flowcast.Options{Direction: dir})
			if err != nil {
				t.Fatalf("%s %s: %v", c, dir, err)
			}
			if len(r.Findings) != len(td.Findings) || len(r.Layout.Items) != len(td.Layout.Items) ||
				len(r.Layout.Edges) != len(td.Layout.Edges) {
				t.Errorf("%s %s: phát hiện %d, phần tử %d, cạnh %d; TD có %d, %d, %d", c, dir,
					len(r.Findings), len(r.Layout.Items), len(r.Layout.Edges),
					len(td.Findings), len(td.Layout.Items), len(td.Layout.Edges))
			}
			checkFlowDirection(t, c, dir, *r.Layout)
		}
	}
}

// checkFlowDirection: mỗi cạnh xuôi phải đi đúng chiều của hướng, đo trên tâm
// của node nguồn và node đích.
func checkFlowDirection(t *testing.T, c, dir string, r layout.Result) {
	t.Helper()
	center := map[string][2]float64{}
	for _, it := range r.Items {
		center[it.ID] = [2]float64{it.X + it.W/2, it.Y + it.H/2}
	}
	for _, e := range r.Edges {
		if e.Back {
			continue
		}
		a, b := center[e.Src], center[e.Dst]
		var ok bool
		switch dir {
		case layout.DirTD:
			ok = b[1] >= a[1]
		case layout.DirBT:
			ok = b[1] <= a[1]
		case layout.DirLR:
			ok = b[0] >= a[0]
		case layout.DirRL:
			ok = b[0] <= a[0]
		}
		if !ok {
			t.Errorf("%s %s: cạnh %s đi ngược chiều: %v tới %v", c, dir, e.ID, a, b)
		}
	}
}

func TestHuongLRVeLaneThanhBangNgang(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(Dir(), "cases", "05-merge-node.md"))
	if err != nil {
		t.Fatal(err)
	}
	src := flowcast.Source{Name: "a.md", Data: data}
	for dir, want := range map[string]bool{layout.DirTD: false, layout.DirBT: false, layout.DirLR: true, layout.DirRL: true} {
		r, err := flowcast.Build(src, flowcast.Options{Direction: dir})
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Contains(r.Text, "horizontal=0"); got != want {
			t.Errorf("%s: lane ngang %v, mong %v", dir, got, want)
		}
	}
}

func TestHuongLaKhongHopLeThiBaoLoi(t *testing.T) {
	src := flowcast.Source{Name: "a.md", Data: []byte("| id | type | parent | content | metadata |\n|-|-|-|-|-|\n")}
	if _, err := flowcast.Build(src, flowcast.Options{Direction: "XY"}); err == nil {
		t.Error("hướng XY phải bị từ chối")
	}
}

func TestMergeChuaHoTroHuongKhacTD(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(Dir(), "cases", "05-merge-node.md"))
	if err != nil {
		t.Fatal(err)
	}
	src := flowcast.Source{Name: "a.md", Data: data}
	fresh, err := flowcast.Build(src, flowcast.Options{Direction: layout.DirLR})
	if err != nil {
		t.Fatal(err)
	}
	old, err := merge.Read([]byte(fresh.Text), "cũ.drawio")
	if err != nil {
		t.Fatal(err)
	}
	_, err = flowcast.Build(src, flowcast.Options{Direction: layout.DirLR, Previous: old})
	if err == nil || !strings.Contains(err.Error(), "merge chưa hỗ trợ") {
		t.Errorf("lỗi %v", err)
	}
}

// Golden cho ba hướng mới: chống hồi quy, sinh lại bằng -update sau khi xem ảnh.
func TestHuongKhopGolden(t *testing.T) {
	for _, c := range dirCases {
		data, err := os.ReadFile(filepath.Join(Dir(), c))
		if err != nil {
			t.Fatal(err)
		}
		for _, dir := range []string{layout.DirBT, layout.DirLR, layout.DirRL} {
			r, err := flowcast.Build(flowcast.Source{Name: filepath.Base(c), Data: data},
				flowcast.Options{Direction: dir})
			if err != nil {
				t.Fatal(err)
			}
			name := strings.TrimSuffix(filepath.Base(c), filepath.Ext(c)) + "-" + dir + ".drawio"
			golden := filepath.Join(Dir(), "golden", "direction", name)
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
				t.Fatalf("%v; chạy go test ./conformance -run Huong -update rồi xem ảnh", err)
			}
			if r.Text != string(want) {
				t.Errorf("%s: lệch golden", name)
			}
		}
	}
}
