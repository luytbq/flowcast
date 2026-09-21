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
	// Back đánh dấu cạnh tạo vòng lặp. Nó không tham gia xếp hàng.
	Back bool
}

// Layout là trạng thái của một lần xếp hình. Các pha chạy lần lượt và mỗi pha
// đọc kết quả của pha trước.
type Layout struct {
	Cfg Config
	tm  *text.Measure

	Lanes   []model.Row
	laneIdx map[string]int

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
			Back: r.Meta["back"] == "true",
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
