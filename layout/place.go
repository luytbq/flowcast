package layout

import (
	"fmt"
	"sort"
)

// nonRect là các loại node không phải hộp chữ nhật. Chúng chỉ nối dây ở bốn
// điểm giữa cạnh, nên một condition cần cả hai mặt bên cho các nhánh.
var nonRect = map[string]bool{"condition": true, "start": true, "end": true, "external": true}

// hspan là một mũi tên ngang đã đặt: nằm trên hàng row, chiếm khoảng lưới mở
// (a, b). Ô nằm trong khoảng đó coi như bị chiếm, vì mũi tên chạy xuyên qua.
type hspan struct {
	row  int
	a, b gk
}

// XKey là vị trí của một cột hoặc một máng trên trục ngang, trước khi có pixel.
// Kind là 'G' cho máng, 'c' cho cột; V là chỉ số máng hoặc số cột.
type XKey struct {
	Kind byte
	Lane int
	V    int
}

// Place gán lane, hàng và cột cho mọi phần tử.
//
// Duyệt theo thứ tự topo. Mỗi node thử đặt cùng hàng với nguồn để mũi tên đi
// ngang; không được thì xuống hàng dưới nguồn sâu nhất. Nhánh phụ dạt sang bên,
// và node hợp nhánh quay về cột của node rẽ chung gần nhất. db và text đặt sát
// cạnh node chúng bám.
func (l *Layout) Place() {
	l.occ = map[cell]string{}
	l.hside = map[string]map[byte]bool{}
	var spans []hspan

	l.Attachments = map[string][]*Item{}
	ordered := append([]*Item(nil), l.ItemOrder...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Order < ordered[j].Order })
	for _, it := range ordered {
		if it.Attach != "" {
			l.Attachments[it.Attach] = append(l.Attachments[it.Attach], it)
		}
	}

	inSpan := func(key gk, r int) bool {
		for _, s := range spans {
			if s.row == r && s.a.less(key) && key.less(s.b) {
				return true
			}
		}
		return false
	}
	blocked := func(lane, c, r int) bool {
		_, taken := l.occ[cell{lane, c, r}]
		return taken || inSpan(gk{lane, c}, r)
	}

	maxRow := -1
	l.TopoOrder = l.topo()
	for _, vid := range l.TopoOrder {
		v := l.items[vid]
		var preds, same []*Edge
		for _, e := range l.ins[vid] {
			if !e.Back && l.items[e.Src].Placed {
				preds = append(preds, e)
				if l.items[e.Src].Lane == v.Lane {
					same = append(same, e)
				}
			}
		}

		sideBranch, col := 0, 0
		if len(same) > 0 {
			e0 := same[0]
			for _, e := range same[1:] {
				a, b := l.items[e.Src], l.items[e0.Src]
				if a.Row > b.Row || (a.Row == b.Row && e.Order > e0.Order) {
					e0 = e
				}
			}
			slot := l.branchSlot(e0)
			col = l.items[e0.Src].Col + slot
			sideBranch = sign(slot)
			if len(same) > 1 {
				if mc, ok := l.mergeCol(same, v); ok {
					col, sideBranch = mc, 0
				}
			}
		}

		var row int
		switch {
		case len(preds) > 0:
			row = -1
			for _, e := range preds {
				if r := l.items[e.Src].Row + 1; r > row {
					row = r
				}
			}
		case v.Kind == "start":
			row = 0
		default:
			row = maxRow + 1
		}

		placed := false
		needsSides := nonRect[v.Kind] && l.sideBranches(v) >= 2
		if len(preds) == 1 && l.nonBackIn(vid) == 1 && !needsSides && (len(same) == 0 || sideBranch != 0) {
			u := l.items[preds[0].Src]
			r := u.Row
			a, b := gk{u.Lane, u.Col}, gk{v.Lane, col}
			if b.less(a) {
				a, b = b, a
			}
			free := !blocked(v.Lane, col, r)
			for k := range l.occ {
				if !free {
					break
				}
				if k.row == r && a.less(gk{k.lane, k.col}) && (gk{k.lane, k.col}).less(b) {
					free = false
				}
			}
			for _, s := range spans {
				if !free {
					break
				}
				if s.row == r && s.a.less(b) && a.less(s.b) {
					free = false
				}
			}
			if free {
				row = r
				spans = append(spans, hspan{r, a, b})
				placed = true
				toRight := (gk{u.Lane, u.Col}).less(gk{v.Lane, col})
				if toRight {
					l.addSide(u.ID, 'R')
					l.addSide(v.ID, 'L')
				} else {
					l.addSide(u.ID, 'L')
					l.addSide(v.ID, 'R')
				}
			}
		}

		// Nhánh phụ thử dạt sang bên tối đa ba cột trước khi chịu xuống hàng.
		tries := 0
		for !placed && blocked(v.Lane, col, row) {
			if sideBranch != 0 && tries < 3 {
				col += sideBranch
			} else {
				row++
			}
			tries++
		}
		v.Row, v.Col, v.Placed = row, col, true
		l.occ[cell{v.Lane, col, row}] = vid
		if row > maxRow {
			maxRow = row
		}

		for _, a := range l.Attachments[vid] {
			side := l.attachSide(v)
			c, found := 0, false
			for _, d := range []int{side, -side, 2 * side, -2 * side} {
				if !blocked(v.Lane, col+d, row) {
					c, found = col+d, true
					break
				}
			}
			if !found {
				// Lùi ra xa chỉ tránh ô đã chiếm, không tránh khoảng của mũi
				// tên ngang, đúng như bản tham chiếu.
				for d := 3; d < 50; d++ {
					if _, taken := l.occ[cell{v.Lane, col + d, row}]; !taken {
						c, found = col+d, true
						break
					}
				}
				if !found {
					panic(fmt.Sprintf("layout: không còn ô nào trong 50 cột cạnh %s", vid))
				}
				l.Warnings = append(l.Warnings, fmt.Sprintf("%s: không còn ô trống cạnh %s, đặt xa hơn", a.ID, vid))
			}
			a.Row, a.Col, a.Placed = row, c, true
			l.occ[cell{v.Lane, c, row}] = a.ID
		}
	}

	l.NRows = maxRow + 1
	l.Cols = map[int][]int{}
	for lane := range l.Lanes {
		l.Cols[lane] = []int{}
	}
	seen := map[[2]int]bool{}
	for k := range l.occ {
		if !seen[[2]int{k.lane, k.col}] {
			seen[[2]int{k.lane, k.col}] = true
			l.Cols[k.lane] = append(l.Cols[k.lane], k.col)
		}
	}
	for lane := range l.Cols {
		sort.Ints(l.Cols[lane])
	}
	l.XOrd = map[XKey]int{}
	n := 0
	for lane := range l.Lanes {
		cols := l.Cols[lane]
		for gi := 0; gi <= len(cols); gi++ {
			l.XOrd[XKey{'G', lane, gi}] = n
			n++
			if gi < len(cols) {
				l.XOrd[XKey{'c', lane, cols[gi]}] = n
				n++
			}
		}
	}
}

func (l *Layout) addSide(id string, side byte) {
	if l.hside[id] == nil {
		l.hside[id] = map[byte]bool{}
	}
	l.hside[id][side] = true
}
