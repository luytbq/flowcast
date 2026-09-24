package drawio

import (
	"testing"

	"github.com/luytbq/flowcast/layout"
)

func TestMoiHinhNguyenThuyDeuCoStyle(t *testing.T) {
	for kind, k := range layout.Kinds {
		if _, ok := shapeStyle[k.Shape]; !ok {
			t.Errorf("hình %s của loại %s chưa có style", k.Shape, kind)
		}
	}
}
