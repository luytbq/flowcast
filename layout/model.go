package layout

import (
	"github.com/luytbq/flowcast/model"
	"github.com/luytbq/flowcast/schema"
	"github.com/luytbq/flowcast/text"
)

// Item là một phần tử được vẽ: node, hoặc db và text đứng cạnh node.
type Item struct {
	ID        string
	Kind      string
	Lane      int
	Lines     []string
	W, H      float64
	Order     int // thứ tự dòng trong bảng, dùng để phá thế hòa
	Highlight bool
	// Attach là id của node mà db hoặc text đứng cạnh. Rỗng với node.
	Attach string

	// Row và Col chỉ có nghĩa khi Placed. Col âm được, vì nhánh phụ dạt sang
	// trái của cột gốc.
	Placed   bool
	Row, Col int
	X, Y     float64
}

// Edge là một mũi tên giữa hai node.
type Edge struct {
	ID        string
	Src, Dst  string
	Lines     []string
	LW, LH    float64
	Order     int
	Dashed    bool
	Highlight bool
	Bold      bool // nét đậm
	NoArrow   bool // không có đầu mũi tên
	// Back đánh dấu cạnh tạo vòng lặp. Nó không tham gia xếp hàng.
	Back bool

	// Kết quả của route. Case là một trong A, B, C, D.
	Case      byte
	ExitSide  byte
	EntrySide byte
	ExitFrac  [2]float64
	EntryFrac [2]float64
	Sym       []SymPair

	// Kết quả của pha hình học và nhãn, theo toạ độ trong pool.
	Pts      [][2]float64
	Label    *[4]float64 // hộp nhãn; nil khi cạnh không có nhãn hoặc không đặt được
	LabelT   float64     // vị trí nhãn dọc đường, từ -1 ở nguồn tới 1 ở đích
	LabelOff [2]float64  // độ lệch từ điểm neo trên đường tới tâm hộp nhãn
}

// Layout là trạng thái của một lần xếp hình. Các pha chạy lần lượt và mỗi pha
// đọc kết quả của pha trước.
type Layout struct {
	Cfg Config
	tm  *text.Measure

	Lanes   []model.Row
	laneIdx map[string]int
	// NoLanes: bảng không có lane nào, Lanes chỉ chứa một lane ẩn.
	NoLanes bool
	depth   map[string]int // bộ nhớ của branchDepth, làm mới mỗi lần Place
	// Dir là hướng của sơ đồ, xem axis.go. Rỗng nghĩa là TD.
	Dir string

	// items và Edges giữ thứ tự dòng trong bảng. Map của Go duyệt ngẫu nhiên,
	// nên mọi vòng duyệt cần thứ tự đều đi qua ItemOrder.
	items     map[string]*Item
	ItemOrder []*Item
	Edges     []*Edge
	outs, ins map[string][]*Edge

	Warnings []string

	// Kết quả của place.
	TopoOrder   []string
	NRows       int
	Cols        map[int][]int
	Attachments map[string][]*Item
	XOrd        map[XKey]int
	occ         map[cell]string
	hside       map[string]map[byte]bool

	// Kết quả của route.
	Segs    []*Seg
	NTracks map[Res]int
	sideOut map[sideKey][]*Edge
	sideIn  map[sideKey][]*Edge

	// Kết quả của pha hình học.
	LaneX, LaneW []float64
	PoolW, PoolH float64
	g            *geom
}

// Item trả về phần tử theo id.
func (l *Layout) Item(id string) *Item { return l.items[id] }

// New dựng trạng thái ban đầu từ các dòng đã qua validate.
//
// Không kiểm lại đầu vào: gọi với bảng còn lỗi là vi phạm điều kiện tiên quyết.
func New(rows []model.Row, cfg Config, tm *text.Measure) *Layout {
	l := &Layout{
		Cfg:     cfg,
		tm:      tm,
		laneIdx: map[string]int{},
		items:   map[string]*Item{},
		outs:    map[string][]*Edge{},
		ins:     map[string][]*Edge{},
	}
	byID := map[string]model.Row{}
	for _, r := range rows {
		byID[r.ID] = r
		if r.Type == "lane" {
			l.laneIdx[r.ID] = len(l.Lanes)
			l.Lanes = append(l.Lanes, r)
		}
	}
	if len(l.Lanes) == 0 {
		// Flowchart: một lane ẩn chứa mọi phần tử, không có header nào. Thuật
		// toán xếp hình chạy nguyên vẹn trên một lane.
		l.NoLanes = true
		l.laneIdx[""] = 0
		l.Lanes = []model.Row{{Type: "lane"}}
		l.Cfg.PoolHeader, l.Cfg.LaneHeader = 0, 0
	}
	for _, r := range rows {
		if !schema.NodeTypes[r.Type] && !(schema.AttachTypes[r.Type] && !isMarker(r)) {
			continue
		}
		attach := ""
		if schema.AttachTypes[r.Type] {
			attach = r.Meta["attach"]
		}
		// db và text vẽ trong lane của node chúng bám, không phải lane chúng
		// tự khai. validate đã cảnh báo khi hai lane này lệch nhau.
		laneID := r.Parent
		if attach != "" {
			laneID = byID[attach].Parent
		}
		lines, w, h := SizeItem(tm, cfg, r.Type, r.Lines)
		it := &Item{
			ID: r.ID, Kind: r.Type, Lane: l.laneIdx[laneID], Lines: lines, W: w, H: h,
			Order: r.Idx, Highlight: hasStyle(r, "highlight"), Attach: attach,
		}
		l.items[r.ID] = it
		l.ItemOrder = append(l.ItemOrder, it)
	}
	for _, r := range rows {
		if r.Type != "edge" {
			continue
		}
		lines, lw, lh := SizeLabel(tm, cfg, r.Lines)
		e := &Edge{
			ID: r.ID, Src: r.Meta["from"], Dst: r.Meta["to"], Lines: lines, LW: lw, LH: lh,
			Order: r.Idx, Dashed: hasStyle(r, "dashed"), Highlight: hasStyle(r, "highlight"),
			Bold: hasStyle(r, "bold"), NoArrow: hasStyle(r, "noarrow"),
			Back:      r.Meta["back"] == "true",
			EntrySide: 'T',
			ExitFrac:  [2]float64{0.5, 1.0},
			EntryFrac: [2]float64{0.5, 0.0},
		}
		l.Edges = append(l.Edges, e)
		l.outs[e.Src] = append(l.outs[e.Src], e)
		l.ins[e.Dst] = append(l.ins[e.Dst], e)
	}
	return l
}

func isMarker(r model.Row) bool {
	return r.Type == "text" && r.Text() == schema.RestMarker && r.Meta["attach"] == ""
}

func hasStyle(r model.Row, v string) bool {
	for _, s := range splitStyle(r.Meta["style"]) {
		if s == v {
			return true
		}
	}
	return false
}
