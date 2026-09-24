package conformance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/luytbq/flowcast"
)

// TestConditionVaoTuDinhRaBaMat duyệt mọi bảng của bộ đối chiếu: dây vào một
// condition luôn nối vào đỉnh, dây ra chỉ đi ở mặt trái, phải hoặc đáy.
func TestConditionVaoTuDinhRaBaMat(t *testing.T) {
	var paths []string
	for _, pat := range []string{"cases/*.md", "flowchart/*.md", "diverge/*.md"} {
		ps, _ := filepath.Glob(filepath.Join(Dir(), pat))
		paths = append(paths, ps...)
	}
	checked := 0
	for _, p := range paths {
		name := filepath.Base(p)
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		r, err := flowcast.Build(flowcast.Source{Name: name, Data: data}, flowcast.Options{})
		// Bộ đối chiếu có cả bảng sai cố ý; chúng không có bố cục để kiểm.
		if err != nil || r.Layout == nil || r.HasErrors() {
			continue
		}
		kind := map[string]string{}
		for _, it := range r.Layout.Items {
			kind[it.ID] = it.Kind
		}
		for _, e := range r.Layout.Edges {
			if kind[e.Dst] == "condition" && e.EntryFrac != [2]float64{0.5, 0} {
				t.Errorf("%s: %s vào condition %s tại %v, không phải đỉnh", name, e.ID, e.Dst, e.EntryFrac)
			}
			if kind[e.Src] == "condition" && e.ExitSide != 'L' && e.ExitSide != 'R' && e.ExitSide != 'B' {
				t.Errorf("%s: %s ra khỏi condition %s ở mặt %c", name, e.ID, e.Src, e.ExitSide)
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
