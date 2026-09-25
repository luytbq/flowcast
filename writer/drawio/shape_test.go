package drawio

import (
	"testing"

	"github.com/luytbq/flowcast/layout"
)

func TestEveryPrimitiveShapeHasStyle(t *testing.T) {
	for kind, k := range layout.Geometry {
		if _, ok := shapeStyle[k.Shape]; !ok {
			t.Errorf("shape %s of kind %s has no style", k.Shape, kind)
		}
	}
}
