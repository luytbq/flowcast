package layout

import (
	"container/heap"
	"sort"
	"strings"
)

// topo sắp các node theo thứ tự topo, bỏ qua cạnh back. Cùng mức thì node đứng
// trước trong bảng ra trước.
//
// Node còn lại sau khi hết đường, tức nằm trong một vòng lặp chưa đánh dấu
// back=true, được nối vào cuối theo thứ tự bảng, kèm một cảnh báo.
func (l *Layout) topo() []string {
	indeg := map[string]int{}
	var flow []*Item
	for _, it := range l.ItemOrder {
		if it.Attach == "" {
			indeg[it.ID] = 0
			flow = append(flow, it)
		}
	}
	for _, e := range l.Edges {
		if _, ok := indeg[e.Dst]; ok && !e.Back {
			indeg[e.Dst]++
		}
	}
	h := &orderHeap{}
	for _, it := range flow {
		if indeg[it.ID] == 0 {
			heap.Push(h, it)
		}
	}
	var out []string
	done := map[string]bool{}
	for h.Len() > 0 {
		n := heap.Pop(h).(*Item)
		out = append(out, n.ID)
		done[n.ID] = true
		for _, e := range l.outs[n.ID] {
			if e.Back {
				continue
			}
			indeg[e.Dst]--
			if indeg[e.Dst] == 0 {
				heap.Push(h, l.items[e.Dst])
			}
		}
	}
	var rest []*Item
	for _, it := range flow {
		if !done[it.ID] {
			rest = append(rest, it)
		}
	}
	sort.SliceStable(rest, func(i, j int) bool { return rest[i].Order < rest[j].Order })
	if len(rest) > 0 {
		ids := make([]string, len(rest))
		for i, it := range rest {
			ids[i] = it.ID
		}
		l.warn("layout.unmarked-cycle", "", "có vòng lặp chưa đánh dấu back=true, xếp theo thứ tự bảng: %s",
			strings.Join(ids, ", "))
		out = append(out, ids...)
	}
	return out
}

// orderHeap là min-heap theo thứ tự dòng trong bảng. Thứ tự đó là duy nhất cho
// mỗi phần tử, nên kết quả không phụ thuộc thứ tự đẩy vào.
type orderHeap []*Item

func (h orderHeap) Len() int           { return len(h) }
func (h orderHeap) Less(i, j int) bool { return h[i].Order < h[j].Order }
func (h orderHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *orderHeap) Push(x any)        { *h = append(*h, x.(*Item)) }
func (h *orderHeap) Pop() any {
	old := *h
	n := old[len(old)-1]
	*h = old[:len(old)-1]
	return n
}
