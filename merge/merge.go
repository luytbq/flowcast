package merge

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/luytbq/flowcast/internal/etree"
	"github.com/luytbq/flowcast/layout"
	"github.com/luytbq/flowcast/num"
)

const (
	pad  = 10.0 // minimum margin between a node and the lane edge or another node
	step = 10.0 // shift-down step when a new node overlaps something
)

// constraintKeys are the style keys that anchor wire ends to nodes. Wires kept
// from the old file carry exactly these keys, in this order.
var constraintKeys = [...]string{"exitX", "exitY", "exitDx", "exitDy", "exitPerimeter",
	"entryX", "entryY", "entryDx", "entryDy", "entryPerimeter"}

// isLayer: "0" is the root and "1" is the default layer of every draw.io page.
func isLayer(id string) bool { return id == "0" || id == "1" }

// looksLikeTableID accepts ids following the table convention, such as API-3,
// SVC-2.1 or E7.1. A digit is any Unicode digit, and the id may end with one
// newline.
func looksLikeTableID(s string) bool {
	s = strings.TrimSuffix(s, "\n")
	digitsDots := func(t string) bool {
		groups := strings.Split(t, ".")
		for _, g := range groups {
			if g == "" {
				return false
			}
			for _, r := range g {
				if !unicode.IsDigit(r) {
					return false
				}
			}
		}
		return true
	}
	if strings.HasPrefix(s, "E") && digitsDots(s[1:]) {
		return true
	}
	if s == "" || s[0] < 'A' || s[0] > 'Z' {
		return false
	}
	i := 1
	for i < len(s) && (s[i] >= 'A' && s[i] <= 'Z' || s[i] >= '0' && s[i] <= '9' || s[i] == '_') {
		i++
	}
	return i < len(s) && s[i] == '-' && digitsDots(s[i+1:])
}

type box [4]float64

func itemBox(it *layout.PlacedItem) box { return box{it.X, it.Y, it.X + it.W, it.Y + it.H} }

func overlap(a, b box) bool {
	return fmin(a[2], b[2])-fmax(a[0], b[0]) > 0 && fmin(a[3], b[3])-fmax(a[1], b[1]) > 0
}

// fmax and fmin return the first argument when the two are equal, so the sign
// of zero is not changed.
func fmax(a, b float64) float64 {
	if b > a {
		return b
	}
	return a
}

func fmin(a, b float64) float64 {
	if b < a {
		return b
	}
	return a
}

// fsum sums with rounding-error compensation (Neumaier's algorithm), so the
// error does not accumulate across many lanes.
func fsum(xs []float64) float64 {
	hi, lo := 0.0, 0.0
	for _, x := range xs {
		t := hi + x
		if math.Abs(hi) >= math.Abs(x) {
			lo += (hi - t) + x
		} else {
			lo += (x - t) + hi
		}
		hi = t
	}
	if lo != 0 && !math.IsInf(lo, 0) && !math.IsNaN(lo) {
		return hi + lo
	}
	return hi
}

type movable struct {
	lane  int
	shift func(dx, dy float64)
	// rel: coordinates relative to the lane, moved along when the lane shifts.
	rel bool
}

// span is the x range of an old lane in the pool, with its shift to the new
// position and the new lane that takes over what used to lie inside it.
type span struct {
	x0, x1, dx float64
	target     int
}

type fhGeom struct {
	g     *etree.Element
	frame string // "lane", "pool" or "abs"
	lane  int
}

type merger struct {
	r   *layout.Result
	old *Old
	rep Report

	items    map[string]*layout.PlacedItem
	laneIdx  map[string]int
	ins      map[string][]*layout.PlacedEdge
	outs     map[string][]*layout.PlacedEdge
	tableIDs map[string]bool
	hasMark  bool
	gen      map[string]bool
	ox, oy   float64

	freshC     map[string][2]float64
	freshLaneX []float64
	freshWP    map[string][][2]float64
	freshFrac  map[string][2][2]float64

	spans     []span
	movables  []movable
	final     []*layout.PlacedItem // in the order their positions were fixed
	inFinal   map[string]bool
	pinned    map[string]bool
	fhBoxes   []box
	fhGeoms   []fhGeom
	freehands []string
}

// Apply adjusts the freshly laid out diagram r to the old file old and returns
// the freehand cells to keep. r is modified in place: node positions, lanes,
// wires and pool size.
//
// Merge does not rerun the self-check: wires are either kept from the old file
// or routed by draw.io, so the self-check has nothing to say about them.
func Apply(r *layout.Result, old *Old) ([]*etree.Element, Report) {
	m := &merger{r: r, old: old,
		items: map[string]*layout.PlacedItem{}, laneIdx: map[string]int{},
		ins: map[string][]*layout.PlacedEdge{}, outs: map[string][]*layout.PlacedEdge{},
		tableIDs: map[string]bool{}, gen: map[string]bool{}, inFinal: map[string]bool{}, pinned: map[string]bool{},
		freshC: map[string][2]float64{}, freshWP: map[string][][2]float64{}, freshFrac: map[string][2][2]float64{},
	}
	return m.run(), m.rep
}

func (m *merger) cell(id string) *oldCell { return m.old.cells[id] }

func (m *merger) isGenerated(c *oldCell) bool {
	if isLayer(c.id) {
		return true
	}
	laneLike := c.parent() == "pool" && strings.HasPrefix(c.styleRaw(), "swimlane")
	shaped := c.id == "pool" || m.tableIDs[c.id] || looksLikeTableID(c.id) || laneLike
	if m.hasMark {
		return shaped && c.style().has("flowtable")
	}
	return shaped
}

func (m *merger) run() []*etree.Element {
	r := m.r
	for i := range r.Items {
		it := &r.Items[i]
		m.items[it.ID] = it
		m.tableIDs[it.ID] = true
	}
	for i, ln := range r.Lanes {
		m.laneIdx[ln.ID] = i
		m.tableIDs[ln.ID] = true
	}
	for i := range r.Edges {
		e := &r.Edges[i]
		m.tableIDs[e.ID] = true
		m.outs[e.Src] = append(m.outs[e.Src], e)
		m.ins[e.Dst] = append(m.ins[e.Dst], e)
	}
	for _, id := range m.old.ids {
		if m.cell(id).style().has("flowtable") {
			m.hasMark = true
		}
	}
	for _, id := range m.old.ids {
		if m.isGenerated(m.cell(id)) {
			m.gen[id] = true
		}
	}
	if pool := m.cell("pool"); pool != nil && pool.vertex() && m.gen["pool"] {
		b := m.old.absBox("pool")
		r.Origin = [2]float64{b[0], b[1]}
	}
	m.ox, m.oy = r.Origin[0], r.Origin[1]

	// Snapshot of the freshly computed layout, used as the offset for new nodes.
	for i := range r.Items {
		it := &r.Items[i]
		m.freshC[it.ID] = [2]float64{it.X + it.W/2, it.Y + it.H/2}
	}
	m.freshLaneX = append([]float64(nil), r.LaneX...)
	for _, e := range r.Edges {
		var wp [][2]float64
		if len(e.Pts) >= 2 {
			wp = e.Pts[1 : len(e.Pts)-1]
		}
		m.freshWP[e.ID] = wp
		m.freshFrac[e.ID] = [2][2]float64{e.ExitFrac, e.EntryFrac}
	}
	for _, id := range m.old.ids {
		if !m.gen[id] {
			m.freehands = append(m.freehands, id)
		}
	}
	sort.SliceStable(m.freehands, func(i, j int) bool {
		return m.cell(m.freehands[i]).order < m.cell(m.freehands[j]).order
	})

	m.layoutLanes()
	m.placeOldItems()
	m.placeNewItems()
	m.routeEdges()
	extras := m.collectFreehand()
	m.growLanes()
	m.fitVertical()
	m.finish()
	var gen []string
	for id := range m.gen {
		if !isLayer(id) && id != "pool" && !m.tableIDs[id] {
			gen = append(gen, id)
		}
	}
	sort.Strings(gen)
	m.rep.Removed = append(m.rep.Removed, gen...)
	return extras
}

// origin is the absolute coordinate origin of the child frame of cell cid: the
// coordinates of nested vertices add up.
func (o *Old) origin(cid string) (float64, float64) {
	x, y := 0.0, 0.0
	seen := map[string]bool{}
	for {
		c, ok := o.cells[cid]
		if !ok || isLayer(cid) || seen[cid] {
			break
		}
		seen[cid] = true
		if !c.vertex() {
			break
		}
		x += attrf(c.geo(), "x")
		y += attrf(c.geo(), "y")
		cid = c.parent()
	}
	return x, y
}

func (o *Old) absBox(cid string) [4]float64 {
	c := o.cells[cid]
	px, py := o.origin(c.parent())
	g := c.geo()
	return [4]float64{px + attrf(g, "x"), py + attrf(g, "y"), attrf(g, "width"), attrf(g, "height")}
}

// ---- lane

func (m *merger) oldLane(cid string) *oldCell {
	c := m.cell(cid)
	if c == nil || !c.vertex() || !m.gen[cid] || c.parent() != "pool" {
		return nil
	}
	return c
}

func (m *merger) layoutLanes() {
	r := m.r
	var newX, newW []float64
	x := 0.0
	for i, ln := range r.Lanes {
		w := r.LaneW[i]
		// The file only stores widths rounded to two decimals. An old width that
		// matches the new width after rounding means the lane has no manual edits;
		// taking back the unrounded number keeps the new pool width total equal
		// to what was generated.
		if oc := m.oldLane(ln.ID); oc != nil {
			if ow := attrf(oc.geo(), "width"); num.Fmt(ow) != num.Fmt(w) {
				w = ow
			}
		}
		newX = append(newX, x)
		newW = append(newW, w)
		x += w
	}
	type oldSpan struct {
		x, w float64
		id   string
	}
	var olds []oldSpan
	for _, id := range m.old.ids {
		c := m.cell(id)
		if c.vertex() && c.parent() == "pool" && m.gen[id] && strings.HasPrefix(c.styleRaw(), "swimlane") {
			olds = append(olds, oldSpan{attrf(c.geo(), "x"), attrf(c.geo(), "width"), id})
		}
	}
	sort.SliceStable(olds, func(i, j int) bool {
		a, b := olds[i], olds[j]
		if a.x != b.x {
			return a.x < b.x
		}
		if a.w != b.w {
			return a.w < b.w
		}
		return a.id < b.id
	})
	for k, o := range olds {
		target, ok := m.laneIdx[o.id]
		base := 0.0
		if ok {
			base = newX[target]
		} else {
			// Lane was removed: what lay inside it follows the next remaining lane,
			// or the last lane if no lane remains after it.
			target, base = len(r.Lanes)-1, x
			for _, o2 := range olds[k+1:] {
				if t, ok := m.laneIdx[o2.id]; ok {
					target, base = t, newX[t]
					break
				}
			}
		}
		m.spans = append(m.spans, span{o.x, o.x + o.w, base - o.x, target})
	}
	r.LaneX, r.LaneW = newX, newW
}

// mapX returns the shift and the new lane for an old x coordinate in the pool.
func (m *merger) mapX(x float64) (float64, int) {
	if len(m.spans) == 0 {
		return 0, 0
	}
	for _, s := range m.spans {
		if s.x0 <= x && x < s.x1 {
			return s.dx, s.target
		}
	}
	if x < m.spans[0].x0 {
		return m.spans[0].dx, m.spans[0].target
	}
	last := m.spans[len(m.spans)-1]
	return last.dx, last.target
}

// ---- node

func (m *merger) addMovable(lane int, shift func(dx, dy float64), rel bool) {
	m.movables = append(m.movables, movable{lane, shift, rel})
}

func (m *merger) fix(it *layout.PlacedItem) {
	m.final = append(m.final, it)
	m.inFinal[it.ID] = true
	m.addMovable(it.Lane, func(dx, dy float64) {
		it.X += dx
		it.Y += dy
	}, false)
}

func (m *merger) placeOldItems() {
	r := m.r
	for i := range r.Items {
		it := &r.Items[i]
		c := m.cell(it.ID)
		if c == nil || !c.vertex() || !m.gen[it.ID] {
			continue
		}
		laneID := m.laneCell(it.Lane)
		if c.parent() != laneID {
			m.rep.LaneChanged = append(m.rep.LaneChanged, it.ID)
			continue
		}
		b := m.old.absBox(it.ID)
		cx, cy := b[0]+b[2]/2-m.ox, b[1]+b[3]/2-m.oy
		dx := 0.0
		if oc := m.oldLane(laneID); oc != nil {
			dx = r.LaneX[it.Lane] - attrf(oc.geo(), "x")
		}
		it.X, it.Y = cx+dx-it.W/2, cy-it.H/2
		m.fix(it)
		m.pinned[it.ID] = true
		m.rep.Pinned = append(m.rep.Pinned, it.ID)
	}
}

// laneCell is the id of the parent cell holding the nodes of lane i in the
// file. A diagram without lanes puts nodes directly on layer "1", so the hidden
// lane maps to that layer.
func (m *merger) laneCell(i int) string {
	if m.r.NoLanes {
		return "1"
	}
	return m.r.Lanes[i].ID
}

func (m *merger) neighbors(id string) []string {
	var out []string
	for _, back := range []bool{false, true} {
		for _, e := range m.ins[id] {
			if e.Back == back {
				out = append(out, e.Src)
			}
		}
		for _, e := range m.outs[id] {
			if e.Back == back {
				out = append(out, e.Dst)
			}
		}
	}
	return out
}

func byBackOrder(es []*layout.PlacedEdge) []*layout.PlacedEdge {
	out := append([]*layout.PlacedEdge(nil), es...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Back != out[j].Back {
			return !out[i].Back
		}
		return out[i].Order < out[j].Order
	})
	return out
}

// findAnchor finds the fixed-position node closest to v in the graph: sources
// of incoming edges first, then targets of outgoing edges, then widening ring
// by ring.
func (m *merger) findAnchor(v *layout.PlacedItem) (string, bool) {
	for _, e := range byBackOrder(m.ins[v.ID]) {
		if m.inFinal[e.Src] {
			return e.Src, true
		}
	}
	for _, e := range byBackOrder(m.outs[v.ID]) {
		if m.inFinal[e.Dst] {
			return e.Dst, true
		}
	}
	seen := map[string]bool{v.ID: true}
	frontier := []string{v.ID}
	for len(frontier) > 0 {
		var nxt []string
		for _, n := range frontier {
			for _, id := range m.neighbors(n) {
				if seen[id] {
					continue
				}
				if m.inFinal[id] {
					return id, true
				}
				seen[id] = true
				nxt = append(nxt, id)
			}
		}
		sort.SliceStable(nxt, func(i, j int) bool { return m.items[nxt[i]].Order < m.items[nxt[j]].Order })
		frontier = nxt
	}
	return "", false
}

func (m *merger) obstacles(skip string) []box {
	var out []box
	for _, it := range m.final {
		if it.ID != skip {
			out = append(out, itemBox(it))
		}
	}
	return append(out, m.fhBoxes...)
}

func (m *merger) put(it *layout.PlacedItem, cx, cy float64) {
	it.X, it.Y = cx-it.W/2, cy-it.H/2
	obs := m.obstacles(it.ID)
	hit := func() bool {
		b := box{it.X - pad, it.Y - pad, it.X + it.W + pad, it.Y + it.H + pad}
		for _, o := range obs {
			if overlap(b, o) {
				return true
			}
		}
		return false
	}
	moved := false
	for hit() {
		it.Y += step
		moved = true
	}
	if moved {
		m.rep.Shifted = append(m.rep.Shifted, it.ID)
	}
	m.fix(it)
	m.rep.Placed = append(m.rep.Placed, it.ID)
}

// laneRelX keeps the node's offset from the lane's left edge as the fresh
// layout computed it, but pulls it inside the lane if the lane is wide enough.
func (m *merger) laneRelX(it *layout.PlacedItem) float64 {
	r := m.r
	fx := m.freshC[it.ID][0] - m.freshLaneX[it.Lane]
	lx, lw := r.LaneX[it.Lane], r.LaneW[it.Lane]
	cx := lx + fx
	if lw >= it.W+2*pad {
		cx = fmin(fmax(cx, lx+it.W/2+pad), lx+lw-it.W/2-pad)
	}
	return cx
}

func (m *merger) placeNewItems() {
	r := m.r
	m.fhBoxes = m.freehandVertexBoxes()
	topo := map[string]int{}
	for k, id := range r.TopoOrder {
		topo[id] = k
	}
	rank := func(id string) int {
		if k, ok := topo[id]; ok {
			return k
		}
		return math.MaxInt
	}
	var flow, atts []*layout.PlacedItem
	for i := range r.Items {
		it := &r.Items[i]
		if m.inFinal[it.ID] {
			continue
		}
		if it.Attach == "" {
			flow = append(flow, it)
		} else {
			atts = append(atts, it)
		}
	}
	sort.SliceStable(flow, func(i, j int) bool {
		a, b := rank(flow[i].ID), rank(flow[j].ID)
		if a != b {
			return a < b
		}
		return flow[i].Order < flow[j].Order
	})
	sort.SliceStable(atts, func(i, j int) bool { return atts[i].Order < atts[j].Order })
	for _, it := range flow {
		a, ok := m.findAnchor(it)
		if !ok {
			bottom := float64(r.PoolHeader + r.LaneHeader)
			for k, b := range m.obstacles(it.ID) {
				if k == 0 {
					bottom = b[3]
				} else {
					bottom = fmax(bottom, b[3])
				}
			}
			m.put(it, m.laneRelX(it), bottom+float64(r.MinChannel)+it.H/2)
			continue
		}
		m.putRelative(it, a)
	}
	for _, it := range atts {
		m.putRelative(it, it.Attach)
	}
}

func (m *merger) putRelative(it *layout.PlacedItem, a string) {
	an := m.items[a]
	fa, fv := m.freshC[a], m.freshC[it.ID]
	ax, ay := an.X+an.W/2, an.Y+an.H/2
	cx := m.laneRelX(it)
	if an.Lane == it.Lane {
		cx = ax + fv[0] - fa[0]
	}
	m.put(it, cx, ay+fv[1]-fa[1])
}

// ---- wires

func (m *merger) routeEdges() {
	for i := range m.r.Edges {
		e := &m.r.Edges[i]
		c := m.cell(e.ID)
		keep := c != nil && c.edge() && m.gen[e.ID]
		if keep {
			src, sok := c.cell.Get("source")
			dst, dok := c.cell.Get("target")
			keep = sok && dok && src == e.Src && dst == e.Dst && m.pinned[e.Src] && m.pinned[e.Dst]
		}
		if !keep {
			e.Auto = true
			m.rep.Rerouted = append(m.rep.Rerouted, e.ID)
			continue
		}
		px, py := m.old.origin(c.parent())
		var lanes []int
		e.Waypoints = nil
		for _, p := range c.points() {
			ax, ay := px+p[0]-m.ox, py+p[1]-m.oy
			dx, lane := m.mapX(ax)
			e.Waypoints = append(e.Waypoints, [2]float64{ax + dx, ay})
			lanes = append(lanes, lane)
		}
		st := c.style()
		e.Constraints = nil
		for _, k := range constraintKeys {
			if v, ok := st.vals[k]; ok && v.set {
				e.Constraints = append(e.Constraints, [2]string{k, v.v})
			}
		}
		e.Kept = true
		m.rep.KeptEdges = append(m.rep.KeptEdges, e.ID)
		if !m.untouched(e) {
			e.NoLabelPos = true
			if len(e.Lines) > 0 {
				m.rep.LabelsCentered = append(m.rep.LabelsCentered, e.ID)
			}
		}
		for k, lane := range lanes {
			k := k
			m.addMovable(lane, func(dx, dy float64) {
				e.Waypoints[k][0] += dx
				e.Waypoints[k][1] += dy
			}, false)
		}
	}
}

// untouched: the wire is still exactly as the fresh layout computed it, i.e.
// the user has not edited it. The computed label position then still holds.
func (m *merger) untouched(e *layout.PlacedEdge) bool {
	fresh := m.freshWP[e.ID]
	if len(fresh) != len(e.Waypoints) {
		return false
	}
	for k, a := range fresh {
		b := e.Waypoints[k]
		if math.Abs(a[0]-b[0]) > 2 || math.Abs(a[1]-b[1]) > 2 {
			return false
		}
	}
	fr := m.freshFrac[e.ID]
	want := []struct {
		k string
		v float64
	}{{"exitX", fr[0][0]}, {"exitY", fr[0][1]}, {"entryX", fr[1][0]}, {"entryY", fr[1][1]}}
	for _, w := range want {
		s, ok := "", false
		for _, kv := range e.Constraints {
			if kv[0] == w.k {
				s, ok = kv[1], true
			}
		}
		if !ok {
			return false
		}
		// The file stores ratios rounded to two decimals, so compare after
		// rounding exactly as on write. A ratio like 1/3 differs by 0.0033 from
		// the number read back.
		v, ok := parseFloat(s)
		if !ok || num.Fmt(v) != num.Fmt(w.v) {
			return false
		}
	}
	return true
}

// ---- freehand cells

// frameOf returns the coordinate frame of a cell: "lane:<id>", "pool", "abs" or
// "nested".
func (m *merger) frameOf(c *oldCell) string {
	p := c.parent()
	switch {
	case isLayer(p):
		return "abs"
	case p == "pool" && m.gen["pool"]:
		return "pool"
	case m.gen[p] && m.oldLane(p) != nil:
		return "lane:" + p
	}
	return "nested"
}

func (m *merger) freehandVertexBoxes() []box {
	var out []box
	for _, id := range m.freehands {
		c := m.cell(id)
		if !c.vertex() {
			continue
		}
		b := m.old.absBox(id)
		x, y := b[0]-m.ox, b[1]-m.oy
		fr := m.frameOf(c)
		if lid, ok := strings.CutPrefix(fr, "lane:"); ok {
			if li, ok := m.laneIdx[lid]; ok {
				x += m.r.LaneX[li] - attrf(m.oldLane(lid).geo(), "x")
			}
		} else if fr == "pool" || fr == "abs" {
			dx, _ := m.mapX(x + b[2]/2)
			x += dx
		}
		out = append(out, box{x, y, x + b[2], y + b[3]})
	}
	return out
}

func (m *merger) collectFreehand() []*etree.Element {
	alive := map[string]bool{"pool": true, "0": true, "1": true}
	for id := range m.tableIDs {
		alive[id] = true
	}
	kept := map[string]bool{}
	for _, id := range m.freehands {
		p := m.cell(id).parent()
		var ok bool
		if m.gen[p] && !alive[p] {
			ok = m.oldLane(p) != nil
		} else {
			ok = m.gen[p] || isLayer(p) || kept[p]
		}
		if ok {
			kept[id] = true
		} else {
			m.rep.FreehandDropped = append(m.rep.FreehandDropped, id)
		}
	}
	var extras []*etree.Element
	for _, id := range m.freehands {
		if !kept[id] {
			continue
		}
		c := m.cell(id)
		p := c.parent()
		el := c.elem.Copy()
		cell := el
		if el.Tag != "mxCell" {
			cell = el.Find("mxCell")
		}
		g := cell.Find("mxGeometry")
		fr := m.frameOf(c)
		if lid, ok := strings.CutPrefix(fr, "lane:"); ok {
			if _, alive := m.laneIdx[lid]; !alive {
				// Lane was removed: move the cell up to the pool, keeping its absolute position.
				ax, ay := m.old.origin(p)
				cell.Set("parent", "pool")
				if g != nil && c.vertex() {
					g.Set("x", num.Fmt(attrf(g, "x")+ax-m.ox))
					g.Set("y", num.Fmt(attrf(g, "y")+ay-m.oy))
				}
				shiftPoints(g, ax-m.ox, ay-m.oy)
				fr = "fixed"
				if c.vertex() && g != nil {
					m.fhGeoms = append(m.fhGeoms, fhGeom{g, "pool", -1})
				}
			}
		}
		if c.edge() {
			m.detach(c, cell, g, alive, kept)
		}
		m.track(c, g, fr)
		extras = append(extras, el)
		m.rep.FreehandKept = append(m.rep.FreehandKept, id)
	}
	return extras
}

func eachPoint(g *etree.Element, f func(pt *etree.Element)) {
	g.Iter("mxPoint", func(pt *etree.Element) {
		if as, _ := pt.Get("as"); as != "offset" {
			f(pt)
		}
	})
}

func movePoint(pt *etree.Element, dx, dy float64) {
	pt.Set("x", num.Fmt(attrf(pt, "x")+dx))
	pt.Set("y", num.Fmt(attrf(pt, "y")+dy))
}

func shiftPoints(g *etree.Element, dx, dy float64) {
	if g == nil {
		return
	}
	eachPoint(g, func(pt *etree.Element) { movePoint(pt, dx, dy) })
}

// shiftGeo moves a geometry. A relative geometry belongs to a wire: its position
// lives in the points, not in x and y.
func shiftGeo(g *etree.Element, dx, dy float64) {
	if v, _ := g.Get("relative"); v == "1" {
		shiftPoints(g, dx, dy)
		return
	}
	g.Set("x", num.Fmt(attrf(g, "x")+dx))
	g.Set("y", num.Fmt(attrf(g, "y")+dy))
}

// detach unhooks a freehand wire end from a node no longer in the diagram,
// replacing it with a free point at that node's old center.
func (m *merger) detach(c *oldCell, cell, g *etree.Element, alive, kept map[string]bool) {
	for _, end := range [][2]string{{"source", "sourcePoint"}, {"target", "targetPoint"}} {
		ref, _ := cell.Get(end[0])
		if ref == "" || kept[ref] || (alive[ref] && m.gen[ref]) {
			continue
		}
		if m.cell(ref) == nil {
			continue
		}
		b := m.old.absBox(ref)
		px, py := m.old.origin(c.parent())
		cell.Del(end[0])
		if g == nil {
			g = cell.Add("mxGeometry", "relative", "1", "as", "geometry")
		}
		for _, pt := range g.FindAll("mxPoint") {
			if as, _ := pt.Get("as"); as == end[1] {
				g.Remove(pt)
			}
		}
		g.Add("mxPoint", "x", num.Fmt(b[0]+b[2]/2-px), "y", num.Fmt(b[1]+b[3]/2-py), "as", end[1])
		m.rep.FreehandDetached = append(m.rep.FreehandDetached, c.id+"."+end[0])
	}
}

// track makes a freehand cell move with its lane when the lane shifts or is
// widened.
func (m *merger) track(c *oldCell, g *etree.Element, fr string) {
	if g == nil || fr == "nested" || fr == "fixed" {
		return
	}
	if lid, ok := strings.CutPrefix(fr, "lane:"); ok {
		lane := m.laneIdx[lid]
		m.addMovable(lane, func(dx, dy float64) { shiftGeo(g, dx, dy) }, true)
		m.fhGeoms = append(m.fhGeoms, fhGeom{g, "lane", lane})
		return
	}
	ox := 0.0
	if fr == "abs" {
		ox = m.ox
	}
	if c.vertex() {
		dx, lane := m.mapX(attrf(g, "x") - ox + attrf(g, "width")/2)
		shiftGeo(g, dx, 0)
		m.addMovable(lane, func(dx, dy float64) { shiftGeo(g, dx, dy) }, false)
		m.fhGeoms = append(m.fhGeoms, fhGeom{g, fr, lane})
		return
	}
	eachPoint(g, func(pt *etree.Element) {
		dx, lane := m.mapX(attrf(pt, "x") - ox)
		pt.Set("x", num.Fmt(attrf(pt, "x")+dx))
		m.addMovable(lane, func(dx, dy float64) { movePoint(pt, dx, dy) }, false)
	})
}

// ---- lane widening, pool resizing

func (m *merger) growLanes() {
	r := m.r
	for i := range r.Lanes {
		first := true
		var left, right float64
		for _, it := range m.final {
			if it.Lane != i {
				continue
			}
			if first {
				left, right, first = it.X, it.X+it.W, false
				continue
			}
			left, right = fmin(left, it.X), fmax(right, it.X+it.W)
		}
		if first {
			continue
		}
		lx, lw := r.LaneX[i], r.LaneW[i]
		left, right = left-lx, right-lx
		gl := fmax(0.0, pad-left)
		gr := fmax(0.0, right-(lw-pad))
		if gl == 0 && gr == 0 {
			continue
		}
		m.rep.LanesGrown = append(m.rep.LanesGrown, r.Lanes[i].ID+" +"+num.Fmt(gl+gr)+"px")
		r.LaneW[i] += gl + gr
		for j := i + 1; j < len(r.Lanes); j++ {
			r.LaneX[j] += gl + gr
		}
		for _, mv := range m.movables {
			if mv.lane == i && gl != 0 {
				mv.shift(gl, 0)
			} else if mv.lane > i && !mv.rel {
				mv.shift(gl+gr, 0)
			}
		}
	}
}

// fitVertical pushes the whole diagram down when a node or waypoint has been
// dragged above the top of the lanes.
func (m *merger) fitVertical() {
	r := m.r
	top := float64(r.PoolHeader + r.LaneHeader + pad)
	var ys []float64
	for _, it := range m.final {
		ys = append(ys, it.Y)
	}
	for _, e := range r.Edges {
		for _, p := range e.Waypoints {
			ys = append(ys, p[1])
		}
	}
	if len(ys) == 0 {
		return
	}
	low := ys[0]
	for _, y := range ys[1:] {
		low = fmin(low, y)
	}
	if low < top {
		dy := top - low
		for _, mv := range m.movables {
			mv.shift(0, dy)
		}
	}
}

func (m *merger) finish() {
	r := m.r
	var bottoms []float64
	for _, it := range m.final {
		bottoms = append(bottoms, it.Y+it.H)
	}
	for _, e := range r.Edges {
		for _, p := range e.Waypoints {
			bottoms = append(bottoms, p[1])
		}
	}
	for _, b := range m.freehandBoxesNow() {
		bottoms = append(bottoms, b[3])
	}
	// A diagram with no elements gets only one minimum channel below the header
	// in the fresh layout; merge keeps exactly that height so the file does not
	// change when the table does not change.
	h := float64(r.PoolHeader + r.LaneHeader + r.MinChannel)
	for _, b := range bottoms {
		h = fmax(h, b+float64(r.MinChannel))
	}
	r.PoolH = h
	r.PoolW = fsum(r.LaneW)
	items := append([]*layout.PlacedItem(nil), m.final...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].Order < items[j].Order })
	for k, a := range items {
		for _, b := range items[k+1:] {
			if overlap(itemBox(a), itemBox(b)) {
				m.rep.Overlaps = append(m.rep.Overlaps, a.ID+"/"+b.ID)
			}
		}
	}
}

func (m *merger) freehandBoxesNow() []box {
	var out []box
	for _, f := range m.fhGeoms {
		x, y, w, h := attrf(f.g, "x"), attrf(f.g, "y"), attrf(f.g, "width"), attrf(f.g, "height")
		switch f.frame {
		case "lane":
			x, y = x+m.r.LaneX[f.lane], y+float64(m.r.PoolHeader)
		case "abs":
			x, y = x-m.ox, y-m.oy
		}
		out = append(out, box{x, y, x + w, y + h})
	}
	return out
}
