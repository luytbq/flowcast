package layout

import "sort"

// sameLaneOuts trả về các cạnh ra không phải back mà đích cùng lane với nguồn,
// theo thứ tự dòng trong bảng.
func (l *Layout) sameLaneOuts(nid string) []*Edge {
	u := l.items[nid]
	var out []*Edge
	for _, x := range l.outs[nid] {
		if !x.Back && l.items[x.Dst].Lane == u.Lane {
			out = append(out, x)
		}
	}
	return out
}

// branchOffset là khoảng cách từ cạnh này tới nhánh chính, tính bằng số cạnh.
// Nhánh chính là cạnh ra cuối cùng và có offset 0.
func (l *Layout) branchOffset(e *Edge) int {
	same := l.sameLaneOuts(e.Src)
	for i, x := range same {
		if x == e {
			return len(same) - 1 - i
		}
	}
	return 0
}

// branchDrift nói nhánh này rốt cuộc đi sang lane bên nào: -1 trái, 1 phải, 0
// chưa rõ.
//
// Đi dọc nhánh tới cạnh đầu tiên rời khỏi lane. Dừng ở node hợp nhánh vì từ đó
// trở đi là luồng chung, không còn là hướng của riêng nhánh này.
func (l *Layout) branchDrift(e *Edge) int {
	v := l.items[e.Dst]
	seen := map[string]bool{v.ID: true}
	queue := []string{v.ID}
	for len(queue) > 0 {
		nid := queue[0]
		queue = queue[1:]
		outs := append([]*Edge(nil), l.outs[nid]...)
		sort.SliceStable(outs, func(i, j int) bool { return outs[i].Order < outs[j].Order })
		for _, x := range outs {
			if x.Back {
				continue
			}
			w := l.items[x.Dst]
			if w.Lane != v.Lane {
				if w.Lane < v.Lane {
					return -1
				}
				return 1
			}
			if seen[w.ID] {
				continue
			}
			if l.nonBackIn(w.ID) > 1 {
				continue
			}
			seen[w.ID] = true
			queue = append(queue, w.ID)
		}
	}
	return 0
}

func (l *Layout) nonBackIn(id string) int {
	n := 0
	for _, y := range l.ins[id] {
		if !y.Back {
			n++
		}
	}
	return n
}

// branchSlots trả về cột tương đối cho mọi cạnh ra cùng lane của một node.
//
// Nhánh chính giữ cột 0. Nhánh phụ, gần nhánh chính trước, lệch sang phía mà
// nhánh đó dẫn tới, để mũi tên rời nhánh không phải vòng ngược qua node khác.
// Nhánh không rõ hướng xen kẽ phải rồi trái. Mặt nào đã có mũi tên ngang thì
// mọi nhánh phụ dồn sang mặt kia.
//
// Kết quả đổi theo hside, tức theo những mũi tên ngang đã đặt tới lúc gọi.
func (l *Layout) branchSlots(nid string) map[string]int {
	same := l.sameLaneOuts(nid)
	res := map[string]int{}
	if len(same) == 0 {
		return res
	}
	res[same[len(same)-1].ID] = 0
	busy := l.hside[nid]
	onlyLeft := busy['R'] && !busy['L']
	onlyRight := busy['L'] && !busy['R']
	next := map[int]int{1: 1, -1: 1}
	for i := len(same) - 2; i >= 0; i-- {
		e := same[i]
		var side int
		switch {
		case onlyLeft:
			side = -1
		case onlyRight:
			side = 1
		default:
			side = l.branchDrift(e)
			if side == 0 {
				side = -1
				if next[1] <= next[-1] {
					side = 1
				}
			}
		}
		res[e.ID] = side * next[side]
		next[side]++
	}
	return res
}

func (l *Layout) branchSlot(e *Edge) int { return l.branchSlots(e.Src)[e.ID] }

// mergeCol trả về cột cho node có nhiều nhánh cùng lane đi vào.
//
// Các nhánh đó rẽ ra từ một node chung; luồng chung nên quay về đúng cột của
// node rẽ gần nhất, thay vì bám theo cột của nhánh được xếp sau cùng. Không có
// node chung thì ok là false và caller giữ cách cũ.
func (l *Layout) mergeCol(same []*Edge, v *Item) (col int, ok bool) {
	var sets []map[string]bool
	for _, e := range same {
		seen := map[string]bool{}
		stack := []string{e.Src}
		for len(stack) > 0 {
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if seen[n] {
				continue
			}
			seen[n] = true
			for _, x := range l.ins[n] {
				w := l.items[x.Src]
				if !x.Back && w.Lane == v.Lane && w.Placed {
					stack = append(stack, x.Src)
				}
			}
		}
		sets = append(sets, seen)
	}
	var best *Item
	for id := range sets[0] {
		inAll := true
		for _, s := range sets[1:] {
			if !s[id] {
				inAll = false
				break
			}
		}
		if !inAll {
			continue
		}
		it := l.items[id]
		// Gần nhất nghĩa là hàng lớn nhất; hòa thì dòng đứng sau trong bảng.
		// Thứ tự dòng là duy nhất nên không phụ thuộc thứ tự duyệt map.
		if best == nil || it.Row > best.Row || (it.Row == best.Row && it.Order > best.Order) {
			best = it
		}
	}
	if best == nil {
		return 0, false
	}
	return best.Col, true
}

// sideBranches đếm số nhánh phụ cùng lane của một node.
func (l *Layout) sideBranches(v *Item) int {
	n := 0
	for _, e := range l.outs[v.ID] {
		if !e.Back && l.items[e.Dst].Lane == v.Lane && l.branchOffset(e) > 0 {
			n++
		}
	}
	return n
}

// attachSide chọn phía cho db và text đứng cạnh node: phía không có cạnh nối.
// Trả 1 cho phải, -1 cho trái. Cả hai phía đều có cạnh thì chọn phải.
func (l *Layout) attachSide(v *Item) int {
	right, left := false, false
	all := append(append([]*Edge(nil), l.outs[v.ID]...), l.ins[v.ID]...)
	for _, e := range all {
		if e.Back {
			continue
		}
		oid := e.Src
		if e.Src == v.ID {
			oid = e.Dst
		}
		o := l.items[oid]
		switch {
		case o.Lane > v.Lane:
			right = true
		case o.Lane < v.Lane:
			left = true
		case e.Src == v.ID && l.branchSlot(e) > 0:
			right = true
		case e.Src == v.ID && l.branchSlot(e) < 0:
			left = true
		}
	}
	if !right {
		return 1
	}
	if !left {
		return -1
	}
	return 1
}
