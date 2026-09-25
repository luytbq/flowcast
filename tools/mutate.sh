#!/bin/sh
# Checks whether the conformance suite really catches bugs, by deliberately
# breaking one spot at a time and seeing whether the tests go red.
#
#     sh tools/mutate.sh
#     ONLY=writer/ sh tools/mutate.sh    # only mutate files whose path starts with writer/
#
# A conformance suite that looks massive but catches no mutation pins nothing,
# and that stays silent until the port is actually wrong. Rerun after adding
# every new Go module.
#
# Runs in two passes. The first only checks that every target string still
# matches exactly once, so that a mutation gone stale after a code edit stops
# the whole run at the start instead of halfway through, after test time is
# spent.
#
# Target strings should avoid alignment whitespace: gofmt realigns columns
# whenever a struct gains a longer field, and the mutation stops matching.
#
# Restore with git, not with a reverse sed: a reverse sed breaks when the
# replacement contains the replaced string, and when the target occurs in
# several places.
set -e
cd "$(dirname "$0")/.."

if [ -z "$PREFLIGHT" ]; then
	if ! git diff --quiet || ! git diff --cached --quiet; then
		echo 'ERROR working tree is dirty; commit or stash before running'
		exit 1
	fi
	PREFLIGHT=1 sh "$0" || exit 1
	echo 'every target string still matches; starting mutations'
	echo
fi

replace=$(mktemp -d)/replaceonce
go build -o "$replace" ./tools/replaceonce

caught=0
missed=0

# If interrupted while a file carries a mutation, restore it, otherwise the
# working tree is left with broken code that tests may still pass.
current=''
trap '[ -n "$current" ] && git checkout -- "$current"' EXIT INT TERM

mutate() {
	label="$1" file="$2" from="$3" to="$4"
	# Filter before touching the file. If a filtered-out mutation were still
	# applied, the function would return early leaving a broken file that the
	# trap does not know to restore.
	if [ -z "$PREFLIGHT" ]; then
		case "$file" in "${ONLY:-}"*) ;; *) return 0 ;; esac
		current="$file"
	fi
	if [ -n "$PREFLIGHT" ]; then
		"$replace" -check "$file" "$from" "$to"
		return 0
	fi
	"$replace" "$file" "$from" "$to"
	if go test -count=1 ./... >/dev/null 2>&1; then
		echo "MISSED     $label"
		missed=$((missed + 1))
	else
		echo "CAUGHT     $label"
		caught=$((caught + 1))
	fi
	git checkout -- "$file"
	current=''
}

# Proven equivalent mutations, not added back because they always survive:
#
# - Reversing branchOffset. It was only used to count edges with a positive
#   offset, and the offsets are a permutation of 0 to n-1, so the count is
#   always n-1. The function was removed.
# - Dropping the check on the target's receiving side in type B. When phase B
#   runs, the target's left or right side can only be taken by an earlier B edge
#   joining the target to a node x. If x is on the same side as the source in
#   the same row, either x lies between and blocks the source's path, or the
#   source lies between and x's edge cannot be B. If x is the source, the
#   source-side check catches it first.
# - Dropping the one-edge-per-side shortcut in assignPorts. With exactly one
#   edge, every branch yields the middle of the side.
# - The center of a column other than the target column in resolvePaths. route
#   only emits references to the target's own column, so that branch never runs.
#   Kept because it makes rx correct for every column reference, not only for
#   what route emits today.
# - Stopping at a zero-cost label candidate. Candidates are chosen with a strict
#   < comparison, so a later zero-cost candidate can never replace an earlier
#   zero-cost one; the break is only an optimization.
# - fmax returning the second number when the two are equal. It only differs for
#   negative zero and positive zero, and negative zero cannot arise in the
#   geometry phase: every coordinate is a sum of positive widths starting at 0,
#   x - x is always positive zero, and the only negative number is d = -1 times
#   a nonzero positive amount.
#
# - A completely empty line in readCSV not yielding an empty row. splitKeepEnds
#   never yields an empty line: every line carries at least its own newline, so
#   the state machine always sees the newline before the end of the line.
# - .text of an xlsx element collecting text after child elements too. The
#   reader only reads the text of t and v, which never have children in valid
#   SpreadsheetML.
# - Idx of a markdown row taking the line number in the file instead of the
#   table row order. Both increase with row order, and Idx only reaches the
#   engine as Order, where every use only compares two Orders. The writer does
#   not write Order.
# - itemsFree dropping the item-over-label check. room already treats every
#   label in the row band as an obstacle when computing the shift range, and
#   every shift step (full range, halving, alignment) stays within that range.
#   The check is kept as a safety net for future moves.
#
# Each one survived tens of thousands of random tables, or has an argument based
# on the structure of the input, before being proven. Reasoning without data is
# not trustworthy: two other branches were once suspected redundant and then
# killed by random tables.
#
# Surviving mutations NOT yet proven equivalent. Unlike the list above, these
# are places where the conformance suite may have a gap, just no case found yet.
#
# - A wire that only touches the left or right edge of a label box counting as a
#   crossing. Survived 133,141 valid tables. Structurally it cannot happen: a
#   vertical label box sits exactly 5 pixels from its wire, a horizontal label
#   box 6 from the segment start, parallel vertical wires are 12 apart and at
#   least 15 from the column edge. Beyond that only coincidence remains, and
#   text widths are fractional. The top/bottom direction has a pinning case.
# - The fallback path of db and text starting at the second column instead of
#   the third. Survived 49,000 valid tables. It only differs when the cell two
#   columns away lies empty on the path of a horizontal arrow while the adjacent
#   cells on both sides and the cell two columns away on the other side are all
#   taken; even then the grid only numbers columns that hold items, so columns
#   +2 and +3 land at the same position unless another item sits in column +3 in
#   the lane.

# num, text, layout/size
mutate "drop - from break points" text/measure.go '=&?-"' '=&?"'
mutate "drop the :: branch from break points" text/measure.go "return i >= 2 && r[i-1] == ':' && r[i-2] == ':'" 'return false'
mutate "unknown codepoint measures 0 instead of notdef" text/metrics.go 'total += m.Notdef' 'total += 0'
mutate "drop binary narrowing in Wrap" text/measure.go 'out = append(out, t.wrapLine(line, float64(hi))...)' 'out = append(out, first...)'
mutate "hard flag not set on a hard cut" text/measure.go 't.hard = true' '_ = 0'
mutate "task wrap budget off by 2px" layout/shape.go 'cfg.TaskMaxW - 32' 'cfg.TaskMaxW - 30'
mutate "note budget uses DBWrap" layout/shape.go 'return float64(cfg.TextWrap)' 'return float64(cfg.DBWrap)'
mutate "diamond wrap budget off by 10px" layout/shape.go 'return float64(cfg.CondWrap)' 'return float64(cfg.CondWrap) + 10'
mutate "Fmt rounds to 3 digits instead of 2" num/num.go "'f', 2, 64" "'f', 3, 64"
mutate "Rnd rounds down instead of up" num/num.go 'math.Ceil(v/2.0)' 'math.Floor(v/2.0)'

# source/markdown
mutate "drop NFC normalization" source/text.go 'return norm.NFC.String(b.String())' 'return b.String()'
mutate "escaped pipe still splits cells" source/text.go "if c == '|' && !prevEscape {" "if c == '|' {"
mutate "markdown escapes not removed" source/text.go 'if r[i] == 0x5c && i+1 < len(r) && strings.ContainsRune(mdEscapable, r[i+1]) {' 'if false {'
mutate "CR not treated as a line boundary" source/text.go "case 0x0d:" "case 0x2400:"
mutate "type column not lowercased" source/markdown.go 'unistr.Lower(cells[1])' 'cells[1]'
mutate "metadata keys also unescaped" source/markdown.go 'key := unistr.Strip(k)' 'key := unesc(unistr.Strip(k))'
mutate "br tag case sensitive" source/text.go "(r[1] != 'b' && r[1] != 'B')" "r[1] != 'b'"
mutate "level-two heading also counts as title" source/markdown.go 'if len(rest) < 2 || !unistr.IsSpace(rest[0]) {' 'if len(rest) < 2 {'
mutate "metadata key order follows map instead of cells" source/markdown.go 'if _, seen := meta[key]; !seen {' 'if _, seen := meta[key]; seen {'

# schema and validate
mutate "swap the from and to check order" schema/schema.go '{Key: "from", Required: true, Note: edgeNote},
			{Key: "to", Required: true, Note: edgeNote},' '{Key: "to", Required: true, Note: edgeNote},
			{Key: "from", Required: true, Note: edgeNote},'
mutate "attach becomes required" schema/schema.go '	Refs:        []Ref{{Key: "attach", Note: attachNote, SameLane: true}},' '	Refs:        []Ref{{Key: "attach", Required: true, Note: attachNote, SameLane: true}},'
mutate "start gets no warning for incoming edges" schema/elements.go '"start":     {AlwaysInFlow: true, WarnInEdge: true},' '"start":     {AlwaysInFlow: true},'
mutate "attached db counts as part of the flow" schema/schema.go 'return e.AlwaysInFlow || (e.InFlowUnattached && meta["attach"] == "")' 'return e.AlwaysInFlow || e.InFlowUnattached'
mutate "db without attach is not part of the flow" schema/schema.go 'return e.AlwaysInFlow || (e.InFlowUnattached && meta["attach"] == "")' 'return e.AlwaysInFlow'
mutate "edge rejects style dashed" schema/schema.go 'StyleValues: []string{"highlight", "dashed", "bold", "noarrow"}' 'StyleValues: []string{"highlight", "bold", "noarrow"}'
mutate "duplicate id lets the later row overwrite the earlier" validate/validate.go 'if j, dup := v.byID[r.ID]; dup {' 'if j, dup := v.byID[r.ID]; false {'
mutate "rest-of-table marker row also needs parent" validate/validate.go 'case !schema.IsRestMarker(r):' 'case true:'
mutate "styleList keeps duplicate values" validate/validate.go 'if !dup {' 'if true {'
mutate "location taken from first row instead of last" validate/validate.go 'if r.ID != "" {
			pos[r.ID] = i
		}' 'if _, seen := pos[r.ID]; r.ID != "" && !seen {
			pos[r.ID] = i
		}'
mutate "db and text not skipped when checking the next edge" validate/validate.go 'for p < len(v.rows) && schema.AttachTypes[v.rows[p].Type] && v.rows[p].Meta["attach"] == r.ID {' 'for false {'
mutate "condition needs only one outgoing edge" schema/elements.go 'MinOutEdges: 2,' 'MinOutEdges: 1,'
mutate "minimum outgoing edges not enforced" validate/validate.go 'if len(es) < el.MinOutEdges {' 'if false {'
mutate "no warning for condition edge without label" validate/validate.go 'if e.Text() == "" {' 'if false {'
mutate "duplicate-id rows not skipped in graph checks" validate/validate.go 'if j, ok := v.byID[r.ID]; !ok || j != i {' 'if j, ok := v.byID[r.ID]; false || j == -1 && !ok {'
mutate "message uses %q instead of straight quotes" validate/validate.go 'fmt.Sprintf("metadata \"%s\" does not apply to type %s, ignored", k, r.Type)' 'fmt.Sprintf("metadata %q does not apply to type %s, ignored", k, r.Type)'

# Static FMA check
mutate "drop float64() wrapper from multiply before add" layout/shape.go 'float64(tw*1.42)+24+pad' 'tw*1.42+24+pad'

# layout/place
mutate "topo prefers later rows instead of earlier" layout/topo.go 'return h[i].Order < h[j].Order' 'return h[i].Order > h[j].Order'
mutate "topo does not skip back edges" layout/topo.go 'if _, ok := indeg[e.Dst]; ok && !e.Back {' 'if _, ok := indeg[e.Dst]; ok {'
mutate "branchDrift reversed" layout/branch.go 'if w.Lane < v.Lane {' 'if w.Lane > v.Lane {'
mutate "branchDrift does not stop at merge node" layout/branch.go 'if l.nonBackIn(w.ID) > 1 {' 'if false {'
mutate "branch with unclear direction prefers left" layout/branch.go 'if next[1] <= next[-1] {' 'if next[1] < next[-1] {'
mutate "skip sides that already have a horizontal arrow" layout/branch.go "onlyLeft := busy['R'] && !busy['L']" 'onlyLeft := false'
mutate "mergeCol picks the farthest branching node" layout/branch.go 'if best == nil || it.Row > best.Row' 'if best == nil || it.Row < best.Row'
mutate "mergeCol disabled" layout/place.go 'if mc, ok := l.mergeCol(same, v); ok && (' 'if mc, ok := l.mergeCol(same, v); false && ok && ('
mutate "attachSide defaults to left" layout/branch.go 'if !right {' 'if right {'
mutate "condition needs three side branches to keep both sides" layout/place.go 'l.sideBranches(v) >= 2' 'l.sideBranches(v) >= 3'
mutate "horizontal arrows disabled" layout/place.go 'if len(preds) == 1 && l.nonBackIn(vid) == 1' 'if false && len(preds) == 1 && l.nonBackIn(vid) == 1'
mutate "side branch drifts only one column" layout/place.go 'tries < 3' 'tries < 1'
mutate "start not placed at row 0" layout/place.go 'case geometryOf(v.Type).StartsFlow:' 'case false:'
mutate "db and text try the opposite side first" layout/place.go '[]int{side, -side, 2 * side, -2 * side}' '[]int{-side, side, 2 * side, -2 * side}'
mutate "horizontal arrow does not avoid other horizontal arrows" layout/place.go 'if s.row == r && s.a.less(b) && a.less(s.b) {' 'if false {'
mutate "horizontal arrow does not avoid occupied cells in between" layout/place.go 'if k.row == r && a.less(gk{k.lane, k.col}) && (gk{k.lane, k.col}).less(b) {' 'if false {'
mutate "reference source is the shallowest source" layout/place.go 'if a.Row > b.Row ||' 'if a.Row < b.Row ||'
mutate "hside side reversed" layout/place.go 'toRight := (gk{u.Lane, u.Col}).less(gk{v.Lane, col})' 'toRight := !(gk{u.Lane, u.Col}).less(gk{v.Lane, col})'
mutate "main branch counted as side branch" layout/branch.go 'return n - 1' 'return n'

# layout/route
mutate "A does not check bottom side already has an outgoing edge" layout/route.go "if len(l.sideOut[sideKey{u.ID, 'B'}]) > 0 {" 'if false {'
mutate "vertical cell lets wires to other targets overlap" layout/route.go 'if id != dst {' 'if false {'
mutate "B does not avoid sides with a db" layout/route.go 'if l.attachSides(u)[s] || l.attachSides(v)[opp(s)] {' 'if false {'
mutate "C does not avoid sides with a db" layout/route.go 'if l.sideUsed(u, s) || l.attachSides(u)[s] {' 'if l.sideUsed(u, s) {'
mutate "C does not check the corner cell" layout/route.go 'hcells := append(l.cellsBetween(gkOf(u), gkOf(v), u.Row), turn)' 'hcells := l.cellsBetween(gkOf(u), gkOf(v), u.Row)'
mutate "back edge allowed to exit from bottom" layout/route.go 'if e.Back || v.Row <= u.Row {' 'if false {'
mutate "D prefers bottom exit over side" layout/route.go "choices := []byte{s, 'B', opp(s)}" "choices := []byte{'B', s, opp(s)}"
mutate "D adjacent-row shortcut disabled" layout/route.go 'if v.Row == u.Row+1 {' 'if false {'
mutate "D empty-column shortcut disabled" layout/route.go 'if all(vcells, func(c cell) bool { return vOK(c, v.ID) }) && gkOf(u) != gkOf(v) {' 'if false {'
mutate "gutter side reversed on bottom exit" layout/route.go 'if !gkOf(u).less(gkOf(v)) {' 'if gkOf(u).less(gkOf(v)) {'
mutate "side gutter starts at row middle instead of below node" layout/route.go 'l.seg(gs, 2*u.Row+1,' 'l.seg(gs, 2*u.Row,'
mutate "side gutter connector side reversed" layout/route.go 'dir = -1' 'dir = 1'
mutate "gutter segment on bottom exit becomes a separate segment" layout/route.go 's2 := l.seg(gt, 2*(u.Row+1), 2*v.Row, v.ID)' 's2 := l.seg(gt, 2*(u.Row+1), 2*v.Row, "")'
mutate "diamond shares one side among several edges" layout/route.go 'return len(used) == 0' 'return true'
mutate "rectangle shares a side that already has a B or C edge" layout/route.go "if x.Case == 'B' || x.Case == 'C' {" 'if false {'
mutate "same column heads left" layout/route.go 'if gkOf(u) == gkOf(v) || gkOf(u).less(gkOf(v)) {' 'if gkOf(u).less(gkOf(v)) {'

# port and track assignment
mutate "drop port set that avoids side middle" layout/tracks.go '} else if (len(fixed) > 0 || entries) && n <= 6 {' '} else if false {'
mutate "outgoing wire not separated from incoming wire on the same side" layout/tracks.go 'entries := len(l.sideIn[key]) > 0' 'entries := false'
mutate "separated port not on diamond outline" layout/shape.go '		inset = math.Abs(t - 0.5)' '		inset = 0'
mutate "condition accepts horizontal wire on a side" layout/shape.go 'Shape: ShapeDiamond, EntryTopOnly: true,' 'Shape: ShapeDiamond,'
mutate "side ports ordered by column instead of row" layout/tracks.go "if side == 'L' || side == 'R' {" 'if false {'
mutate "left side port placed at right edge" layout/shape.go '	return [2]float64{inset, t}' '	return [2]float64{far, t}'
mutate "shared track to same target not preferred" layout/tracks.go 'if fits(s, ti, true) {' 'if false {'
mutate "segments to same target may not overlap" layout/tracks.go 'if !(s.Key != "" && t.Key == s.Key) && overlap(t, s) {' 'if overlap(t, s) {'
mutate "drop connector order constraint" layout/tracks.go 'if pa.Pos == pb.Pos && pa.Dir < pb.Dir {' 'if false {'
mutate "segments to same target still bound by order constraint" layout/tracks.go 'if a.Key != "" && a.Key == b.Key {' 'if false {'
mutate "two segments touching at ends not counted as overlap" layout/tracks.go 'return a.Lo <= b.Hi && b.Lo <= a.Hi' 'return a.Lo < b.Hi && b.Lo < a.Hi'
mutate "new track always appended at end" layout/tracks.go 'chosen = max(lo, min(hi, len(tracks)))' 'chosen = len(tracks)'
mutate "orderOK ignores forward constraint" layout/tracks.go 'if mustPrecede(s, t) && !(ti < tj) {' 'if false {'

# layout/geometry
mutate "too little room for label of bottom-exit edge" layout/geometry.go 'e.LH+8)' 'e.LH+4)'
mutate "no room for label of D edge" layout/geometry.go "case e.Case == 'D' || (crosses && spans != nil):" 'case (crosses && spans != nil):'
mutate "label shortfall computed with margin 8 instead of 16" layout/geometry.go 'e.LW + 16 - spans[e.ID]' 'e.LW + 8 - spans[e.ID]'
mutate "label room always in left half of gutter" layout/geometry.go "key := gzKey{u.Lane, l.gutter(u.Lane, u.Col, e.ExitSide), 'r'}" "key := gzKey{u.Lane, l.gutter(u.Lane, u.Col, e.ExitSide), 'l'}"
mutate "hugging also applies to db two columns away" layout/geometry.go 'abs(a.Col-u.Col) == 1' 'abs(a.Col-u.Col) <= 2'
mutate "hugging also applies to side with a wire" layout/geometry.go 'if len(near) != 1 || l.sideUsed(u, side) {' 'if len(near) != 1 {'
mutate "hugging db still takes column width" layout/geometry.go 'if _, hugged := g.hugs[it.ID]; !hugged {' 'if true {'
mutate "lane header margin 20 instead of 30" layout/geometry.go '"\n"))) + 30' '"\n"))) + 20'
mutate "lane names joined with space instead of newline" layout/geometry.go 'strings.Join(lrow.Lines, "\n")' 'strings.Join(lrow.Lines, " ")'
mutate "lane width remainder all goes to first gutter" layout/geometry.go 'widths[0] += need / 2' 'widths[0] += need'
mutate "gutter counts one extra track" layout/geometry.go 'inner = float64(2*cfg.GutterMargin + (n-1)*cfg.TrackGap)' 'inner = float64(2*cfg.GutterMargin + n*cfg.TrackGap)'
mutate "channel has no minimum height" layout/geometry.go 'h := fmax(float64(cfg.MinChannel), lzC[k]+inner)' 'h := lzC[k] + inner'
mutate "db hugging right side has no gap to node" layout/geometry.go 'a.X = h.u.X + h.u.W + float64(cfg.AttachGap)' 'a.X = h.u.X + h.u.W'
mutate "trackY drops label room" layout/geometry.go 'return g.chanY[s.Res.A] + g.lzC[s.Res.A] +' 'return g.chanY[s.Res.A] +'
mutate "collinear points not removed" layout/geometry.go 'if vertical || horizontal {' 'if false {'
mutate "duplicate points not removed" layout/geometry.go 'if math.Abs(p[0]-last[0]) < 0.01 && math.Abs(p[1]-last[1]) < 0.01 {' 'if false {'
mutate "bend points rounded to one digit" layout/geometry.go 'num.Round(out[i][0], 2)' 'num.Round(out[i][0], 1)'
mutate "second geometry pass dropped" layout/labels.go 'lzG, lzC = l.labelReservations(l.horizontalSpans())' 'lzG, lzC = l.labelReservations(nil)'

# layout/labels
mutate "node overlap cost lighter" layout/labels.go 'cost += float64(area(c.b, nb.b) * 4)' 'cost += float64(area(c.b, nb.b) * 2)'
mutate "placed labels not avoided" layout/labels.go 'cost += float64(area(c.b, pb) * 4)' 'cost += 0'
mutate "edge's own wire counted" layout/labels.go 'if w.id != e.ID && segHits(w.a, w.b, c.b) {' 'if segHits(w.a, w.b, c.b) {'
mutate "cost tie picks later candidate" layout/labels.go 'if best == nil || cost < bestCost {' 'if best == nil || cost <= bestCost {'
mutate "segments shorter than 20 get no label" layout/labels.go 'if L >= 12 {' 'if L >= 20 {'
mutate "horizontal label 4 from segment start instead of 6" layout/labels.go 'a[0] + float64(d*(6+w/2))' 'a[0] + float64(d*(4+w/2))'
mutate "vertical label tries left first" layout/labels.go '[]float64{a[0] + 5 + w/2, a[0] - 5 - w/2}' '[]float64{a[0] - 5 - w/2, a[0] + 5 + w/2}'
mutate "label_t rounded to two digits" layout/labels.go 'num.Round(float64(2*best.dist)/total-1, 4)' 'num.Round(float64(2*best.dist)/total-1, 2)'
mutate "header box dropped" layout/labels.go 'box{-1e6, -1e6, 1e6,' 'box{-1e6, -1e6, -1e6,'
mutate "lane separator line dropped" layout/labels.go 'for _, x := range l.LaneX[1:] {' 'for _, x := range l.LaneX[:0] {'
mutate "touching top or bottom edge of box counts as crossing" layout/labels.go 'return x1 < bx[2] && x2 > bx[0] && y1 < bx[3] && y2 > bx[1]' 'return x1 < bx[2] && x2 > bx[0] && y1 <= bx[3] && y2 >= bx[1]'

# layout/check
mutate "node-over-node not checked" layout/check.go 'if area(boxes[a], boxes[b]) > 0 {' 'if false {'
mutate "lane overflow checked only on left edge" layout/check.go 'if it.X < lx || it.X+it.W > lx+lw {' 'if it.X < lx {'
mutate "diagonal segment needs offset on only one axis" layout/check.go 'if math.Abs(a[0]-b[0]) > 0.01 && math.Abs(a[1]-b[1]) > 0.01 {' 'if math.Abs(a[0]-b[0]) > 0.01 || math.Abs(a[1]-b[1]) > 0.01 {'
mutate "every segment touching source and target node exempted" layout/check.go '(it.ID == e.Src || it.ID == e.Dst) && (k == 0 || k == len(pts)-2)' '(it.ID == e.Src || it.ID == e.Dst)'
mutate "node box not shrunk before crossing check" layout/check.go 'segHits(a, b, shrink(boxes[it.ID], 1))' 'segHits(a, b, shrink(boxes[it.ID], 0))'
mutate "wires to same target counted as overlap" layout/check.go 'if w1.edge == w2.edge || e1.Dst == e2.Dst || e1.Src == e2.Src {' 'if w1.edge == w2.edge || e1.Src == e2.Src {'
mutate "wires from same source counted as overlap" layout/check.go 'if w1.edge == w2.edge || e1.Dst == e2.Dst || e1.Src == e2.Src {' 'if w1.edge == w2.edge || e1.Dst == e2.Dst {'
mutate "wire overlap needs to be longer than 3 pixels" layout/check.go 'if collinearOverlap(w1.a, w1.b, w2.a, w2.b) > 1 {' 'if collinearOverlap(w1.a, w1.b, w2.a, w2.b) > 3 {'
mutate "two horizontal wires count as same line only below 0.1 apart" layout/check.go 'math.Abs(a1[1]-a2[1]) < 0.5' 'math.Abs(a1[1]-a2[1]) < 0.1'
mutate "overlapping vertical wires not checked" layout/check.go 'if math.Abs(a1[0]-b1[0]) < 0.01 && math.Abs(a2[0]-b2[0]) < 0.01 && math.Abs(a1[0]-a2[0]) < 0.5 {' 'if false {'
mutate "label over node not checked" layout/check.go 'if area(lb.b, boxes[it.ID]) > 0 {' 'if false {'
mutate "label over label not checked" layout/check.go 'if area(lb.b, lb2.b) > 0 {' 'if false {'
mutate "label over its own edge wire also reported" layout/check.go 'if w.edge != lb.edge && segHits(w.a, w.b, lb.b) {' 'if segHits(w.a, w.b, lb.b) {'
mutate "duplicate detection not deduplicated" layout/check.go 'if !seen[f] {' 'if true {'
mutate "overlapping node pairs not sorted by id" layout/check.go 'sort.Strings(ids)' 'sort.Sort(sort.Reverse(sort.StringSlice(ids)))'
mutate "diagonal segment index counted from 1" layout/check.go '"%s: segment %d is not orthogonal", e.ID, k)' '"%s: segment %d is not orthogonal", e.ID, k+1)'

# writer/drawio
mutate "& not escaped in html content" writer/drawio/write.go 'r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")' 'r := strings.NewReplacer("<", "&lt;", ">", "&gt;")'
mutate "lines joined with newline instead of br" writer/drawio/write.go 'return strings.Join(out, "<br>")' 'return strings.Join(out, "\n")'
mutate "page name truncated at 40 characters" writer/drawio/write.go 'if len(name) > 80 {' 'if len(name) > 40 {'
mutate "page name truncated by byte instead of rune" writer/drawio/write.go 'name := []rune(title)' 'name := []rune(string([]byte(title)[:min(len(title), 80)]))'
mutate "highlighted note uses node style" writer/drawio/write.go 'if it.Shape == layout.ShapeNote {' 'if false {'
mutate "dashes dropped" writer/drawio/write.go 'style += "dashed=1;"' 'style += ""'
mutate "label x attribute dropped" writer/drawio/write.go 'g.Set("x", f(e.LabelT))' '_ = e.LabelT'
mutate "start and end points written into points" writer/drawio/write.go 'pts = e.Pts[1 : len(e.Pts)-1]' 'pts = e.Pts'
mutate "wire kept from old file still writes computed anchors" writer/drawio/write.go '		case e.Kept:
			for _, kv := range e.Constraints {' '		case false:
			for _, kv := range e.Constraints {'
mutate "auto-routed wire still writes anchors" writer/drawio/write.go '		switch {
		case e.Auto:
		case e.Kept:
			for' '		switch {
		case false:
		case e.Kept:
			for'
mutate "wire kept from old file still writes computed bend points" writer/drawio/write.go '			pts = e.Waypoints' '			pts = e.Pts'
mutate "label reset to middle still writes position" writer/drawio/write.go 'withLabel := e.Label != nil && !e.NoLabelPos && !e.Auto' 'withLabel := e.Label != nil && !e.Auto'
mutate "auto-routed wire still writes label position" writer/drawio/write.go 'withLabel := e.Label != nil && !e.NoLabelPos && !e.Auto' 'withLabel := e.Label != nil && !e.NoLabelPos'
mutate "hand-drawn cells dropped" writer/drawio/write.go '	root.Children = append(root.Children, extras...)' '	_ = extras'
mutate "other pages dropped" writer/drawio/write.go '	mxfile.Children = append(mxfile.Children, pages...)' '	_ = pages'

# internal/etree
mutate "double quote not escaped in attribute" internal/etree/etree.go '"\"", "&quot;",' ''
mutate "tab not escaped in attribute" internal/etree/etree.go '"\t", "&#09;")' '"\t", "\t")'
mutate "newline not escaped in attribute" internal/etree/etree.go '"\n", "&#10;",' '"\n", "\n",'
mutate "double quote escaped in text" internal/etree/etree.go 'textEsc = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")' 'textEsc = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")'
mutate "empty element has no space before slash" internal/etree/etree.go 'b.WriteString(" />")' 'b.WriteString("/>")'
mutate "text-only element written as empty tag" internal/etree/etree.go 'if e.Text != "" || len(e.Children) > 0 {' 'if len(e.Children) > 0 {'
mutate "tail dropped on write" internal/etree/etree.go '	b.WriteString(textEsc.Replace(e.Tail))' '	_ = e.Tail'
mutate "indent of four spaces" internal/etree/etree.go 'levels = append(levels, levels[level]+"  ")' 'levels = append(levels, levels[level]+"    ")'
mutate "indentation overwrites real text" internal/etree/etree.go '		if unistr.Strip(e.Text) == "" {
			e.Text = child' '		if true {
			e.Text = child'
mutate "indentation overwrites real tail" internal/etree/etree.go '			if unistr.Strip(c.Tail) == "" {' '			if true {'
mutate "last child's tail not dedented to parent level" internal/etree/etree.go '			last.Tail = levels[level]' '			last.Tail = child'
mutate "text after child element merged into parent text" internal/etree/etree.go '			if len(cur.Children) == 0 {' '			if true {'
mutate "newlines in attribute not normalized" internal/etree/etree.go '			case '"'"'\n'"'"', '"'"'\t'"'"':
				out.WriteByte('"'"' '"'"')' '			case '"'"'\t'"'"':
				out.WriteByte('"'"' '"'"')'
mutate "CRLF in attribute becomes two spaces" internal/etree/etree.go '				if i < n && data[i] == '"'"'\n'"'"' {
					i++
				}' ''
mutate "normalization also applied inside comments" internal/etree/etree.go '		if j, ok := skip(i, "<!--", "-->"); ok {' '		if j, ok := skip(i, "<!-- ", "-->"); ok {'
mutate "Set moves existing attribute to the end" internal/etree/etree.go '		if e.Attrs[i][0] == k {
			e.Attrs[i][1] = v
			return
		}' ''
mutate "Copy shares attributes" internal/etree/etree.go 'Attrs: append([][2]string(nil), e.Attrs...)}' 'Attrs: e.Attrs}'

mutate "lane does not subtract pool header" writer/drawio/write.go 'geo(c, r.LaneX[i], float64(r.PoolHeader), r.LaneW[i], r.PoolH-float64(r.PoolHeader))' 'geo(c, r.LaneX[i], float64(r.PoolHeader), r.LaneW[i], r.PoolH)'

# layout/optimize
mutate "optimizer ignores round count" layout/optimize.go '	if cfg.Optimize <= 0 {' '	if true {'
mutate "optimizer lets wires pass through items" layout/optimize.go '			if segHits(a, b, grow(bx, gap)) {' '			if false {'
mutate "optimizer accepts steps that do not lower cost" layout/optimize.go '	bestCost := base - minGain' '	bestCost := base + 1e9'
mutate "optimizer may flip first and last segments" layout/optimize.go '	if !sameDir(pts[0], pts[1], out[0], out[1]) || !sameDir(pts[n-2], pts[n-1], out[n-2], out[n-1]) {' '	if false {'
mutate "optimizer keeps no distance from other wires" layout/optimize.go '				if tooClose(a, b, f.Pts[q], f.Pts[q+1], gap, shared) {' '				if false {'
mutate "optimizer does not re-anchor labels" layout/optimize.go '	p, dist, _ := anchor(pts, *e.Label, e.LabelOff)' '	return
	p, dist, _ := anchor(pts, *e.Label, e.LabelOff)'
mutate "optimizer ignores crossings" layout/optimize.go '					c += costCross' '					c += 0'

mutate "item move ignores its own wires' fixed segments as obstacles" layout/optimize_items.go '			if moving[k] || moving[k+1] {' '			if true {'
mutate "no pull toward lane center" layout/optimize_items.go 'const costGravity = 0.05' 'const costGravity = 0'
mutate "note pulled closer than attach-gap" layout/optimize_items.go 'math.Abs(gap-float64(o.cfg.AttachGap))' 'math.Max(gap-float64(o.cfg.AttachGap), 0)'
mutate "shorter moves not tried" layout/optimize_items.go '			if dx, paths, ok = o.bisectShift(m, edges, dx); !ok {' '			if true {'
mutate "lane narrowing does not move labels with wires" layout/optimize_items.go '				labelDX = d' '				labelDX = 0'
mutate "lane narrowing ignores labels" layout/optimize_items.go '				add(e.Label[0], e.Label[2])' '				_ = e.Label'
mutate "lane narrowed below minimum width" layout/optimize_items.go 'w := math.Max(o.laneMinW(li), hi-lo+float64(2*margin))' 'w := hi - lo + float64(2*margin)'
mutate "group wires checked against each other's old paths" layout/optimize_items.go '		o.r.Edges[i].Pts = paths[i]
	}
	defer func() {' '		_ = paths[i]
	}
	defer func() {'
mutate "port offset on ellipse so draw.io reprojects it" writer/drawio/write.go '				style += "exitPerimeter=0;"' '				style += ""'
# cmd/flowcast
mutate "errors not listed before warnings" cmd/flowcast/main.go 'return sorted[i].Level == model.LevelError && sorted[j].Level != model.LevelError' 'return false'
mutate "build line writes size with spaces" cmd/flowcast/main.go '(%d lanes, %d elements, %d edges, %sx%spx)' '(%d lanes, %d elements, %d edges, %s x %spx)'
mutate "default output path keeps source extension" cmd/flowcast/main.go 'out = strings.TrimSuffix(c.a.file, source.Ext(c.a.file)) + ".drawio"' 'out = c.a.file + ".drawio"'
mutate "leading dot of file name counts as extension separator" source/source.go 'trimmed := strings.TrimLeft(base, ".")' 'trimmed := base'
mutate "system error message not capitalized" cmd/flowcast/main.go 'r[0] = unicode.ToUpper(r[0])' '_ = r'
mutate "force does not back up old file" cmd/flowcast/main.go 'if mode != "new" && !c.a.noBackup {' 'if mode == "merge" && !c.a.noBackup {'
mutate "still asks for mode without a terminal" cmd/flowcast/main.go 'if !isTerminal(c.stdin) {' 'if false {'
mutate "geometry error does not change exit code" cmd/flowcast/main.go '		code = 2' '		code = 0'
mutate "layout warning prefix dropped" cmd/flowcast/main.go 'c.println("WARNING layout: " + w.Msg)' 'c.println(w.Msg)'
mutate ".txt not accepted as markdown" cmd/flowcast/main.go '".txt": true,' ''
mutate "check does not exit 1 on an invalid table" cmd/flowcast/main.go '	if c.printIssues(r.Issues) > 0 {
		return 1
	}
	return 0
}' '	c.printIssues(r.Issues)
	return 0
}'
mutate "--no-backup has no effect" cmd/flowcast/args.go 'a.noBackup = true' 'a.noBackup = false'
mutate "config flag before file name ignored" cmd/flowcast/args.go '			*ints[name] = n' '			_ = n'
mutate "--flag=value syntax does not split value" cmd/flowcast/args.go 'name, val, hasVal := strings.Cut(tok[2:], "=")' 'name, val, hasVal := tok[2:], "", false'

# source/csv
mutate "text after closing quote not appended to cell" source/csvread.go '				state = inField
				return add(c)
			}
		case eatCRNL:' '				state = inField
				return nil
			}
		case eatCRNL:'
mutate "unfinished cell dropped at end of data inside quotes" source/csvread.go 'if len(field) != 0 || state == inQuotedField {' 'if len(field) != 0 {'
mutate "lone CR not a line boundary when reading csv" source/csvread.go 'i := strings.IndexAny(s, "\r\n")' 'i := strings.IndexAny(s, "\n")'
mutate "utf-8-sig does not strip BOM" source/encoding.go 'return strings.TrimPrefix(string(raw), "\ufeff"), true' 'return string(raw), true'
mutate "cp1252 accepts undefined bytes" source/encoding.go '			if !ok {
				return "", false
			}
			b.WriteRune(r)' '			_ = ok
			b.WriteRune(r)'
mutate "cp1252 tried before utf-8" source/encoding.go 'var csvEncodings = []string{"utf-8-sig", "utf-8", "cp1252"}' 'var csvEncodings = []string{"cp1252", "utf-8-sig", "utf-8"}'
mutate "semicolon guessed before comma" source/grid.go 'delims := []string{",", ";", "\t"}' 'delims := []string{";", ",", "\t"}'
mutate "header must start in first column" source/grid.go 'for c0 := 0; c0 < max(1, len(low)-4); c0++ {' 'for c0 := 0; c0 < 1; c0++ {'
mutate "title not taken from above header" source/grid.go 'for r := 0; r < hr && title == ""; r++ {' 'for r := 0; r < 0 && title == ""; r++ {'
mutate "table does not stop at empty row" source/grid.go '		if empty {
			break
		}' '		if empty {
			continue
		}'
mutate "no warning for markdown escapes in csv" source/grid.go 'if strings.Contains(cells[3], `\|`) || strings.Contains(cells[4], `\|`) {' 'if false {'
mutate "csv cell not split at real newline" source/grid.go 'parts = append(parts, strings.Split(chunk, "\n")...)' 'parts = append(parts, chunk)'
mutate "cp1252 warning compares normalized name" source/grid.go 'if used == "cp1252" {' 'if c, _ := normalizeEncoding(used); c == "cp1252" {'
mutate "whitespace drops U+001C to U+001F" internal/unistr/unistr.go 'return unicode.IsSpace(r) || (r >= 0x1c && r <= 0x1f)' 'return unicode.IsSpace(r)'
mutate "Lower does not split İ into two characters" internal/unistr/unistr.go 'if r == 0x130 {' 'if false {'
mutate "br tag does not accept Unicode whitespace" source/text.go 'for i < len(r) && unistr.IsSpace(r[i]) {
		i++
	}
	if i < len(r) && r[i] == '"'"'/'"'"' {' 'for i < len(r) && r[i] == '"'"' '"'"' {
		i++
	}
	if i < len(r) && r[i] == '"'"'/'"'"' {'
mutate "all-whitespace title does not match" source/markdown.go '		return unescape(string(rest[len(rest)-1:])), true' '		return "", false'
mutate "separator row trims trailing whitespace too" source/markdown.go '	s := unistr.LStrip(ln)
	if !strings.HasPrefix(s, "|") || len(s) == 1 {' '	s := unistr.Strip(ln)
	if !strings.HasPrefix(s, "|") || len(s) == 1 {'

# source/xlsx
mutate "shared string read as empty" source/xlsx.go '						val = shared[i]' '						_ = i'
mutate "inline string read as empty" source/xlsx.go '					val = is.textOf("t")' '					_ = is'
mutate "integer does not drop .0" source/xlsx.go "					val = val[:strings.IndexByte(val, '.')]" '					_ = val'
mutate "numeric cell warned only in id column" source/xlsx.go '(ci == 0 || ci == 2)' '(ci == 0)'
mutate "no warning for empty formula cell" source/xlsx.go 'if c.find("f") != nil {' 'if false {'
mutate "no warning for hidden row" source/xlsx.go 'if h, _ := rowEl.get("hidden"); h == "1" {' 'if false {'
mutate "merged cell carries no row number" source/xlsx.go '		r, _ := strconv.Atoi(digits)' '		r := 0 * len(digits)'
mutate "warnings outside table range too" source/xlsx.go 'if hr <= n.row && n.row <= hr+len(rows) {' 'if true {'
mutate "sheet without header stops instead of trying next sheet" source/xlsx.go 'if _, _, found := findHeader(grid); !found {' 'if false {'
mutate "absolute sheet path does not strip leading slash" source/xlsx.go 'if strings.HasPrefix(target, "/xl/") {' 'if false {'
mutate "--sheet does not filter" source/xlsx.go '			if s.name == sheet {' '			if true {'
mutate "columns taken by cell order instead of address" source/xlsx.go '				ci = colIndex(ref)' '				ci = len(cells)'

# merge
mutate "number does not accept surrounding whitespace" merge/parse.go '	t := strings.Trim(b.String(), " \t\n\v\f\r")' '	t := b.String()'
mutate "number does not accept underscore" merge/parse.go "			if c == '_' && n > 0 && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {" '			if false {'
mutate "number does not accept inf" merge/parse.go '	case "inf", "infinity":' '	case "infinity":'
mutate "number accepts two signs" merge/parse.go '	if len(t)-len(body) > 1 {' '	if len(t)-len(body) > 2 {'
mutate "base64 stops at any padding character" merge/parse.go '			if quad >= 2 {
				pads++' '			if true {
				pads++'
mutate "base64 errors on unknown character" merge/parse.go '		if v < 0 {
			continue
		}' '		if v < 0 {
			return nil, errors.New("Incorrect padding")
		}'
mutate "unquote decodes invalid %XX too" merge/parse.go '			if ok1 && ok2 {' '			if ok1 || ok2 {'
mutate "%XX decoding dropped" merge/parse.go '		b.WriteString(decodeReplace(unquoteBytes(s[:n])))' '		b.WriteString(s[:n])'
mutate "broken UTF-8 replaced byte by byte" merge/parse.go '		if need > 0 && j-i == need+1 {
			b.Write(p[i:j])
		} else {
			b.WriteRune(utf8.RuneError)
		}
		i = j' '		if need > 0 && j-i == need+1 {
			b.Write(p[i:j])
			i = j
		} else {
			b.WriteRune(utf8.RuneError)
			i++
		}'
mutate "first page read skips object elements" merge/read.go '			if cell = el.Find("mxCell"); cell == nil {' '			if cell = nil; cell == nil {'
mutate "duplicate id keeps first cell" merge/read.go '		old.cells[cid] = &oldCell{cid, el, cell, i}' '		if old.cells[cid] == nil {
			old.cells[cid] = &oldCell{cid, el, cell, i}
		}'
mutate "other pages not kept" merge/read.go '		old.Pages = pages[1:]' '		_ = pages'
mutate "style does not record keys without equals sign" merge/read.go '		st.vals[k] = styleVal{v, found}' '		st.vals[k] = styleVal{v, true}'
mutate "broken number read as one" merge/read.go '	v, ok := parseFloat(s)
	if !ok {
		return 0
	}' '	v, ok := parseFloat(s)
	if !ok {
		return 1
	}'
mutate "conventional id does not accept child digits" merge/merge.go '	if strings.HasPrefix(s, "E") && digitsDots(s[1:]) {' '	if strings.HasPrefix(s, "E") && digitsDots(strings.Split(s[1:], ".")[0]) && !strings.Contains(s, ".") {'
mutate "conventional id does not accept underscore" merge/merge.go "s[i] >= '0' && s[i] <= '9' || s[i] == '_') {" "s[i] >= '0' && s[i] <= '9') {"
mutate "without marker every cell belongs to the tool" merge/merge.go '		return shaped && c.style().has("flowtable")' '		return true'
mutate "lane not recognized by swimlane style" merge/merge.go '	laneLike := c.parent() == "pool" && strings.HasPrefix(c.styleRaw(), "swimlane")' '	laneLike := false'
mutate "old pool position not kept" merge/merge.go '		r.Origin = [2]float64{b[0], b[1]}' '		_ = b'
mutate "coordinate origin not accumulated through parents" merge/merge.go '		cid = c.parent()
	}
	return x, y' '		break
	}
	return x, y'
mutate "lane keeps newly computed width" merge/merge.go '				w = ow' '				_ = ow'
mutate "unedited lane takes rounded width" merge/merge.go 'if ow := attrf(oc.geo(), "width"); num.Fmt(ow) != num.Fmt(w) {' 'if ow := attrf(oc.geo(), "width"); true {'
mutate "deleted lane follows last lane instead of next lane" merge/merge.go '					target, base = t, newX[t]
					break' '					_ = t
					break'
mutate "mapX before first lane takes last lane" merge/merge.go '		return m.spans[0].dx, m.spans[0].target' '		return m.spans[len(m.spans)-1].dx, m.spans[len(m.spans)-1].target'
mutate "mapX accepts right edge too" merge/merge.go '		if s.x0 <= x && x < s.x1 {' '		if s.x0 <= x && x <= s.x1 {'
mutate "node that changed lane keeps position" merge/merge.go '		if c.parent() != laneID {' '		if false {'
mutate "kept node does not follow shifted lane" merge/merge.go '			dx = r.LaneX[it.Lane] - attrf(oc.geo(), "x")
		}
		it.X, it.Y' '			_ = oc
		}
		it.X, it.Y'
mutate "anchor search finds outgoing edge before incoming" merge/merge.go '	for _, e := range byBackOrder(m.ins[v.ID]) {
		if m.inFinal[e.Src] {' '	for _, e := range byBackOrder(nil) {
		if m.inFinal[e.Src] {'
mutate "anchor ignores last loop edge" merge/merge.go '			return !out[i].Back' '			return out[i].Back'
mutate "anchor search not widened" merge/merge.go '		frontier = nxt
	}' '		frontier = nil
	}'
mutate "neighbors do not sort last loop edge" merge/merge.go '	for _, back := range []bool{false, true} {' '	for _, back := range []bool{true, false} {'
mutate "obstacles drop hand-drawn cells" merge/merge.go '	return append(out, m.fhBoxes...)' '	return out'
mutate "new node not shifted on overlap" merge/merge.go '		it.Y += step' '		break'
mutate "overlap ignores margin" merge/merge.go '		b := box{it.X - pad, it.Y - pad, it.X + it.W + pad, it.Y + it.H + pad}' '		b := box{it.X, it.Y, it.X + it.W, it.Y + it.H}'
mutate "new node not pulled inside lane" merge/merge.go '	if lw >= it.W+2*pad {' '	if false {'
mutate "new nodes ordered by table order instead of topo" merge/merge.go '		if a != b {
			return a < b
		}
		return flow[i].Order < flow[j].Order' '		return flow[i].Order < flow[j].Order'
mutate "node without anchor placed at lane start" merge/merge.go '					bottom = fmax(bottom, b[3])' '					bottom = fmin(bottom, b[3])'
mutate "node in other lane keeps anchor x offset" merge/merge.go '	if an.Lane == it.Lane {
		cx = ax + fv[0] - fa[0]' '	if true {
		cx = ax + fv[0] - fa[0]'
mutate "wire kept although its endpoint changed" merge/merge.go '			keep = sok && dok && src == e.Src && dst == e.Dst && m.pinned[e.Src] && m.pinned[e.Dst]' '			keep = sok && dok && m.pinned[e.Src] && m.pinned[e.Dst]'
mutate "wire kept although its endpoint node was re-placed" merge/merge.go '			keep = sok && dok && src == e.Src && dst == e.Dst && m.pinned[e.Src] && m.pinned[e.Dst]' '			keep = sok && dok && src == e.Src && dst == e.Dst'
mutate "bend points do not follow shifted lane" merge/merge.go '			e.Waypoints = append(e.Waypoints, [2]float64{ax + dx, ay})' '			e.Waypoints = append(e.Waypoints, [2]float64{ax, ay})'
mutate "anchor keys without value kept" merge/merge.go '			if v, ok := st.vals[k]; ok && v.set {' '			if v, ok := st.vals[k]; ok {'
mutate "bend point counted as unedited however far it moved" merge/merge.go '		if math.Abs(a[0]-b[0]) > 2 || math.Abs(a[1]-b[1]) > 2 {' '		if false {'
mutate "edited anchor counted as unedited" merge/merge.go '		if !ok || num.Fmt(v) != num.Fmt(w.v) {' '		if !ok {'
mutate "anchor compared without rounding" merge/merge.go '		if !ok || num.Fmt(v) != num.Fmt(w.v) {' '		if !ok || v != w.v {'
mutate "hand-edited wire without label still reports label reset" merge/merge.go '			if len(e.Lines) > 0 {
				m.rep.LabelsCentered' '			if true {
				m.rep.LabelsCentered'
mutate "bend points do not follow widened lane" merge/merge.go '				e.Waypoints[k][0] += dx' '				_ = k'
mutate "hand-drawn cell in pool does not follow shifted lane" merge/merge.go '			dx, _ := m.mapX(x + b[2]/2)
			x += dx' '			_, _ = m.mapX(x + b[2]/2)'
mutate "hand-drawn cell with deleted parent kept" merge/merge.go '			ok = m.gen[p] || isLayer(p) || kept[p]' '			ok = true'
mutate "hand-drawn cell in deleted lane dropped" merge/merge.go '			ok = m.oldLane(p) != nil' '			ok = false'
mutate "hand-drawn cell in deleted lane not moved to pool" merge/merge.go '				cell.Set("parent", "pool")' '				_ = cell'
mutate "hand-drawn cell in deleted lane does not add lane origin" merge/merge.go '					g.Set("x", num.Fmt(attrf(g, "x")+ax-m.ox))' '					_ = ax'
mutate "hand-drawn wire endpoints not detached" merge/merge.go '		if c.edge() {
			m.detach(c, cell, g, alive, kept)
		}' ''
mutate "endpoint into kept hand-drawn cell also detached" merge/merge.go '		if ref == "" || kept[ref] || (alive[ref] && m.gen[ref]) {' '		if ref == "" || (alive[ref] && m.gen[ref]) {'
mutate "free point does not drop old point with same name" merge/merge.go '				g.Remove(pt)' '				_ = pt'
mutate "offset point moved too" merge/merge.go '		if as, _ := pt.Get("as"); as != "offset" {' '		if true {'
mutate "relative geometry moves x and y" merge/merge.go '	if v, _ := g.Get("relative"); v == "1" {' '	if false {'
mutate "lane-bound hand-drawn cell does not follow widened lane" merge/merge.go '		m.addMovable(lane, func(dx, dy float64) { shiftGeo(g, dx, dy) }, true)' '		_ = lane'
mutate "absolute hand-drawn cell does not subtract pool origin" merge/merge.go '	if fr == "abs" {
		ox = m.ox
	}' ''
mutate "hand-drawn wire does not follow lane" merge/merge.go '		pt.Set("x", num.Fmt(attrf(pt, "x")+dx))
		m.addMovable' '		m.addMovable'
mutate "widened lane does not shift later lanes" merge/merge.go '			r.LaneX[j] += gl + gr' '			_ = j'
mutate "widened lane shifts its contents by the total" merge/merge.go '			if mv.lane == i && gl != 0 {
				mv.shift(gl, 0)' '			if mv.lane == i && gl != 0 {
				mv.shift(gl+gr, 0)'
mutate "widened lane shifts relative contents of later lanes too" merge/merge.go '			} else if mv.lane > i && !mv.rel {' '			} else if mv.lane > i {'
mutate "widened lane has no margin" merge/merge.go '		gr := fmax(0.0, right-(lw-pad))' '		gr := fmax(0.0, right-lw)'
mutate "diagram not pushed down when pulled to top" merge/merge.go '	if low < top {' '	if false {'
mutate "lane start ignores margin" merge/merge.go '	top := float64(r.PoolHeader + r.LaneHeader + pad)' '	top := float64(r.PoolHeader + r.LaneHeader)'
mutate "pool height ignores bend points" merge/merge.go '			bottoms = append(bottoms, p[1])
		}
	}
	for _, b := range m.freehandBoxesNow() {' '		}
	}
	for _, b := range m.freehandBoxesNow() {'
mutate "pool height ignores hand-drawn cells" merge/merge.go '		bottoms = append(bottoms, b[3])' '		_ = b'
mutate "pool height leaves no last channel" merge/merge.go '		h = fmax(h, b+float64(r.MinChannel))' '		h = fmax(h, b)'
mutate "lane-bound hand-drawn cell measured without header" merge/merge.go '			x, y = x+m.r.LaneX[f.lane], y+float64(m.r.PoolHeader)' '			x = x + m.r.LaneX[f.lane]'
mutate "overlapping elements not reported" merge/merge.go '				m.rep.Overlaps = append(m.rep.Overlaps, a.ID+"/"+b.ID)' '				_ = b'
mutate "pool cell reported as deleted too" merge/merge.go '		if !isLayer(id) && id != "pool" && !m.tableIDs[id] {' '		if !isLayer(id) && !m.tableIDs[id] {'
mutate "fsum does not compensate error" merge/merge.go '	if lo != 0 && !math.IsInf(lo, 0) && !math.IsNaN(lo) {' '	if false {'
mutate "merge does not back up old file" cmd/flowcast/main.go 'if mode != "new" && !c.a.noBackup {' 'if mode == "force" && !c.a.noBackup {'
mutate "merge prints report but still prints self-check" cmd/flowcast/main.go '		c.println("layout: merge does not self-check wires and labels; see the list above and the PNG image")
		return c.exportAndVerify(out, r, 0)' '		c.println("layout: merge does not self-check wires and labels; see the list above and the PNG image")
		for _, f := range r.Findings {
			c.println(f.Msg)
		}
		return c.exportAndVerify(out, r, 0)'
# render
mutate "render check not skipped when SVG has no id" render/render.go '	if len(cells) == 0 {' '	if false {'
mutate "anchor shape takes rect without width too" render/render.go 'case n.tag == "rect" && n.attrs["width"] != "":' 'case n.tag == "rect":'
mutate "ellipse anchor shape does not subtract radius" render/render.go 'rect = &[2]float64{attrFloat(n, "cx") - attrFloat(n, "rx"), attrFloat(n, "cy") - attrFloat(n, "ry")}' 'rect = &[2]float64{attrFloat(n, "cx"), attrFloat(n, "cy")}'
mutate "path anchor shape takes first point instead of corner" render/render.go '				rect = &[2]float64{mx, my}' '				rect = &[2]float64{nums[0], nums[1]}'
mutate "anchor is the last element" render/render.go '		if it.Order < anchor.Order {' '		if it.Order > anchor.Order {'
mutate "wire path accepts filled path too" render/render.go 'if n.tag == "path" && n.attrs["d"] != "" && n.attrs["fill"] == "none" {' 'if n.tag == "path" && n.attrs["d"] != "" {'
mutate "tolerance of 5 pixels" render/render.go '	const tol = 2.0' '	const tol = 5.0'
mutate "end point compared like an ordinary point" render/render.go '		for k := 0; k < len(got)-1; k++ {' '		for k := 0; k < len(got); k++ {'
mutate "reports every off point of a wire" render/render.go '					e.ID, k, a[0], a[1], b[0], b[1]))
				break' '					e.ID, k, a[0], a[1], b[0], b[1]))'
mutate "last segment compared on wrong axis" render/render.go '			bad = math.Abs(lastG[0]-lastW[0]) > tol
		} else {' '			bad = math.Abs(lastG[1]-lastW[1]) > tol
		} else {'
mutate "duplicate id keeps first g" render/render.go '		if id := n.attrs["data-cell-id"]; n.tag == "g" && id != "" {' '		if id := n.attrs["data-cell-id"]; n.tag == "g" && id != "" && cells[id] == nil {'
mutate "elements outside svg namespace read too" render/render.go '			if t.Name.Space == svgNS {' '			if true {'
mutate "export sets no scale" render/render.go '	if scale != 0 {' '	if false {'
mutate "failed export does not use stdout when stderr is empty" render/render.go '				msg = strings.TrimSpace(stdout.String())' '				_ = stdout'
mutate "export treats missing output file as success" render/render.go 'if _, statErr := os.Stat(out); err != nil || statErr != nil {' 'if err != nil {'
mutate "image export error overwrites self-check code" cmd/flowcast/main.go '		if code != 0 {
			return code
		}
		return 3' '		return 3'
mutate "render check overwrites self-check code" cmd/flowcast/main.go '		if len(probs) > 0 && code == 0 {' '		if len(probs) > 0 {'
mutate "merge still runs render check" cmd/flowcast/main.go '	if r.Merge != nil && verify {' '	if false {'
mutate "png default keeps .drawio extension" cmd/flowcast/main.go 'png = strings.TrimSuffix(out, source.Ext(out)) + ".png"' 'png = out + ".png"'

# resource limits
mutate "input size not limited" limits.go '	if b.lim.MaxBytes > 0 && len(src.Data) > b.lim.MaxBytes {' '	if false {'
mutate "row limit off by one" limits.go '	if b.lim.MaxRows > 0 && len(t.Rows) > b.lim.MaxRows {' '	if b.lim.MaxRows > 0 && len(t.Rows) >= b.lim.MaxRows {'
mutate "edges not counted" limits.go '			edges++' '			_ = r'
mutate "decompression limit not passed" limits.go '		src.MaxUnzipped = b.lim.MaxUnzipped' '		_ = src'
mutate "time not checked" limits.go '	if b.lim.Timeout > 0 && time.Since(b.start) > b.lim.Timeout {' '	if false {'
mutate "decompression not counted cumulatively" source/xlsx.go '		unzipped += int64(len(b))' '		_ = b'
mutate "Check ignores limits" build.go '	t, err := readAndTrace(src, newBudget(opt.Limits), opt.Trace)' '	t, err := readAndTrace(src, newBudget(nil), opt.Trace)'

mutate "verbose prints nothing" cmd/flowcast/main.go '	if !c.a.verbose {
		return nil
	}' '	if true {
		return nil
	}'
mutate "verbose prints even when off" cmd/flowcast/main.go '	if !c.a.verbose {
		return nil
	}' '	if false {
		return nil
	}'

# cmd/flowcastd
mutate "web does not limit request body" cmd/flowcastd/server.go '	r.Body = http.MaxBytesReader(w, r.Body, int64(s.lim.MaxBytes)+multipartSlack)' '	_ = multipartSlack'
mutate "web keeps path in file name" cmd/flowcastd/server.go '	name := path.Base(strings.ReplaceAll(hdr.Filename, "\\", "/"))' '	name := hdr.Filename'
mutate "web never reports busy" cmd/flowcastd/server.go '	case <-t.C:
	case <-ctx.Done():
	}
	return false' '	case <-t.C:
	case <-ctx.Done():
	}
	return true'
mutate "web returns 200 for invalid table" cmd/flowcastd/server.go '		status = http.StatusUnprocessableEntity
	}
	if build && resp.OK {' '	}
	if build && resp.OK {'
mutate "web returns 422 when over limit" cmd/flowcastd/server.go '	case strings.HasPrefix(code, "limit."):
		return http.StatusRequestEntityTooLarge' '	case false:
		return http.StatusRequestEntityTooLarge'
mutate "web drops nosniff header" cmd/flowcastd/server.go '		w.Header().Set("X-Content-Type-Options", "nosniff")' ''
mutate "web does not pass read options" cmd/flowcastd/server.go '			src.Options[k] = v' '			_ = v'
mutate "web does not use limits" cmd/flowcastd/server.go 'Config: &cfg, Limits: &s.lim,' 'Config: &cfg,'

# flowchart: table without lanes
mutate "table without lanes still requires parent" validate/validate.go '			if r.Parent == "" && len(v.lanes) == 0 {' '			if false {'
mutate "flowchart keeps pool and lane headers" layout/model.go '		l.Cfg.PoolHeader, l.Cfg.LaneHeader = 0, 0' '		_ = l.Cfg'
mutate "flowchart still draws pool" writer/drawio/write.go '	if !r.NoLanes {
		writePool' '	if true {
		writePool'
mutate "flowchart does not add origin to nodes" writer/drawio/write.go '			geo(c, it.X+ox, it.Y+oy, it.W, it.H)' '			geo(c, it.X, it.Y, it.W, it.H)'
mutate "flowchart does not add origin to bend points" writer/drawio/write.go '		edgeParent, dx, dy = "1", ox, oy' '		edgeParent = "1"'
mutate "merge flowchart compares parent with hidden lane" merge/merge.go '	if m.r.NoLanes {
		return "1"
	}' ''

# source/mermaid
mutate "mermaid does not recognize start/end shapes" source/mermaid.go '	{"([", "])", "stadium"},' ''
mutate "mermaid reads (( as (" source/mermaid.go '	{"((", "))", "circle"},' ''
mutate "mermaid drops dashes" source/mermaid.go '		lk.edge.dashed = true' '		_ = lk'
mutate "mermaid drops bold" source/mermaid.go "		lk.edge.bold = ch == '='" '		_ = ch'
mutate "mermaid open arrow still has a head" source/mermaid.go '		lk.edge.noarrow = true' '		_ = lk'
mutate "mermaid drops |...| label" source/mermaid.go '	lk.edge.text = strings.TrimSpace(c.rest()[1 : end+1])' '	_ = end'
mutate "mermaid does not accept label inside arrow" source/mermaid.go '		if n >= 3 || h != "" {' '		if true {'
mutate "mermaid id does not accept hyphen and dot" source/mermaid.go "		if (r == '-' || r == '.') && c.i > start && c.i+1 < len(c.s) {" '		if false {'
mutate "mermaid id does not accept dot" source/mermaid.go "if (r == '-' || r == '.') &&" "if (r == '-') &&"
mutate "mermaid does not read &" source/mermaid.go "		if c.i < len(c.s) && c.s[c.i] == '&' {" '		if false {'
mutate "mermaid does not mark back" source/mermaid.go '			if pos[e.to] <= pos[e.from] {' '			if false {'
mutate "mermaid merge does not wait for sources" source/mermaid.go '			if !back[e] && e.from != id && !written[e.from] {' '			if false {'
mutate "mermaid cylinder drawn as task" source/mermaid.go '	case "cylinder":
		return "db"' '	case "cylinder":
		return "task"'
mutate "mermaid one-branch condition stays condition" source/mermaid.go '		if types[id] == "condition" && len(outs[id]) < 2 {' '		if false {'
mutate "mermaid node outside subgraph has no lane" source/mermaid.go '		out[id] = loose' '		_ = id'
mutate "mermaid nested subgraph becomes its own lane" source/mermaid.go '	if len(m.stack) > 0 {
		m.warn(ln, "mermaid.nested_subgraph",' '	if false {
		m.warn(ln, "mermaid.nested_subgraph",'
mutate "mermaid does not rename reserved ids" source/mermaid.go '		if id == "0" || id == "1" || id == "pool" {' '		if false {'
mutate "mermaid duplicate edge has no suffix" source/mermaid.go '		if seen[base] > 1 {' '		if false {'
mutate "mermaid does not decode #quot;" source/mermaid.go '				if v, ok := entities[code]; ok {' '				if v, ok := entities[""]; ok {'
mutate "mermaid does not split lines at br" source/mermaid.go '	parts := splitBR(text)' '	parts := []string{text}'
mutate "mermaid drops highlight color" source/mermaid.go '	if props["fill"] == highlightFill {' '	if false {'
mutate "mermaid does not read title" source/mermaid.go '			m.title = normalize(unquote(strings.TrimSpace(v)))' '			_ = v'
mutate "mermaid does not read block in markdown" source/source.go '		if block, ok := mermaidBlock(s.Data); ok && isCode(err, "source.no_header") {' '		if block, ok := mermaidBlock(s.Data); false && ok {'
mutate "mermaid warnings not in line order" source/mermaid.go '	sort.SliceStable(m.issues, func(i, j int) bool { return m.issues[i].Loc.Line < m.issues[j].Loc.Line })' ''
mutate "edge does not accept bold" layout/model.go '			Bold: hasStyle(r, "bold"), NoArrow: hasStyle(r, "noarrow"),' '			NoArrow: hasStyle(r, "noarrow"),'
mutate "bold written before highlight color" writer/drawio/write.go '			style += "strokeWidth=3;"' '			style += ""'

# main branch heuristic for diagrams without lanes
mutate "main branch is always the last edge" layout/branch.go '	last := len(same) - 1
	if !l.NoLanes {' '	last := len(same) - 1
	if true {'
mutate "depth tie takes the earlier edge" layout/branch.go '		if d := l.branchDepth(same[i].Dst); d > bd {' '		if d := l.branchDepth(same[i].Dst); d >= bd {'
mutate "depth includes shared flow after merge" layout/branch.go '	if l.nonBackIn(id) <= 1 {' '	if true {'
mutate "depth follows loop edges too" layout/branch.go '			if !x.Back {
				if k := 1 + l.branchDepth(x.Dst); k > d {' '			if true {
				if k := 1 + l.branchDepth(x.Dst); k > d {'
mutate "merge outside spine still returns to main column" layout/place.go 'ok && (!l.NoLanes || spine[vid]) {' 'ok {'
mutate "spine goes only one step" layout/branch.go '			id = same[l.mainEdge(same)].Dst
		}' '			_ = same
			break
		}'

# direction
mutate "LR direction does not swap node size" layout/axis.go '		it.W, it.H = it.H, it.W' '		_ = it'
mutate "LR direction does not swap label size" layout/axis.go '		e.LW, e.LH = e.LH, e.LW' '		_ = e'
mutate "BT flip also flips header" layout/axis.go '		return [2]float64{x, o.hdr + o.end - y}' '		return [2]float64{x, o.end - y}'
mutate "RL not flipped after axis swap" layout/axis.go '		return [2]float64{o.hdr + o.end - y, x}' '		return [2]float64{y, x}'
mutate "axis swap does not swap anchors" layout/axis.go '	case DirLR:
		return [2]float64{f[1], f[0]}' '	case DirLR:
		return f'
mutate "BT flip does not flip anchors" layout/axis.go '		return [2]float64{f[0], 1 - f[1]}' '		return f'
mutate "axis swap does not swap label offset" layout/axis.go '	case DirLR:
		return [2]float64{v[1], v[0]}' '	case DirLR:
		return v'
mutate "axis swap does not swap pool size" layout/axis.go '		out.PoolW, out.PoolH = r.PoolH, r.PoolW' '		_ = out'
mutate "axis swap forgets label box" layout/axis.go '			b := o.box(*e.Label)
			e.Label = &b' ''
mutate "horizontal lane drawn like vertical lane" writer/drawio/write.go 'func horizontalLanes(r layout.Result) bool { return r.Dir == layout.DirLR || r.Dir == layout.DirRL }' 'func horizontalLanes(r layout.Result) bool { return false }'
mutate "item in horizontal lane uses vertical lane coordinates" writer/drawio/write.go '			geo(c, it.X-float64(r.PoolHeader), it.Y-r.LaneX[it.Lane], it.W, it.H)' '			geo(c, it.X-r.LaneX[it.Lane], it.Y-float64(r.PoolHeader), it.W, it.H)'
mutate "merge accepts directions other than TD" build.go '	if opt.Previous != nil && dir != layout.DirTD {' '	if false {'
mutate "unknown direction not rejected" build.go '	if !layout.ValidDirection(dir) {' '	if false {'
mutate "source direction ignored" build.go '		dir = t.Direction' '		_ = t.Direction'

# config field declarations
mutate "default not taken from field declaration" layout/config.go '		*f.Get(&c) = f.Default' '		_ = f'
mutate "range not checked" layout/config.go '		if v := *f.Get(&c); v < f.Lo || v > f.Hi {' '		if false {'
mutate "range drops upper bound" layout/config.go '		if v := *f.Get(&c); v < f.Lo || v > f.Hi {' '		if v := *f.Get(&c); v < f.Lo {'
mutate "Fields returns the internal slice itself" layout/config.go '{ return append([]Field(nil), fields...) }' '{ return fields }'
mutate "web ignores layout parameters" cmd/flowcastd/server.go '		*f.Get(&cfg) = n' '		_ = n'
mutate "web ignores direction" cmd/flowcastd/server.go '		Direction: strings.TrimSpace(r.FormValue("direction"))}' '	}'
mutate "web accepts non-numeric parameter" cmd/flowcastd/server.go '		n, err := strconv.Atoi(v)
		if err != nil {' '		n, _ := strconv.Atoi(v)
		if false {'
mutate "web returns 422 for invalid config" cmd/flowcastd/server.go '	case code == "schema.out_of_range" || code == "config.direction" || code == "merge.direction":' '	case false:'
mutate "web does not declare directions" cmd/flowcastd/server.go '		"directions": []string{layout.DirTD, layout.DirBT, layout.DirLR, layout.DirRL},' '		"directions": []string{},'
mutate "web declares ranges incompletely" cmd/flowcastd/server.go '		out = append(out, apiField{f.Name, f.Help, f.Default, f.Lo, f.Hi})' '		out = append(out, apiField{f.Name, f.Help, f.Default, 0, 0})'

mutate "help does not list layout parameters" cmd/flowcast/args.go '	for _, f := range layout.Fields() {
		fmt.Fprintf(&b,' '	for _, f := range nil {
		fmt.Fprintf(&b,'
mutate "help only accepted in first position" cmd/flowcast/args.go '		case name == "help":
			return nil, errHelp' '		case false:
			return nil, errHelp'

[ -n "$PREFLIGHT" ] && exit 0
# Every mutation must be restored. A dirty tree means this tool itself is
# broken, and none of the results above can be trusted.
if ! git diff --quiet; then
	echo 'ERROR working tree still dirty after the run:'
	git status --short
	exit 1
fi
echo
echo "caught $caught, missed $missed"
[ "$missed" -eq 0 ]
