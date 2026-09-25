package source

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/luytbq/flowcast/model"
)

// Mermaid flowchart reader. It reduces the diagram to exactly the Table the three
// table formats produce, so validate, layout and writer do not know the input
// was mermaid.
//
// Mapping, see also section 12 of docs/core-design.md:
//
//	A[x] A(x) A[[x]]      task
//	A{x}                  condition; with only one outgoing edge it becomes task
//	A([x])                start if it has no incoming edge, end if it has no outgoing edge
//	A[(x)]                db, placed beside the only node it connects to
//	A((x))                external
//	A(((x)))              end
//	subgraph              lane
//	-.->  ==>  ---        style dashed, bold, noarrow
//
// Anything with no place in the Flow Table is reported as a warning-level Issue
// whose code starts with "mermaid.", with the line number in the source.

type mmNode struct {
	id      string
	text    string
	shape   string
	line    int
	order   int
	lane    string
	classes []string
	props   map[string]string
}

type mmEdge struct {
	from, to string
	text     string
	dashed   bool
	bold     bool
	noarrow  bool
	line     int
}

type mmLane struct {
	id, title string
	line      int
	order     int
}

type mermaid struct {
	nodes     map[string]*mmNode
	nodeOrder []string
	edges     []*mmEdge
	lanes     []*mmLane
	laneByID  map[string]*mmLane
	stack     []string // open subgraphs, outermost first
	classDefs map[string]map[string]string
	links     map[int]map[string]string // linkStyle by edge index
	linkAll   map[string]string
	issues    []model.Issue
	title     string
	dir       string
	seq       int // shared order of appearance of nodes and lanes
}

func (m *mermaid) warn(line int, code, msg string) {
	m.issues = append(m.issues, model.Issue{
		Code: code, Level: model.LevelWarning, Loc: model.Location{Kind: "text", Line: line}, Msg: msg})
}

// ParseMermaid reads a mermaid flowchart.
func ParseMermaid(data []byte, name string) (model.Table, error) {
	text := strings.TrimPrefix(string(data), "\ufeff")
	if !utf8.ValidString(text) {
		return model.Table{}, model.Errf("mermaid.encoding", "%s is not UTF-8", name)
	}
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	m := &mermaid{nodes: map[string]*mmNode{}, laneByID: map[string]*mmLane{},
		classDefs: map[string]map[string]string{}, links: map[int]map[string]string{}}
	lines := strings.Split(text, "\n")
	start := m.frontmatter(lines)
	header := false
	for i := start; i < len(lines); i++ {
		ln := i + 1
		raw := strings.TrimSpace(lines[i])
		if raw == "" {
			continue
		}
		if strings.HasPrefix(raw, "%%{") {
			m.warn(ln, "mermaid.directive", "ignoring config directive %%{...}%%")
			continue
		}
		if strings.HasPrefix(raw, "%%") {
			continue
		}
		for _, st := range splitStatements(raw) {
			if st = strings.TrimSpace(st); st == "" {
				continue
			}
			if !header {
				if err := m.header(st, ln, name); err != nil {
					return model.Table{}, err
				}
				header = true
				continue
			}
			m.statement(st, ln)
		}
	}
	if !header {
		return model.Table{}, model.Errf("mermaid.empty", "%s contains no mermaid diagram", name)
	}
	for _, id := range m.stack {
		m.warn(m.laneByID[id].line, "mermaid.unclosed_subgraph", "subgraph "+id+" has no end")
	}
	return m.table(), nil
}

// frontmatter reads the --- ... --- block at the start of the file, where mermaid declares the title.
func (m *mermaid) frontmatter(lines []string) int {
	i := 0
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i >= len(lines) || strings.TrimSpace(lines[i]) != "---" {
		return 0
	}
	for j := i + 1; j < len(lines); j++ {
		t := strings.TrimSpace(lines[j])
		if t == "---" {
			return j + 1
		}
		if v, ok := strings.CutPrefix(t, "title:"); ok {
			m.title = normalize(unquote(strings.TrimSpace(v)))
		}
	}
	return 0
}

func (m *mermaid) header(st string, ln int, name string) error {
	f := strings.Fields(st)
	if f[0] != "flowchart" && f[0] != "graph" {
		return model.Errf("mermaid.not_flowchart",
			"%s is not a mermaid flowchart: line %d starts with %q, only flowchart or graph is accepted", name, ln, f[0])
	}
	if len(f) > 1 {
		switch f[1] {
		case "TD", "TB":
			m.dir = "TD"
		case "LR", "RL", "BT":
			m.dir = f[1]
		default:
			m.warn(ln, "mermaid.direction", "direction "+f[1]+" is invalid, the diagram is drawn top-down")
		}
	}
	if len(f) > 2 {
		m.statement(strings.Join(f[2:], " "), ln)
	}
	return nil
}

// splitStatements splits a line at semicolons outside quotes and brackets.
func splitStatements(s string) []string {
	var out []string
	depth, quote, start := 0, false, 0
	for i, r := range s {
		switch {
		case r == '"':
			quote = !quote
		case quote:
		case strings.ContainsRune("[({", r):
			depth++
		case strings.ContainsRune("])}", r) && depth > 0:
			depth--
		case r == ';' && depth == 0:
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

func firstWord(s string) (string, string) {
	i := strings.IndexFunc(s, unicode.IsSpace)
	if i < 0 {
		return s, ""
	}
	return s[:i], strings.TrimSpace(s[i:])
}

func (m *mermaid) statement(st string, ln int) {
	word, rest := firstWord(st)
	switch word {
	case "subgraph":
		m.subgraph(rest, ln)
	case "end":
		if rest != "" {
			break
		}
		if len(m.stack) == 0 {
			m.warn(ln, "mermaid.syntax", "end does not close any subgraph")
			return
		}
		m.stack = m.stack[:len(m.stack)-1]
		return
	case "direction":
		m.warn(ln, "mermaid.direction", "ignoring direction inside a subgraph")
		return
	case "classDef":
		names, props := firstWord(rest)
		for _, n := range strings.Split(names, ",") {
			m.classDefs[strings.TrimSpace(n)] = parseProps(props)
		}
		return
	case "class":
		ids, cls := firstWord(rest)
		for _, id := range strings.Split(ids, ",") {
			n := m.node(strings.TrimSpace(id), ln)
			n.classes = append(n.classes, strings.TrimSpace(cls))
		}
		return
	case "style":
		id, props := firstWord(rest)
		n := m.node(id, ln)
		if n.props == nil {
			n.props = map[string]string{}
		}
		for k, v := range parseProps(props) {
			n.props[k] = v
		}
		return
	case "linkStyle":
		idx, props := firstWord(rest)
		p := parseProps(props)
		if idx == "default" {
			m.linkAll = p
			return
		}
		for _, s := range strings.Split(idx, ",") {
			k, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil {
				m.warn(ln, "mermaid.syntax", "linkStyle has an invalid index: "+s)
				continue
			}
			m.links[k] = p
		}
		return
	case "click", "accTitle", "accTitle:", "accDescr", "accDescr:":
		m.warn(ln, "mermaid.ignored", "ignoring "+strings.TrimSuffix(word, ":")+": it has no place in a draw.io diagram")
		return
	}
	if word == "subgraph" {
		return
	}
	m.chain(st, ln)
}

func (m *mermaid) subgraph(rest string, ln int) {
	id, title := rest, rest
	if i := strings.IndexByte(rest, '['); i > 0 && strings.HasSuffix(rest, "]") {
		id = strings.TrimSpace(rest[:i])
		title = strings.TrimSpace(rest[i+1 : len(rest)-1])
	}
	title = unquote(title)
	if id == "" || strings.HasPrefix(id, "\"") {
		id = unquote(id)
	}
	if id == "" {
		id = fmt.Sprintf("subgraph%d", len(m.lanes)+1)
	}
	if len(m.stack) > 0 {
		m.warn(ln, "mermaid.nested_subgraph",
			"subgraph "+id+" is nested in "+m.stack[0]+"; nested lanes are not supported, its nodes go into "+m.stack[0])
		m.stack = append(m.stack, id)
		return
	}
	if _, dup := m.laneByID[id]; !dup {
		m.seq++
		l := &mmLane{id: id, title: normalize(labelText(title)), line: ln, order: m.seq}
		m.lanes = append(m.lanes, l)
		m.laneByID[id] = l
	}
	m.stack = append(m.stack, id)
}

// node returns node id, creating it if it does not exist yet. A node belongs to
// the outermost open subgraph the first time it is mentioned inside a subgraph.
func (m *mermaid) node(id string, ln int) *mmNode {
	n, ok := m.nodes[id]
	if !ok {
		m.seq++
		n = &mmNode{id: id, text: id, shape: "rect", line: ln, order: m.seq}
		m.nodes[id] = n
		m.nodeOrder = append(m.nodeOrder, id)
	}
	if n.lane == "" && len(m.stack) > 0 {
		n.lane = m.stack[0]
	}
	return n
}

// ---- chains of nodes and edges

type cursor struct {
	s string
	i int
}

func (c *cursor) skip() {
	for c.i < len(c.s) && (c.s[c.i] == ' ' || c.s[c.i] == '\t') {
		c.i++
	}
}

func (c *cursor) rest() string { return c.s[c.i:] }
func (c *cursor) done() bool   { c.skip(); return c.i >= len(c.s) }

func isIDRune(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) }

func (m *mermaid) chain(st string, ln int) {
	c := &cursor{s: st}
	from, ok := m.group(c, ln)
	if !ok {
		m.warn(ln, "mermaid.syntax", "unrecognized statement, ignored: "+st)
		return
	}
	for !c.done() {
		lk, ok := m.link(c, ln)
		if !ok {
			m.warn(ln, "mermaid.syntax", "unrecognized edge syntax at \""+c.rest()+"\", dropping the rest of the statement")
			return
		}
		to, ok := m.group(c, ln)
		if !ok {
			m.warn(ln, "mermaid.syntax", "edge is missing its target node at \""+c.rest()+"\"")
			return
		}
		for _, a := range from {
			for _, b := range to {
				if lk.invisible {
					continue
				}
				e := lk.edge
				e.from, e.to, e.line = a, b, ln
				m.edges = append(m.edges, &e)
			}
		}
		from = to
	}
}

func (m *mermaid) group(c *cursor, ln int) ([]string, bool) {
	var ids []string
	for {
		id, ok := m.parseNode(c, ln)
		if !ok {
			return nil, false
		}
		ids = append(ids, id)
		c.skip()
		if c.i < len(c.s) && c.s[c.i] == '&' {
			c.i++
			continue
		}
		return ids, true
	}
}

// shapes in trial order: longer openers before shorter ones, so "((" is not read as "(".
var shapes = []struct{ open, close, name string }{
	{"(((", ")))", "double"},
	{"((", "))", "circle"},
	{"([", "])", "stadium"},
	{"[(", ")]", "cylinder"},
	{"[[", "]]", "subroutine"},
	{"{{", "}}", "hexagon"},
	{"[/", "/]", "parallelogram"},
	{"[\\", "\\]", "parallelogram"},
	{"[/", "\\]", "trapezoid"},
	{"[\\", "/]", "trapezoid"},
	{"(", ")", "round"},
	{"[", "]", "rect"},
	{"{", "}", "rhombus"},
	{">", "]", "asymmetric"},
}

func (m *mermaid) parseNode(c *cursor, ln int) (string, bool) {
	c.skip()
	start := c.i
	for c.i < len(c.s) {
		r, w := utf8.DecodeRuneInString(c.s[c.i:])
		if isIDRune(r) {
			c.i += w
			continue
		}
		// A hyphen or dot between two id characters, as in API-1 or E4.1, is
		// neither an arrow nor a terminator.
		if (r == '-' || r == '.') && c.i > start && c.i+1 < len(c.s) {
			if r2, _ := utf8.DecodeRuneInString(c.s[c.i+1:]); isIDRune(r2) {
				c.i++
				continue
			}
		}
		break
	}
	if c.i == start {
		return "", false
	}
	id := c.s[start:c.i]
	n := m.node(id, ln)
	if strings.HasPrefix(c.rest(), "@{") {
		end := strings.IndexByte(c.rest(), '}')
		if end < 0 {
			return "", false
		}
		m.warn(ln, "mermaid.shape", "@{...} syntax of "+id+" is not supported, drawn as task")
		c.i += end + 1
		return id, true
	}
	for _, sh := range shapes {
		if !strings.HasPrefix(c.rest(), sh.open) {
			continue
		}
		body := c.s[c.i+len(sh.open):]
		var text string
		var used int
		if t := strings.TrimLeft(body, " "); strings.HasPrefix(t, "\"") {
			q := strings.IndexByte(t[1:], '"')
			if q < 0 {
				return "", false
			}
			after := strings.TrimLeft(t[q+2:], " ")
			if !strings.HasPrefix(after, sh.close) {
				continue
			}
			text = t[1 : q+1]
			used = len(body) - len(after) + len(sh.close)
		} else {
			end := strings.Index(body, sh.close)
			if end < 0 {
				continue
			}
			text = strings.TrimSpace(body[:end])
			used = end + len(sh.close)
		}
		c.i += len(sh.open) + used
		n.text = normalize(labelText(text))
		n.shape = sh.name
		break
	}
	if strings.HasPrefix(c.rest(), ":::") {
		c.i += 3
		s := c.i
		for c.i < len(c.s) {
			r, w := utf8.DecodeRuneInString(c.s[c.i:])
			if !isIDRune(r) && r != '-' {
				break
			}
			c.i += w
		}
		n.classes = append(n.classes, c.s[s:c.i])
	}
	return id, true
}

type link struct {
	edge      mmEdge
	invisible bool
}

// link reads an arrow: --> --- -.-> ==> ~~~, their longer variants, o or x
// heads, bidirectional arrows, and labels written inside the arrow or in |...|.
func (m *mermaid) link(c *cursor, ln int) (link, bool) {
	c.skip()
	s := c.rest()
	var lk link
	bidir := false
	if strings.HasPrefix(s, "<") {
		bidir = true
		s = s[1:]
	}
	consumed := len(c.rest()) - len(s)
	head := func(t string) (string, int) {
		if t == "" {
			return "", 0
		}
		switch t[0] {
		case '>':
			return ">", 1
		case 'o', 'x':
			// o and x are arrow heads only when not attached to a following id.
			if len(t) == 1 || !isIDRune(rune(t[1])) {
				return string(t[0]), 1
			}
		}
		return "", 0
	}
	run := func(t string, ch byte) int {
		n := 0
		for n < len(t) && t[n] == ch {
			n++
		}
		return n
	}
	switch {
	case strings.HasPrefix(s, "~~~"):
		n := run(s, '~')
		c.i += consumed + n
		lk.invisible = true
		m.warn(ln, "mermaid.invisible_link", "ignoring invisible link ~~~: it only guides mermaid's placement")
		m.label(c, &lk)
		return lk, true
	case strings.HasPrefix(s, "-."):
		lk.edge.dashed = true
		dots := run(s[1:], '.')
		t := s[1+dots:]
		if strings.HasPrefix(t, "-") {
			h, hn := head(t[1:])
			c.i += consumed + 1 + dots + 1 + hn
			m.finish(&lk, h, bidir, ln)
			m.label(c, &lk)
			return lk, true
		}
		// -. label .->
		end := strings.Index(t, ".-")
		if end < 0 {
			return lk, false
		}
		lk.edge.text = strings.TrimSpace(t[:end])
		after := t[end+1:]
		dn := run(after, '-')
		h, hn := head(after[dn:])
		c.i += consumed + 1 + dots + end + 1 + dn + hn
		m.finish(&lk, h, bidir, ln)
		m.label(c, &lk)
		return lk, true
	case strings.HasPrefix(s, "--") || strings.HasPrefix(s, "=="):
		ch := s[0]
		lk.edge.bold = ch == '='
		n := run(s, ch)
		h, hn := head(s[n:])
		if n >= 3 || h != "" {
			c.i += consumed + n + hn
			m.finish(&lk, h, bidir, ln)
			m.label(c, &lk)
			return lk, true
		}
		// -- label --> and == label ==>
		t := s[n:]
		end := strings.Index(t, string([]byte{ch, ch}))
		if end < 0 {
			return lk, false
		}
		lk.edge.text = strings.TrimSpace(t[:end])
		after := t[end:]
		dn := run(after, ch)
		h, hn = head(after[dn:])
		c.i += consumed + n + end + dn + hn
		m.finish(&lk, h, bidir, ln)
		m.label(c, &lk)
		return lk, true
	}
	return lk, false
}

func (m *mermaid) finish(lk *link, head string, bidir bool, ln int) {
	switch head {
	case "":
		lk.edge.noarrow = true
	case "o", "x":
		m.warn(ln, "mermaid.arrow_head", "arrow head "+head+" is not supported, drawn as a plain arrow")
	}
	if bidir {
		m.warn(ln, "mermaid.bidirectional", "bidirectional arrow drawn as one-directional")
	}
}

// label reads a |...| label right after the arrow.
func (m *mermaid) label(c *cursor, lk *link) {
	c.skip()
	if !strings.HasPrefix(c.rest(), "|") {
		return
	}
	end := strings.IndexByte(c.rest()[1:], '|')
	if end < 0 {
		return
	}
	lk.edge.text = strings.TrimSpace(c.rest()[1 : end+1])
	c.i += end + 2
}

// ---- label text

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

var entities = map[string]string{"quot": "\"", "amp": "&", "lt": "<", "gt": ">", "nbsp": " ",
	"apos": "'", "semi": ";", "num": "#"}

// labelText strips mermaid label syntax: quotes, markdown strings `...`, bold
// and italic tags, and character codes #quot; #35;. br tags are kept for line
// splitting later.
func labelText(s string) string {
	s = unquote(strings.TrimSpace(s))
	if len(s) >= 2 && s[0] == '`' && s[len(s)-1] == '`' {
		s = s[1 : len(s)-1]
		s = strings.ReplaceAll(strings.ReplaceAll(s, "**", ""), "__", "")
		s = strings.ReplaceAll(s, "\n", "<br>")
	}
	for _, tag := range []string{"b", "i", "u", "strong", "em"} {
		s = strings.ReplaceAll(strings.ReplaceAll(s, "<"+tag+">", ""), "</"+tag+">", "")
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '#' {
			if j := strings.IndexByte(s[i:], ';'); j > 1 && j < 12 {
				code := s[i+1 : i+j]
				if v, ok := entities[code]; ok {
					b.WriteString(v)
					i += j + 1
					continue
				}
				if n, err := strconv.Atoi(code); err == nil && n > 0 && n <= unicode.MaxRune {
					b.WriteRune(rune(n))
					i += j + 1
					continue
				}
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func lines(text string) []string {
	if text == "" {
		return []string{""}
	}
	parts := splitBR(text)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// parseProps reads style properties like fill:#f9f,stroke:#333.
func parseProps(s string) map[string]string {
	out := map[string]string{}
	for _, p := range strings.Split(strings.TrimSuffix(strings.TrimSpace(s), ";"), ",") {
		k, v, ok := strings.Cut(p, ":")
		if ok {
			out[strings.ToLower(strings.TrimSpace(k))] = strings.ToLower(strings.TrimSpace(v))
		}
	}
	return out
}

// The writer's highlight colors. A node or edge painted exactly this color gets
// style highlight; other colors have no place, because style carries a role, not
// a color.
const (
	highlightFill   = "#dae8fc"
	highlightStroke = "#6c8ebf"
)

func sortedKeys(p map[string]string) []string {
	var ks []string
	for k := range p {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// ---- building the table

func (m *mermaid) table() model.Table {
	if m.linkAll != nil {
		for i := range m.edges {
			if _, ok := m.links[i]; !ok {
				m.links[i] = m.linkAll
			}
		}
	}
	// A node with the same id as a subgraph means an edge to the whole subgraph,
	// which has no place in the Flow Table.
	var edges []*mmEdge
	for _, e := range m.edges {
		if m.laneByID[e.from] != nil || m.laneByID[e.to] != nil {
			m.warn(e.line, "mermaid.edge_to_subgraph", "dropping edge "+e.from+" --> "+e.to+": edges to a subgraph are not supported")
			continue
		}
		edges = append(edges, e)
	}
	var order []string
	for _, id := range m.nodeOrder {
		if m.laneByID[id] == nil {
			order = append(order, id)
		}
	}

	ins, outs := map[string][]*mmEdge{}, map[string][]*mmEdge{}
	for _, e := range edges {
		outs[e.from] = append(outs[e.from], e)
		ins[e.to] = append(ins[e.to], e)
	}
	types := map[string]string{}
	attach := map[string]string{}
	dropped := map[*mmEdge]bool{}
	for _, id := range order {
		types[id] = m.nodeType(m.nodes[id], ins[id], outs[id], attach, dropped)
	}
	// Edges of a db replaced by attach are no longer edges.
	var kept []*mmEdge
	for _, e := range edges {
		if !dropped[e] {
			kept = append(kept, e)
		}
	}
	ins, outs = map[string][]*mmEdge{}, map[string][]*mmEdge{}
	for _, e := range kept {
		outs[e.from] = append(outs[e.from], e)
		ins[e.to] = append(ins[e.to], e)
	}
	for _, id := range order {
		if types[id] == "condition" && len(outs[id]) < 2 {
			types[id] = "task"
			m.warn(m.nodes[id].line, "mermaid.condition_one_branch",
				fmt.Sprintf("%s is a diamond but has %d outgoing edges, drawn as task", id, len(outs[id])))
		}
	}

	laneOf := m.assignLanes(order)
	ids := m.rowIDs(order)
	flow := orderFlow(order, types, kept, ins, outs, attach)

	var rows []model.Row
	add := func(r model.Row) {
		r.Idx = len(rows)
		rows = append(rows, r)
	}
	loc := func(line int) model.Location { return model.Location{Kind: "text", Line: line} }
	for _, l := range m.lanes {
		add(model.Row{ID: l.id, Type: "lane", Lines: lines(l.title), Loc: loc(l.line), Meta: map[string]string{}})
	}
	pos := map[string]int{}
	for i, id := range flow.nodes {
		pos[id] = i
	}
	edgeIdx := map[*mmEdge]int{}
	for i, e := range m.edges {
		edgeIdx[e] = i
	}
	edgeIDs := map[*mmEdge]string{}
	seen := map[string]int{}
	for _, e := range kept {
		base := ids[e.from] + "-->" + ids[e.to]
		seen[base]++
		if seen[base] > 1 {
			base += "#" + strconv.Itoa(seen[base])
		}
		edgeIDs[e] = base
	}
	for _, id := range flow.nodes {
		n := m.nodes[id]
		r := model.Row{ID: ids[id], Type: types[id], Parent: laneOf[id], Lines: lines(n.text), Loc: loc(n.line),
			Meta: map[string]string{}}
		if a := attach[id]; a != "" {
			r.Meta["attach"], r.MetaKeys = ids[a], []string{"attach"}
		}
		if m.nodeHighlight(n) {
			r.Meta["style"] = "highlight"
			r.MetaKeys = append(r.MetaKeys, "style")
		}
		add(r)
		for _, a := range flow.attached[id] {
			an := m.nodes[a]
			ar := model.Row{ID: ids[a], Type: "db", Parent: laneOf[a], Lines: lines(an.text), Loc: loc(an.line),
				Meta: map[string]string{"attach": ids[id]}, MetaKeys: []string{"attach"}}
			if m.nodeHighlight(an) {
				ar.Meta["style"] = "highlight"
				ar.MetaKeys = append(ar.MetaKeys, "style")
			}
			add(ar)
		}
		for _, e := range outs[id] {
			r := model.Row{ID: edgeIDs[e], Type: "edge", Lines: lines(normalize(labelText(e.text))), Loc: loc(e.line),
				Meta: map[string]string{"from": ids[e.from], "to": ids[e.to]}, MetaKeys: []string{"from", "to"}}
			var styles []string
			p := m.links[edgeIdx[e]]
			if e.dashed || p["stroke-dasharray"] != "" {
				styles = append(styles, "dashed")
			}
			if e.bold {
				styles = append(styles, "bold")
			}
			if e.noarrow {
				styles = append(styles, "noarrow")
			}
			if p["stroke"] == highlightStroke {
				styles = append(styles, "highlight")
			} else if len(p) > 0 {
				m.warn(e.line, "mermaid.style", "ignoring linkStyle of edge "+edgeIDs[e]+": "+strings.Join(sortedKeys(p), ", "))
			}
			if len(styles) > 0 {
				r.Meta["style"] = strings.Join(styles, ",")
				r.MetaKeys = append(r.MetaKeys, "style")
			}
			if pos[e.to] <= pos[e.from] {
				r.Meta["back"] = "true"
				r.MetaKeys = append(r.MetaKeys, "back")
			}
			add(r)
		}
	}
	// Warnings are produced in processing order, not line order; readers need
	// line order.
	sort.SliceStable(m.issues, func(i, j int) bool { return m.issues[i].Loc.Line < m.issues[j].Loc.Line })
	return model.Table{Title: m.title, Rows: rows, Issues: m.issues, Source: "mermaid", Direction: m.dir}
}

// nodeType infers the type from the node's shape. A db connected to exactly one
// node is placed beside that node, and the edges connecting it are dropped.
func (m *mermaid) nodeType(n *mmNode, ins, outs []*mmEdge, attach map[string]string, dropped map[*mmEdge]bool) string {
	in, out := 0, 0
	for _, e := range ins {
		if e.from != n.id {
			in++
		}
	}
	for _, e := range outs {
		if e.to != n.id {
			out++
		}
	}
	switch n.shape {
	case "rect", "round", "subroutine":
		return "task"
	case "rhombus":
		return "condition"
	case "circle":
		return "external"
	case "double":
		return "end"
	case "stadium":
		switch {
		case in == 0:
			return "start"
		case out == 0:
			return "end"
		}
		m.warn(n.line, "mermaid.shape", n.id+" has a start/end shape but sits mid-flow, drawn as task")
		return "task"
	case "cylinder":
		peers := map[string]bool{}
		var touching []*mmEdge
		for _, e := range append(append([]*mmEdge{}, ins...), outs...) {
			if e.from == n.id && e.to == n.id {
				continue
			}
			touching = append(touching, e)
			if e.from == n.id {
				peers[e.to] = true
			} else {
				peers[e.from] = true
			}
		}
		if len(peers) != 1 {
			m.warn(n.line, "mermaid.db_shape",
				fmt.Sprintf("db %s connects to %d nodes; a db sits beside exactly one node, so it is drawn as task", n.id, len(peers)))
			return "task"
		}
		var peer string
		for p := range peers {
			peer = p
		}
		if m.nodes[peer].shape == "cylinder" {
			m.warn(n.line, "mermaid.db_shape", "db "+n.id+" only connects to another db, drawn as task")
			return "task"
		}
		for _, e := range touching {
			dropped[e] = true
			if e.text != "" {
				m.warn(e.line, "mermaid.db_edge_label", "dropping label \""+e.text+"\" of the edge to db "+n.id)
			}
		}
		attach[n.id] = peer
		m.warn(n.line, "mermaid.db_attach", "db "+n.id+" placed beside "+peer+" instead of a connecting arrow")
		return "db"
	}
	m.warn(n.line, "mermaid.shape", "shape "+n.shape+" of "+n.id+" is not supported, drawn as task")
	return "task"
}

func (m *mermaid) nodeHighlight(n *mmNode) bool {
	props := map[string]string{}
	if d, ok := m.classDefs["default"]; ok {
		for k, v := range d {
			props[k] = v
		}
	}
	for _, c := range n.classes {
		for k, v := range m.classDefs[c] {
			props[k] = v
		}
	}
	for k, v := range n.props {
		props[k] = v
	}
	if len(props) == 0 {
		return false
	}
	if props["fill"] == highlightFill {
		return true
	}
	m.warn(n.line, "mermaid.style", "ignoring style of "+n.id+": "+strings.Join(sortedKeys(props), ", ")+
		"; only the highlight color "+highlightFill+" has a place")
	return false
}

// assignLanes returns the lane of each node. When the diagram has subgraphs,
// nodes outside every subgraph are grouped into an unnamed lane.
func (m *mermaid) assignLanes(order []string) map[string]string {
	out := map[string]string{}
	if len(m.lanes) == 0 {
		return out
	}
	var outside []string
	first := 0
	for _, id := range order {
		n := m.nodes[id]
		if n.lane == "" {
			if len(outside) == 0 {
				first = n.order
			}
			outside = append(outside, id)
			continue
		}
		out[id] = n.lane
	}
	if len(outside) == 0 {
		return out
	}
	const loose = "_"
	l := &mmLane{id: loose, line: m.nodes[outside[0]].line, order: first}
	m.lanes = append(m.lanes, l)
	sort.SliceStable(m.lanes, func(i, j int) bool { return m.lanes[i].order < m.lanes[j].order })
	for _, id := range outside {
		out[id] = loose
	}
	m.warn(l.line, "mermaid.outside_subgraph",
		fmt.Sprintf("%d nodes are outside every subgraph, grouped into an unnamed lane: %s", len(outside), strings.Join(outside, ", ")))
	return out
}

// rowIDs renames ids that collide with ids reserved by draw.io and the writer.
func (m *mermaid) rowIDs(order []string) map[string]string {
	out := map[string]string{}
	for _, id := range order {
		out[id] = id
		if id == "0" || id == "1" || id == "pool" {
			out[id] = "n_" + id
			m.warn(m.nodes[id].line, "mermaid.renamed_id", "id "+id+" collides with a draw.io reserved id, renamed to n_"+id)
		}
	}
	return out
}

type flowOrder struct {
	nodes    []string            // nodes in row order, excluding db placed beside a node
	attached map[string][]string // db placed beside each node
}

// orderFlow orders nodes by the reading order of the Flow Table spec: follow the
// flow from the entry points, each node immediately followed by its outgoing
// edges, and a merge node is written only after all of its sources that are not
// loop edges have appeared.
//
// Loop edges are determined first, by a DFS in the same order: an edge pointing
// back to a node on the current path. This way the merge condition never waits
// for a source reachable only through that very node.
func orderFlow(order []string, types map[string]string, edges []*mmEdge,
	ins, outs map[string][]*mmEdge, attach map[string]string) flowOrder {
	fo := flowOrder{attached: map[string][]string{}}
	var nodes []string
	for _, id := range order {
		if types[id] == "db" {
			fo.attached[attach[id]] = append(fo.attached[attach[id]], id)
			continue
		}
		nodes = append(nodes, id)
	}
	// Roots: start first, then nodes with no incoming edge, in declaration order.
	var roots []string
	for _, id := range nodes {
		if types[id] == "start" {
			roots = append(roots, id)
		}
	}
	for _, id := range nodes {
		if types[id] != "start" && len(ins[id]) == 0 {
			roots = append(roots, id)
		}
	}
	back := map[*mmEdge]bool{}
	state := map[string]int{} // 1: on the current path, 2: done
	var dfs func(string)
	dfs = func(u string) {
		state[u] = 1
		for _, e := range outs[u] {
			switch state[e.to] {
			case 0:
				dfs(e.to)
			case 1:
				back[e] = true
			}
		}
		state[u] = 2
	}
	for _, id := range append(append([]string{}, roots...), nodes...) {
		if state[id] == 0 {
			dfs(id)
		}
	}
	written := map[string]bool{}
	ready := func(id string) bool {
		for _, e := range ins[id] {
			if !back[e] && e.from != id && !written[e.from] {
				return false
			}
		}
		return true
	}
	var visit func(string)
	visit = func(u string) {
		written[u] = true
		fo.nodes = append(fo.nodes, u)
		for _, e := range outs[u] {
			if !written[e.to] && ready(e.to) {
				visit(e.to)
			}
		}
	}
	for _, r := range roots {
		if !written[r] && ready(r) {
			visit(r)
		}
	}
	// The remainder, such as loops with no entry point: take the first ready
	// node in declaration order. Such a node always exists, because without loop
	// edges the graph has no cycles.
	for len(fo.nodes) < len(nodes) {
		for _, id := range nodes {
			if !written[id] && ready(id) {
				visit(id)
				break
			}
		}
	}
	return fo
}
