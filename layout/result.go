package layout

// Result là sơ đồ đã xếp xong, ở dạng dữ liệu thuần: toạ độ đã có và thuật toán
// đã chạy xong.
//
// Tự kiểm đọc Result chứ không đọc Layout. Nhờ vậy nó kiểm được bất kỳ hình học
// nào, kể cả hình học hỏng mà một engine đúng không bao giờ sinh ra; đó là cách
// duy nhất để kiểm chính tự kiểm.
type Result struct {
	LaneX, LaneW []float64
	Items        []PlacedItem // theo thứ tự dòng trong bảng
	Edges        []PlacedEdge // theo thứ tự dòng trong bảng
}

type PlacedItem struct {
	ID         string
	Lane       int
	X, Y, W, H float64
}

func (it PlacedItem) box() box { return box{it.X, it.Y, it.X + it.W, it.Y + it.H} }

type PlacedEdge struct {
	ID, Src, Dst string
	Pts          [][2]float64
	Label        *[4]float64
}

// Result chụp lại kết quả hiện tại của Layout. Gọi sau Run.
func (l *Layout) Result() Result {
	r := Result{LaneX: l.LaneX, LaneW: l.LaneW}
	for _, it := range l.ItemOrder {
		r.Items = append(r.Items, PlacedItem{it.ID, it.Lane, it.X, it.Y, it.W, it.H})
	}
	for _, e := range l.Edges {
		r.Edges = append(r.Edges, PlacedEdge{e.ID, e.Src, e.Dst, e.Pts, e.Label})
	}
	return r
}
