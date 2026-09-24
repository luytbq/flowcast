package layout

// Result là sơ đồ đã xếp xong, ở dạng dữ liệu thuần: toạ độ đã có và thuật toán
// đã chạy xong.
//
// Tự kiểm và writer đọc Result chứ không đọc Layout. Tự kiểm nhờ vậy kiểm được
// bất kỳ hình học nào, kể cả hình học hỏng mà một engine đúng không bao giờ sinh
// ra; đó là cách duy nhất để kiểm chính tự kiểm.
//
// Phần tử mang hình nguyên thủy trong Shape, và writer chỉ đọc Shape. Kind là
// loại ngữ nghĩa (task, condition, ...), giữ lại cho caller muốn biết phần tử
// là gì, ví dụ khi ghi toạ độ ra JSON.
type Result struct {
	PoolW, PoolH           float64
	Origin                 [2]float64
	PoolHeader, LaneHeader int
	// NoLanes: sơ đồ không có lane. Lanes khi đó là một lane ẩn không được vẽ,
	// và không có pool.
	NoLanes bool
	// Dir là hướng của sơ đồ. Result do Layout.Result trả về luôn ở không gian
	// TD ảo; Orient đưa nó về hướng thật. Writer cần biết hướng để vẽ lane
	// thành băng dọc hay băng ngang.
	Dir          string
	MinChannel   int
	Lanes        []PlacedLane
	LaneX, LaneW []float64
	Items        []PlacedItem // theo thứ tự dòng trong bảng
	Edges        []PlacedEdge // theo thứ tự dòng trong bảng
	// TopoOrder là thứ tự topo của các node. Merge đặt node mới theo thứ tự này
	// để node đứng trước trong luồng có chỗ trước.
	TopoOrder []string
}

type PlacedLane struct {
	ID    string
	Lines []string // tên lane như trong bảng, chưa ngắt dòng
}

type PlacedItem struct {
	ID         string
	Kind       string
	Shape      Shape
	Lane       int
	Order      int
	Attach     string
	Row, Col   int
	Lines      []string
	Highlight  bool
	X, Y, W, H float64
}

func (it PlacedItem) box() box { return box{it.X, it.Y, it.X + it.W, it.Y + it.H} }

type PlacedEdge struct {
	ID, Src, Dst        string
	Case, ExitSide      byte
	Order               int
	Lines               []string
	Dashed, Highlight   bool
	Bold, NoArrow       bool
	Back                bool
	ExitFrac, EntryFrac [2]float64
	Pts                 [][2]float64
	Label               *[4]float64
	LabelT              float64
	LabelOff            [2]float64

	// Các trường dưới chỉ do merge đặt.

	// Auto: để draw.io tự đi dây, không ghi điểm neo, điểm gấp hay nhãn.
	Auto bool
	// Kept: dây giữ nguyên từ file cũ. Constraints thay cho điểm neo tính được,
	// Waypoints thay cho các điểm gấp.
	Kept        bool
	Constraints [][2]string
	Waypoints   [][2]float64
	// NoLabelPos: nhãn về giữa đường, vì dây đã bị sửa tay và vị trí nhãn tính
	// cho đường cũ không còn đúng.
	NoLabelPos bool
}

// Result chụp lại kết quả hiện tại của Layout. Gọi sau Run.
func (l *Layout) Result() Result {
	r := Result{
		PoolW: l.PoolW, PoolH: l.PoolH, Origin: Origin,
		PoolHeader: l.Cfg.PoolHeader, LaneHeader: l.Cfg.LaneHeader, MinChannel: l.Cfg.MinChannel,
		NoLanes: l.NoLanes, Dir: l.Dir,
		LaneX: l.LaneX, LaneW: l.LaneW, TopoOrder: l.TopoOrder,
	}
	for _, ln := range l.Lanes {
		r.Lanes = append(r.Lanes, PlacedLane{ln.ID, ln.Lines})
	}
	for _, it := range l.ItemOrder {
		r.Items = append(r.Items, PlacedItem{ID: it.ID, Kind: it.Kind, Shape: kindOf(it.Kind).Shape, Lane: it.Lane, Order: it.Order, Attach: it.Attach,
			Row: it.Row, Col: it.Col, Lines: it.Lines, Highlight: it.Highlight, X: it.X, Y: it.Y, W: it.W, H: it.H})
	}
	for _, e := range l.Edges {
		r.Edges = append(r.Edges, PlacedEdge{
			ID: e.ID, Src: e.Src, Dst: e.Dst, Case: e.Case, ExitSide: e.ExitSide, Order: e.Order, Back: e.Back,
			Lines: e.Lines, Dashed: e.Dashed, Highlight: e.Highlight, Bold: e.Bold, NoArrow: e.NoArrow,
			ExitFrac: e.ExitFrac, EntryFrac: e.EntryFrac, Pts: e.Pts,
			Label: e.Label, LabelT: e.LabelT, LabelOff: e.LabelOff,
		})
	}
	return r
}
