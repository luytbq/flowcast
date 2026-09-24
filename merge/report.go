package merge

import (
	"strconv"
	"strings"
)

// Report liệt kê những gì merge đã làm, để người dùng biết chỗ nào cần xem lại.
type Report struct {
	Pinned           []string // node giữ vị trí cũ
	Placed           []string // node mới được đặt
	Shifted          []string // node mới phải dịch xuống vì chồng lên thứ khác
	LaneChanged      []string // node đổi lane, được đặt lại như node mới
	KeptEdges        []string // dây giữ điểm gấp cũ
	Rerouted         []string // dây để draw.io tự đi
	LabelsCentered   []string // dây đã sửa tay, nhãn về giữa đường
	Removed          []string // cell của tool không còn trong bảng
	LanesGrown       []string // "id +Npx"
	FreehandKept     []string
	FreehandDropped  []string // cha đã bị xoá
	FreehandDetached []string // "id.source" hoặc "id.target"
	Overlaps         []string // "a/b"
}

// Lines là các dòng CLI in ra. Thứ tự và lời là giao diện, được chốt bằng bản
// ghi CLI.
func (r Report) Lines() []string {
	row := func(label string, xs []string) string {
		list := "-"
		if len(xs) > 0 {
			list = strings.Join(xs, ", ")
		}
		return "merge: " + label + " (" + strconv.Itoa(len(xs)) + "): " + list
	}
	return []string{
		row("giữ vị trí", r.Pinned),
		row("element mới", r.Placed),
		row("element mới phải dịch xuống vì chồng", r.Shifted),
		row("đổi lane trong bảng hoặc bị kéo sang lane khác, đặt lại", r.LaneChanged),
		row("edge giữ waypoint", r.KeptEdges),
		row("edge để draw.io tự đi dây, cần chỉnh tay", r.Rerouted),
		row("edge đã sửa tay, nhãn về giữa đường", r.LabelsCentered),
		row("xoá vì không còn trong bảng", r.Removed),
		row("lane được nới", r.LanesGrown),
		row("cell tự vẽ giữ lại", r.FreehandKept),
		row("cell tự vẽ bỏ đi (cha đã bị xoá)", r.FreehandDropped),
		row("cell tự vẽ mất đầu nối", r.FreehandDetached),
		row("element chồng nhau", r.Overlaps),
	}
}
