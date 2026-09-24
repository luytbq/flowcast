package layout

import (
	"testing"

	"github.com/luytbq/flowcast/schema"
)

func TestMoiLoaiPhanTuDeuCoKhaiBaoHinhHoc(t *testing.T) {
	for _, set := range []map[string]bool{schema.NodeTypes, schema.AttachTypes} {
		for kind := range set {
			if _, ok := Kinds[kind]; !ok {
				t.Errorf("loại %s chưa có khai báo trong Kinds", kind)
			}
		}
	}
}

func TestDiemTrenDuongVienNamTrenHinh(t *testing.T) {
	for _, tc := range []struct {
		shape Shape
		side  byte
		t     float64
		want  [2]float64
	}{
		{ShapeRect, 'L', 0.25, [2]float64{0, 0.25}},
		{ShapeDiamond, 'L', 0.25, [2]float64{0.25, 0.25}},
		{ShapeDiamond, 'B', 0.75, [2]float64{0.75, 0.75}},
		{ShapeEllipse, 'R', 0.75, [2]float64{0.93, 0.75}},
		{ShapeEllipse, 'L', 0.5, [2]float64{0, 0.5}},
	} {
		if got := tc.shape.outline(tc.side, tc.t); got != tc.want {
			t.Errorf("%s mặt %c tại %v: %v, muốn %v", tc.shape, tc.side, tc.t, got, tc.want)
		}
	}
}
