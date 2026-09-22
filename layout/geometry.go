package layout

import (
	"math"
	"strings"

	"github.com/luytbq/flowcast/num"
)

// gzKey là một nửa của máng: phần sát cột bên trái ('l') hoặc bên phải ('r').
// Chỗ chừa cho nhãn và cho db bám sát được cộng vào đúng nửa đó.
type gzKey struct {
	lane, gi int
	side     byte
}

type hug struct {
	u    *Item
	side byte
}

// Hình học của một lần tính. Pha hình học chạy hai lượt, nên mọi thứ ở đây
// được dựng lại từ đầu mỗi lượt.
type geom struct {
	lzG, azG map[gzKey]float64
	lzC      map[int]float64
	hugs     map[string]hug
	gutX     map[[2]int]float64
	colX     map[[2]int][2]float64 // x trái và bề rộng của một cột
	chanY    map[int]float64
	rowY     map[int][2]float64 // y trên và bề cao của một hàng
}

// Origin là khoảng lề từ mép trang tới pool.
var Origin = [2]float64{40, 40}

// labelReservations chừa chỗ cho nhãn cạnh: ở máng phía cổng ra khi cạnh ra
// ngang, ở kênh ngay dưới nguồn khi cạnh ra đáy.
//
// spans là bề dài phần ngang đầu tiên của các cạnh B và C, đo được sau lượt
// tính đầu. Lượt đầu chưa có nên truyền nil.
func (l *Layout) labelReservations(spans map[string]float64) (map[gzKey]float64, map[int]float64) {
	lzG := map[gzKey]float64{}
	lzC := map[int]float64{}
	for _, e := range l.Edges {
		if len(e.Lines) == 0 {
			continue
		}
		u := l.items[e.Src]
		if e.ExitSide == 'B' {
			lzC[u.Row+1] = pyMax(lzC[u.Row+1], e.LH+8)
			continue
		}
		key := gzKey{u.Lane, l.gutter(u.Lane, u.Col, e.ExitSide), 'r'}
		if e.ExitSide == 'R' {
			key.side = 'l'
		}
		crosses := l.items[e.Dst].Lane != u.Lane
		switch {
		case e.Case == 'D' || (crosses && spans != nil):
			lzG[key] = pyMax(lzG[key], e.LW+8)
		case spans != nil:
			if deficit := e.LW + 16 - spans[e.ID]; deficit > 0 {
				lzG[key] = pyMax(lzG[key], deficit)
			}
		}
	}
	return lzG, lzC
}

// hugPlan chọn db và text nào được đặt bám sát node, và ở mặt nào.
//
// Chỉ bám khi mặt đó có đúng một phần tử ở ô sát bên, và không có dây nào ra
// hoặc vào mặt đó: đoạn dây ngang sát node sẽ cắt qua chỗ định đặt. Mặt bận thì
// phần tử căn giữa ô bên cạnh như mọi phần tử khác.
func (l *Layout) hugPlan() map[string]hug {
	hugs := map[string]hug{}
	for aid, atts := range l.Attachments {
		u := l.items[aid]
		for _, side := range []byte{'L', 'R'} {
			var near []*Item
			for _, a := range atts {
				if (a.Col > u.Col) == (side == 'R') && abs(a.Col-u.Col) == 1 {
					near = append(near, a)
				}
			}
			if len(near) != 1 || l.sideUsed(u, side) {
				continue
			}
			hugs[near[0].ID] = hug{u, side}
		}
	}
	return hugs
}

// computeGeometry đổi lưới lane, cột, hàng và số track thành toạ độ pixel.
//
// Thứ tự cộng giữ đúng như bản tham chiếu, vì cộng số thực không có tính kết
// hợp: x của cột tiếp theo được cộng dồn riêng từ bề rộng từng máng và từng
// cột, nên nó có thể lệch bit cuối so với x của lane cộng với tổng bề rộng.
func (l *Layout) computeGeometry(lzG map[gzKey]float64, lzC map[int]float64) {
	cfg := l.Cfg
	g := &geom{
		lzG: lzG, lzC: lzC, azG: map[gzKey]float64{}, hugs: l.hugPlan(),
		gutX: map[[2]int]float64{}, colX: map[[2]int][2]float64{},
		chanY: map[int]float64{}, rowY: map[int][2]float64{},
	}
	colW := map[[2]int]float64{}
	rowH := map[int]float64{}
	for _, it := range l.ItemOrder {
		if _, hugged := g.hugs[it.ID]; !hugged {
			colW[[2]int{it.Lane, it.Col}] = pyMax(colW[[2]int{it.Lane, it.Col}], it.W)
		}
		rowH[it.Row] = pyMax(rowH[it.Row], it.H)
	}
	for aid, h := range g.hugs {
		key := gzKey{h.u.Lane, l.gutter(h.u.Lane, h.u.Col, h.side), 'r'}
		if h.side == 'R' {
			key.side = 'l'
		}
		g.azG[key] = pyMax(g.azG[key], float64(cfg.AttachGap)+l.items[aid].W)
	}

	l.LaneX, l.LaneW = nil, nil
	x := 0.0
	for lane, lrow := range l.Lanes {
		cols := l.Cols[lane]
		widths := make([]float64, len(cols)+1)
		for gi := range widths {
			n := l.NTracks[Res{Kind: 'G', A: lane, B: gi}]
			inner := 0.0
			if n > 0 {
				inner = float64(2*cfg.GutterMargin + (n-1)*cfg.TrackGap)
			}
			widths[gi] = g.azG[gzKey{lane, gi, 'l'}] + g.azG[gzKey{lane, gi, 'r'}] +
				lzG[gzKey{lane, gi, 'l'}] + lzG[gzKey{lane, gi, 'r'}] + pyMax(float64(cfg.MinGutter), inner)
		}
		total := 0.0
		for _, w := range widths {
			total += w
		}
		cw := 0.0
		for _, c := range cols {
			cw += colW[[2]int{lane, c}]
		}
		total += cw
		// Tên lane nối các dòng bằng ký tự xuống dòng rồi mới đo, nên mỗi lần
		// xuống dòng được tính thêm bề rộng một glyph .notdef.
		head := l.tm.W(strings.TrimSpace(strings.Join(lrow.Lines, "\n"))) + 30
		if need := pyMax(float64(cfg.MinLaneW), head) - total; need > 0 {
			widths[0] += need / 2
			widths[len(widths)-1] += need / 2
			total += need
		}
		l.LaneX = append(l.LaneX, x)
		l.LaneW = append(l.LaneW, total)
		cx := x
		for gi := range widths {
			g.gutX[[2]int{lane, gi}] = cx
			cx += widths[gi]
			if gi < len(cols) {
				w := colW[[2]int{lane, cols[gi]}]
				g.colX[[2]int{lane, cols[gi]}] = [2]float64{cx, w}
				cx += w
			}
		}
		x += total
	}
	l.PoolW = x

	y := float64(cfg.PoolHeader + cfg.LaneHeader)
	for k := 0; k <= l.NRows; k++ {
		n := l.NTracks[Res{Kind: 'C', A: k}]
		inner := 0.0
		if n > 0 {
			inner = float64(2*cfg.ChannelMargin + (n-1)*cfg.TrackGap)
		}
		h := pyMax(float64(cfg.MinChannel), lzC[k]+inner)
		g.chanY[k] = y
		y += h
		if k < l.NRows {
			g.rowY[k] = [2]float64{y, rowH[k]}
			y += rowH[k]
		}
	}
	l.PoolH = y
	l.g = g

	for _, it := range l.ItemOrder {
		c, r := g.colX[[2]int{it.Lane, it.Col}], g.rowY[it.Row]
		it.X = c[0] + (c[1]-it.W)/2
		it.Y = r[0] + (r[1]-it.H)/2
	}
	for aid, h := range g.hugs {
		a := l.items[aid]
		if h.side == 'R' {
			a.X = h.u.X + h.u.W + float64(cfg.AttachGap)
		} else {
			a.X = h.u.X - float64(cfg.AttachGap) - a.W
		}
	}
}

func (l *Layout) trackX(s *Seg) float64 {
	g, cfg := l.g, l.Cfg
	return g.gutX[[2]int{s.Res.A, s.Res.B}] + g.azG[gzKey{s.Res.A, s.Res.B, 'l'}] +
		g.lzG[gzKey{s.Res.A, s.Res.B, 'l'}] + float64(cfg.GutterMargin) + float64(s.Track*cfg.TrackGap)
}

func (l *Layout) trackY(s *Seg) float64 {
	g, cfg := l.g, l.Cfg
	return g.chanY[s.Res.A] + g.lzC[s.Res.A] + float64(cfg.ChannelMargin) + float64(s.Track*cfg.TrackGap)
}

// resolvePaths dựng đường gấp khúc cho từng cạnh từ sym của nó, rồi bỏ điểm
// trùng và điểm nằm giữa hai đoạn thẳng hàng.
func (l *Layout) resolvePaths() {
	for _, e := range l.Edges {
		u, v := l.items[e.Src], l.items[e.Dst]
		p0 := [2]float64{u.X + float64(e.ExitFrac[0]*u.W), u.Y + float64(e.ExitFrac[1]*u.H)}
		p1 := [2]float64{v.X + float64(e.EntryFrac[0]*v.W), v.Y + float64(e.EntryFrac[1]*v.H)}
		rx := func(r Ref) float64 {
			switch r.Kind {
			case RefCol:
				if r.Lane == v.Lane && r.Col == v.Col {
					return v.X + v.W/2
				}
				c := l.g.colX[[2]int{r.Lane, r.Col}]
				return c[0] + c[1]/2
			case RefSeg:
				return l.trackX(r.Seg)
			}
			return p0[0]
		}
		ry := func(r Ref) float64 {
			if r.Kind == RefSeg {
				return l.trackY(r.Seg)
			}
			return p0[1]
		}
		pts := [][2]float64{p0}
		for _, p := range e.Sym {
			pts = append(pts, [2]float64{rx(p.X), ry(p.Y)})
		}
		pts = append(pts, p1)

		clean := [][2]float64{pts[0]}
		for _, p := range pts[1:] {
			last := clean[len(clean)-1]
			if math.Abs(p[0]-last[0]) < 0.01 && math.Abs(p[1]-last[1]) < 0.01 {
				continue
			}
			clean = append(clean, p)
		}
		out := [][2]float64{clean[0]}
		for i := 1; i < len(clean)-1; i++ {
			a, b, c := out[len(out)-1], clean[i], clean[i+1]
			vertical := math.Abs(a[0]-b[0]) < 0.01 && math.Abs(b[0]-c[0]) < 0.01
			horizontal := math.Abs(a[1]-b[1]) < 0.01 && math.Abs(b[1]-c[1]) < 0.01
			if vertical || horizontal {
				continue
			}
			out = append(out, b)
		}
		// Luôn nối điểm cuối, kể cả khi clean chỉ còn một điểm: khi đó đường có
		// hai điểm trùng nhau, đúng như bản tham chiếu.
		out = append(out, clean[len(clean)-1])
		for i := range out {
			out[i] = [2]float64{num.Round(out[i][0], 2), num.Round(out[i][1], 2)}
		}
		e.Pts = out
	}
}

// horizontalSpans đo bề dài phần ngang đầu tiên của các cạnh B và C có nhãn,
// tức chỗ nhãn sẽ nằm, sau lượt tính đầu.
func (l *Layout) horizontalSpans() map[string]float64 {
	spans := map[string]float64{}
	for _, e := range l.Edges {
		if (e.Case == 'B' || e.Case == 'C') && len(e.Lines) > 0 && len(e.Pts) >= 2 {
			spans[e.ID] = math.Abs(e.Pts[1][0] - e.Pts[0][0])
		}
	}
	return spans
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
