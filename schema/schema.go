// Package schema khai báo các key metadata hợp lệ cho từng loại dòng.
//
// Bảng Flow Table giữ đúng năm cột, nên cột metadata chở cả tham chiếu bắt
// buộc như from, to, attach lẫn key tùy chọn như style. Tính tường minh vì vậy
// không nằm được trong bảng; nó nằm ở đây. Xem docs/adr/0001.
package schema

// Loại dòng.
var (
	NodeTypes   = set("start", "end", "task", "condition", "external")
	AttachTypes = set("db", "text")
	AllTypes    = union(NodeTypes, AttachTypes, set("lane", "edge"))
)

// ReservedIDs là các id draw.io tự dùng, không được trùng.
var ReservedIDs = set("0", "1", "pool")

// RestMarker là nội dung dòng text đánh dấu phần còn lại của bảng.
const RestMarker = "Phần còn lại"

// Ref là một key metadata trỏ tới id của dòng khác.
type Ref struct {
	Key      string
	Required bool
	// Note là đuôi câu khi tham chiếu trỏ tới loại không nhận được. Nằm ở đây
	// vì thông điệp là một phần của giao diện: người dùng đọc nó, và cổng đối
	// chiếu so nguyên văn.
	Note string
	// SameLane bật thì cảnh báo khi đích nằm ở lane khác.
	SameLane bool
}

// TypeSpec là lược đồ metadata của một loại dòng.
type TypeSpec struct {
	// Keys là các key được phép, theo thứ tự khai báo.
	Keys        []string
	StyleValues []string
	// Refs theo đúng thứ tự cần kiểm, vì thứ tự phát hiện là một phần của
	// giao diện.
	Refs []Ref
}

const edgeNote = "cạnh chỉ nối start/end/task/condition/external"
const attachNote = "chỉ gắn được vào start/end/task/condition/external"

// defaultSpec áp cho mọi loại node: chỉ nhận style, và style chỉ nhận highlight.
var defaultSpec = TypeSpec{Keys: []string{"style"}, StyleValues: []string{"highlight"}}

var swimlane = map[string]TypeSpec{
	"lane": {Keys: nil, StyleValues: []string{"highlight"}},
	"edge": {
		Keys:        []string{"from", "to", "style", "back"},
		StyleValues: []string{"highlight", "dashed"},
		Refs: []Ref{
			{Key: "from", Required: true, Note: edgeNote},
			{Key: "to", Required: true, Note: edgeNote},
		},
	},
	"db": {
		Keys:        []string{"attach", "style"},
		StyleValues: []string{"highlight"},
		Refs:        []Ref{{Key: "attach", Required: true, Note: attachNote, SameLane: true}},
	},
	"text": {
		Keys:        []string{"attach", "style"},
		StyleValues: []string{"highlight"},
		Refs:        []Ref{{Key: "attach", Note: attachNote, SameLane: true}},
	},
}

// For trả về lược đồ của một loại dòng.
func For(kind string) TypeSpec {
	if s, ok := swimlane[kind]; ok {
		return s
	}
	return defaultSpec
}

func (s TypeSpec) AllowsKey(k string) bool   { return contains(s.Keys, k) }
func (s TypeSpec) AllowsStyle(v string) bool { return contains(s.StyleValues, v) }

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func set(xs ...string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

func union(ms ...map[string]bool) map[string]bool {
	out := map[string]bool{}
	for _, m := range ms {
		for k := range m {
			out[k] = true
		}
	}
	return out
}
