package layout

import (
	"math"

	"github.com/luytbq/flowcast/num"
)

// Shape là hình nguyên thủy của một phần tử. Result và writer nói bằng hình
// nguyên thủy, không bằng loại ngữ nghĩa, để writer không phải biết về kind.
type Shape string

const (
	ShapeRect          Shape = "rect"
	ShapeDiamond       Shape = "diamond"
	ShapeEllipse       Shape = "ellipse"
	ShapeDoubleEllipse Shape = "double-ellipse"
	ShapeDashedEllipse Shape = "dashed-ellipse"
	ShapeCylinder      Shape = "cylinder"
	ShapeNote          Shape = "note"
)

// NodeKind là khai báo hình học của một loại phần tử: engine và writer chỉ đọc
// khai báo này, không so tên loại. Thêm một loại phần tử là thêm một dòng vào
// Kinds; kích thước và ngắt dòng của loại mới lấy mặc định ở SizeItem nếu không
// khai báo riêng.
//
// Luật dây ra dùng chung cho mọi loại nên không nằm ở đây: dây ra chỉ đi ở mặt
// trái, phải hoặc đáy, không bao giờ ở đỉnh.
type NodeKind struct {
	Shape Shape
	// EntryTopOnly: dây vào chỉ nối vào đỉnh. Engine không đặt phần tử cùng hàng
	// với nguồn của nó và không cho dây đi ngang vào mặt bên.
	EntryTopOnly bool
}

// Kinds khai báo hình học cho từng loại phần tử.
var Kinds = map[string]NodeKind{
	"task":      {Shape: ShapeRect},
	"condition": {Shape: ShapeDiamond, EntryTopOnly: true},
	"start":     {Shape: ShapeEllipse},
	"end":       {Shape: ShapeDoubleEllipse},
	"external":  {Shape: ShapeDashedEllipse},
	"db":        {Shape: ShapeCylinder},
	"text":      {Shape: ShapeNote},
}

// kindOf trả về khai báo của một loại; loại chưa khai báo coi như hộp chữ nhật.
func kindOf(kind string) NodeKind {
	if k, ok := Kinds[kind]; ok {
		return k
	}
	return NodeKind{Shape: ShapeRect}
}

// singlePort nói hình này chỉ có một điểm nối giữa mỗi mặt. Đường viền xiên
// hoặc cong không có đoạn thẳng nào để chia cổng như cạnh hộp chữ nhật, nên
// nhiều dây trên cùng một mặt phải gộp vào đúng đỉnh của mặt đó, trừ khi buộc
// phải tách.
func (s Shape) singlePort() bool {
	switch s {
	case ShapeDiamond, ShapeEllipse, ShapeDoubleEllipse, ShapeDashedEllipse:
		return true
	}
	return false
}

// outline là điểm trên đường viền của hình, trên mặt side, ở vị trí t dọc theo
// mặt đó, tính bằng tỉ lệ khung bao. Hộp chữ nhật thì điểm nằm ngay trên cạnh;
// hình thoi và elip thì điểm lùi vào trong khung bao cho tới khi chạm đường
// viền, để draw.io vẽ đầu dây đúng chỗ đã tính. Làm tròn tới hai chữ số như
// writer ghi ra, để merge đọc lại file thấy cổng khớp với cổng tính được.
func (s Shape) outline(side byte, t float64) [2]float64 {
	inset := 0.0
	switch s {
	case ShapeDiamond:
		inset = math.Abs(t - 0.5)
	case ShapeEllipse, ShapeDoubleEllipse, ShapeDashedEllipse:
		d := float64(2*t) - 1
		inset = num.Round(0.5-float64(0.5*math.Sqrt(1-float64(d*d))), 2)
	}
	far := num.Round(1-inset, 2)
	switch side {
	case 'B':
		return [2]float64{t, far}
	case 'T':
		return [2]float64{t, inset}
	case 'R':
		return [2]float64{far, t}
	}
	return [2]float64{inset, t}
}
