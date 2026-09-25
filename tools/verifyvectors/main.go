// Command verifyvectors generates the vectors for the render check.
//
//	go run ./tools/verifyvectors conformance/verify-vectors.json
//
// The render check exports SVG with the drawio CLI and compares the wires
// draw.io drew with the computed coordinates. The comparison is a pure function
// of the SVG, so it can be pinned without drawio at test time: this command
// exports real SVG for a few cases, distorts it in one way at a time, and
// records what render.Verify reports. The intact SVG of every case must report
// nothing; the command fails otherwise instead of recording a broken baseline.
//
// The SVG is trimmed to svg, g, rect, ellipse and path, because the render check
// reads nothing else and the full SVG weighs tens of KB.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/luytbq/flowcast"
	"github.com/luytbq/flowcast/layout"
	"github.com/luytbq/flowcast/render"
)

const svgNS = "http://www.w3.org/2000/svg"

var cases = []string{
	"01-linear", "03-condition-two", "05-merge-node", "06-back-edge", "11-db-attach",
	"15-route-l-shape", "16-route-general", "17-track-sharing", "21-styles", "22-many-lanes",
	"23-external", "28-tracks-fan-in", "62-route-c-attach-side", "66-route-back-no-bottom",
}

// Same pattern render.Verify uses to read path coordinates, so a nudge lands on
// the number the check will read.
var pathNum = regexp.MustCompile(`[-+]?\d*\.?\d+(?:e[-+]?\d+)?`)

type attr struct{ space, local, value string }

type elem struct {
	space, local string
	attrs        []attr
	children     []*elem
}

func (e *elem) is(local string) bool { return e.space == svgNS && e.local == local }

func (e *elem) get(k string) string {
	for _, a := range e.attrs {
		if a.space == "" && a.local == k {
			return a.value
		}
	}
	return ""
}

func (e *elem) has(k string) bool {
	for _, a := range e.attrs {
		if a.space == "" && a.local == k {
			return true
		}
	}
	return false
}

func (e *elem) set(k, v string) {
	for i, a := range e.attrs {
		if a.space == "" && a.local == k {
			e.attrs[i].value = v
			return
		}
	}
	e.attrs = append(e.attrs, attr{local: k, value: v})
}

func (e *elem) del(k string) {
	for i, a := range e.attrs {
		if a.space == "" && a.local == k {
			e.attrs = append(e.attrs[:i], e.attrs[i+1:]...)
			return
		}
	}
}

// iter visits e and its descendants in document order.
func (e *elem) iter(f func(*elem)) {
	f(e)
	for _, c := range e.children {
		c.iter(f)
	}
}

func (e *elem) all() []*elem {
	var out []*elem
	e.iter(func(x *elem) { out = append(out, x) })
	return out
}

func (e *elem) clone() *elem {
	c := &elem{space: e.space, local: e.local, attrs: append([]attr(nil), e.attrs...)}
	for _, ch := range e.children {
		c.children = append(c.children, ch.clone())
	}
	return c
}

func newElem(space, local string, kv ...string) *elem {
	e := &elem{space: space, local: local}
	for i := 0; i+1 < len(kv); i += 2 {
		e.attrs = append(e.attrs, attr{local: kv[i], value: kv[i+1]})
	}
	return e
}

// parse reads the SVG and keeps only the elements the render check reads.
// Namespace declarations are dropped; write declares what the tree still uses.
func parse(data []byte) (*elem, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	var root *elem
	var stack []*elem
	skip := 0
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if skip > 0 {
				skip++
				continue
			}
			e := &elem{space: t.Name.Space, local: t.Name.Local}
			for _, a := range t.Attr {
				if a.Name.Space == "xmlns" || (a.Name.Space == "" && a.Name.Local == "xmlns") {
					continue
				}
				e.attrs = append(e.attrs, attr{a.Name.Space, a.Name.Local, a.Value})
			}
			if root == nil {
				root = e
			} else {
				keep := e.space == svgNS && (e.local == "g" || e.local == "rect" || e.local == "ellipse" || e.local == "path")
				if !keep {
					skip = 1
					continue
				}
				p := stack[len(stack)-1]
				p.children = append(p.children, e)
			}
			stack = append(stack, e)
		case xml.EndElement:
			if skip > 0 {
				skip--
				continue
			}
			stack = stack[:len(stack)-1]
		}
	}
	if root == nil || !root.is("svg") {
		return nil, errors.New("not an SVG document")
	}
	return root, nil
}

var wellKnown = map[string]string{
	"http://www.w3.org/1999/xlink":         "xlink",
	"http://www.w3.org/XML/1998/namespace": "xml",
}

// write serializes the tree with every namespace declared on the root under a
// generated prefix, the way Python's ElementTree does, so the output stays
// comparable with vectors written by earlier generators.
func write(root *elem) string {
	prefix := map[string]string{}
	n := 0
	name := func(space, local string) string {
		if space == "" {
			return local
		}
		p, ok := prefix[space]
		if !ok {
			if p, ok = wellKnown[space]; !ok {
				p = "ns" + strconv.Itoa(n)
				n++
			}
			prefix[space] = p
		}
		return p + ":" + local
	}
	root.iter(func(e *elem) {
		name(e.space, e.local)
		for _, a := range e.attrs {
			name(a.space, a.local)
		}
	})
	type decl struct{ p, uri string }
	var decls []decl
	for uri, p := range prefix {
		decls = append(decls, decl{p, uri})
	}
	sort.Slice(decls, func(i, j int) bool { return decls[i].p < decls[j].p })

	var b strings.Builder
	var emit func(e *elem, top bool)
	emit = func(e *elem, top bool) {
		b.WriteString("<" + name(e.space, e.local))
		if top {
			for _, d := range decls {
				fmt.Fprintf(&b, ` xmlns:%s="%s"`, d.p, escAttr(d.uri))
			}
		}
		for _, a := range e.attrs {
			fmt.Fprintf(&b, ` %s="%s"`, name(a.space, a.local), escAttr(a.value))
		}
		if len(e.children) == 0 {
			b.WriteString(" />")
			return
		}
		b.WriteString(">")
		for _, c := range e.children {
			emit(c, false)
		}
		b.WriteString("</" + name(e.space, e.local) + ">")
	}
	emit(root, true)
	return b.String()
}

var attrEsc = strings.NewReplacer(
	"&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;",
	"\r", "&#13;", "\n", "&#10;", "\t", "&#09;",
)

func escAttr(s string) string { return attrEsc.Replace(s) }

func cells(root *elem) map[string]*elem {
	out := map[string]*elem{}
	root.iter(func(e *elem) {
		if id := e.get("data-cell-id"); e.is("g") && id != "" {
			out[id] = e
		}
	})
	return out
}

// edgePath finds the wire the same way render.Verify does: the first path with
// a d and no fill. The arrowhead is a filled path in the same group.
func edgePath(g *elem) *elem {
	for _, e := range g.all() {
		if e.is("path") && e.get("d") != "" && e.get("fill") == "none" {
			return e
		}
	}
	return nil
}

// nudge adds delta to the k-th number in a path's d attribute.
func nudge(d string, k int, delta float64) string {
	loc := pathNum.FindAllStringIndex(d, -1)
	v, _ := strconv.ParseFloat(d[loc[k][0]:loc[k][1]], 64)
	return d[:loc[k][0]] + strconv.FormatFloat(v+delta, 'f', -1, 64) + d[loc[k][1]:]
}

func isShape(e *elem) bool { return e.is("rect") || e.is("ellipse") || e.is("path") }

func removeShapes(a *elem) {
	for _, e := range a.all() {
		kept := e.children[:0]
		for _, c := range e.children {
			if !isShape(c) {
				kept = append(kept, c)
			}
		}
		e.children = kept
	}
}

type variant struct {
	name string
	root *elem
}

var names = []string{
	"middle point offset", "end point offset", "offset below threshold", "extra point",
	"edge removed", "path removed", "anchor shape offset", "anchor shape removed",
	"no id", "duplicate id", "anchor rect missing width", "anchor rect outside namespace",
	"anchor drawn as path",
}

func variants(rng *rand.Rand, root *elem, lay layout.Result) []variant {
	out := []variant{{"unchanged", root}}
	anchor := lay.Items[0]
	for _, it := range lay.Items[1:] {
		if it.Order < anchor.Order {
			anchor = it
		}
	}
	pick := func(xs ...float64) float64 { return xs[rng.IntN(len(xs))] }
	for _, name := range names {
		r := root.clone()
		cs := cells(r)
		e := lay.Edges[rng.IntN(len(lay.Edges))]
		g := cs[e.ID]
		if g == nil {
			continue
		}
		p := edgePath(g)
		if p == nil {
			continue
		}
		n := len(pathNum.FindAllString(p.get("d"), -1))
		a := cs[anchor.ID]
		switch name {
		case "middle point offset":
			// A path shorter than three points has no middle point.
			if n < 6 {
				continue
			}
			k := int(pick(0, 1, 2, 3))
			p.set("d", nudge(p.get("d"), k, pick(-6, 3, 40)))
		case "end point offset":
			k := int(pick(float64(n-2), float64(n-1)))
			p.set("d", nudge(p.get("d"), k, pick(-5, 9)))
		case "offset below threshold":
			p.set("d", nudge(p.get("d"), rng.IntN(n), 1.5))
		case "extra point":
			p.set("d", p.get("d")+" L 1 2")
		case "edge removed":
			for _, x := range r.all() {
				for i, c := range x.children {
					if c == g {
						x.children = append(x.children[:i], x.children[i+1:]...)
						break
					}
				}
			}
		case "path removed":
			p.set("fill", "black")
		case "anchor shape offset":
			for _, el := range a.all() {
				if el.is("rect") && el.get("width") != "" {
					v, _ := strconv.ParseFloat(el.get("x"), 64)
					el.set("x", strconv.FormatFloat(v+3, 'f', -1, 64))
					break
				}
				if el.is("ellipse") {
					v, _ := strconv.ParseFloat(el.get("cx"), 64)
					el.set("cx", strconv.FormatFloat(v+3, 'f', -1, 64))
					break
				}
				if el.is("path") && el.get("d") != "" {
					el.set("d", nudge(el.get("d"), 0, 3))
					break
				}
			}
		case "anchor shape removed":
			removeShapes(a)
		case "anchor rect missing width":
			// A rect without width is a background or hit area, not a node shape.
			a.children = append([]*elem{newElem(svgNS, "rect", "x", "999", "y", "999")}, a.children...)
		case "anchor rect outside namespace":
			a.children = append([]*elem{newElem("urn:x-other", "rect", "x", "999", "y", "999", "width", "10")}, a.children...)
		case "anchor drawn as path":
			removeShapes(a)
			// The path does not start at the top left corner: the anchor is the
			// smallest corner, not the first point.
			a.children = append(a.children, newElem(svgNS, "path", "d", "M 200 300 L 100 300 L 100 200 L 200 200 Z"))
		case "no id":
			r.iter(func(x *elem) {
				if x.is("g") && x.has("data-cell-id") {
					x.del("data-cell-id")
				}
			})
		case "duplicate id":
			dup := g.clone()
			dp := edgePath(dup)
			dp.set("d", nudge(dp.get("d"), 0, 25))
			r.children = append(r.children, dup)
		}
		out = append(out, variant{name, r})
	}
	return out
}

// export runs the drawio CLI, retrying because it occasionally fails for
// reasons unrelated to the input, such as GPU initialization on a cold start.
func export(src, out string) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		os.Remove(out)
		if err = render.Export(context.Background(), render.Bin(), src, out, "svg", 0); err == nil {
			return nil
		}
		fmt.Fprintf(os.Stderr, "retry %s: %v\n", filepath.Base(src), err)
	}
	return err
}

type vector struct {
	Case, Variant, SVG string
	Problems           []string
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: verifyvectors OUT.json")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR", err)
		os.Exit(1)
	}
}

func run(out string) error {
	if render.Bin() == "" {
		return errors.New("drawio CLI not found")
	}
	tmp, err := os.MkdirTemp("", "verifyvectors")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	rng := rand.New(rand.NewPCG(41, 0))
	var vs []vector
	for _, c := range cases {
		data, err := os.ReadFile(filepath.Join("conformance", "cases", c+".md"))
		if err != nil {
			return fmt.Errorf("%w (run from the repo root)", err)
		}
		res, err := flowcast.Build(flowcast.Source{Name: c + ".md", Data: data}, flowcast.Options{})
		if err != nil || res.Layout == nil {
			return fmt.Errorf("%s: build failed: %v", c, err)
		}
		src := filepath.Join(tmp, c+".drawio")
		if err := os.WriteFile(src, []byte(res.Text), 0o644); err != nil {
			return err
		}
		svgPath := filepath.Join(tmp, c+".svg")
		if err := export(src, svgPath); err != nil {
			return fmt.Errorf("%s: %w", c, err)
		}
		raw, err := os.ReadFile(svgPath)
		if err != nil {
			return err
		}
		root, err := parse(raw)
		if err != nil {
			return fmt.Errorf("%s: %w", c, err)
		}
		for _, v := range variants(rng, root, *res.Layout) {
			svg := write(v.root)
			probs, err := render.Verify(*res.Layout, []byte(svg))
			if err != nil {
				return fmt.Errorf("%s/%s: %w", c, v.name, err)
			}
			if v.name == "unchanged" && len(probs) > 0 {
				return fmt.Errorf("%s: intact SVG reports problems: %q", c, probs)
			}
			vs = append(vs, vector{c, v.name, svg, probs})
		}
	}
	if err := os.WriteFile(out, encode(vs), 0o644); err != nil {
		return err
	}
	bad := 0
	for _, v := range vs {
		if len(v.Problems) > 0 {
			bad++
		}
	}
	fmt.Printf("%s: %d vectors, %d with problems\n", out, len(vs), bad)
	return nil
}

// encode writes one key or list item per line with no indentation, keeping
// the file diffable line by line even though each SVG is a single long string.
func encode(vs []vector) []byte {
	var b bytes.Buffer
	b.WriteString("[\n")
	for i, v := range vs {
		if i > 0 {
			b.WriteString(",\n")
		}
		fmt.Fprintf(&b, "{\n\"case\": %s,\n\"variant\": %s,\n\"svg\": %s,\n\"problems\": ", str(v.Case), str(v.Variant), str(v.SVG))
		if len(v.Problems) == 0 {
			b.WriteString("[]")
		} else {
			b.WriteString("[\n")
			for j, p := range v.Problems {
				if j > 0 {
					b.WriteString(",\n")
				}
				b.WriteString(str(p))
			}
			b.WriteString("\n]")
		}
		b.WriteString("\n}")
	}
	b.WriteString("\n]")
	return b.Bytes()
}

func str(s string) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return strings.TrimSuffix(b.String(), "\n")
}
