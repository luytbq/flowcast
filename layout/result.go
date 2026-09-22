package layout

// Result là sơ đồ đã xếp xong, ở dạng dữ liệu thuần: toạ độ đã có và thuật toán
// đã chạy xong.
//
// Tự kiểm và writer đọc Result chứ không đọc Layout. Tự kiểm nhờ vậy kiểm được
// bất kỳ hình học nào, kể cả hình học hỏng mà một engine đúng không bao giờ sinh
// ra; đó là cách duy nhất để kiểm chính tự kiểm.
//
// Phần tử còn mang loại ngữ nghĩa (task, condition, ...) chứ chưa mang hình
// nguyên thủy như mục 8 của docs/core-design.md mô tả. Lớp trừu tượng đó chỉ có
// giá trị khi có kind thứ hai; tới lúc đó writer mới thôi đọc Kind.
type Result struct {
	PoolW, PoolH           float64
	Origin                 [2]float64
	PoolHeader, LaneHeader int
	Lanes                  []PlacedLane
	LaneX, LaneW           []float64
	Items                  []PlacedItem // theo thứ tự dòng trong bảng
	Edges                  []PlacedEdge // theo thứ tự dòng trong bảng
}

type PlacedLane struct {
	ID    string
	Lines []string // tên lane như trong bảng, chưa ngắt dòng
}

type PlacedItem struct {
	ID         string
	Kind       string
	Lane       int
	Row, Col   int
	Lines      []string
	Highlight  bool
	X, Y, W, H float64
}

func (it PlacedItem) box() box { return box{it.X, it.Y, it.X + it.W, it.Y + it.H} }

type PlacedEdge struct {
	ID, Src, Dst        string
	Case, ExitSide      byte
	Lines               []string
	Dashed, Highlight   bool
	ExitFrac, EntryFrac [2]float64
	Pts                 [][2]float64
	Label               *[4]float64
	LabelT              float64
	LabelOff            [2]float64
}

// Result chụp lại kết quả hiện tại của Layout. Gọi sau Run.
func (l *Layout) Result() Result {
	r := Result{
		PoolW: l.PoolW, PoolH: l.PoolH, Origin: Origin,
		PoolHeader: l.Cfg.PoolHeader, LaneHeader: l.Cfg.LaneHeader,
		LaneX: l.LaneX, LaneW: l.LaneW,
	}
	for _, ln := range l.Lanes {
		r.Lanes = append(r.Lanes, PlacedLane{ln.ID, ln.Lines})
	}
	for _, it := range l.ItemOrder {
		r.Items = append(r.Items, PlacedItem{it.ID, it.Kind, it.Lane, it.Row, it.Col, it.Lines, it.Highlight, it.X, it.Y, it.W, it.H})
	}
	for _, e := range l.Edges {
		r.Edges = append(r.Edges, PlacedEdge{
			ID: e.ID, Src: e.Src, Dst: e.Dst, Case: e.Case, ExitSide: e.ExitSide,
			Lines: e.Lines, Dashed: e.Dashed, Highlight: e.Highlight,
			ExitFrac: e.ExitFrac, EntryFrac: e.EntryFrac, Pts: e.Pts,
			Label: e.Label, LabelT: e.LabelT, LabelOff: e.LabelOff,
		})
	}
	return r
}
