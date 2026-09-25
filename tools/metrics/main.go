// Command metrics measures layout quality over a set of diagrams, so that layout
// heuristics are judged by numbers rather than by feel.
//
//	go run ./tools/metrics conformance/mermaid/*.mmd conformance/flowchart/*.md
//
// For each diagram it prints: the number of wires per routing case (A straight
// vertical, B straight horizontal, C L-shaped, D detour through a channel), the
// total number of bend points, the number of places where two wires cross, the
// total wire length, the size, the number
// of self-check findings, and the number of main-path edges drawn straight vertical
// out of the total number of main-path edges. The main path is the longest path
// from a start point to an end point that does not go through a loop edge.
package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/luytbq/flowcast"
	"github.com/luytbq/flowcast/layout"
)

type row struct {
	cases              [4]int
	bends, crossings   int
	length, w, h       float64
	findings           int
	spineA, spineTotal int
}

func measure(r layout.Result, findings int) row {
	var m row
	m.w, m.h, m.findings = r.PoolW, r.PoolH, findings
	for _, e := range r.Edges {
		if e.Case >= 'A' && e.Case <= 'D' {
			m.cases[e.Case-'A']++
		}
		if len(e.Pts) > 2 {
			m.bends += len(e.Pts) - 2
		}
		for i := 1; i < len(e.Pts); i++ {
			m.length += math.Abs(e.Pts[i][0]-e.Pts[i-1][0]) + math.Abs(e.Pts[i][1]-e.Pts[i-1][1])
		}
	}
	m.crossings = crossings(r)
	spine := longestPath(r)
	m.spineTotal = len(spine)
	for _, e := range spine {
		if e.Case == 'A' {
			m.spineA++
		}
	}
	return m
}

// crossings counts the pairs of wire segments from two different edges that cross.
// A vertical and a horizontal segment crossing is where the reader has to stop and
// trace which wire goes where, so fewer is better.
func crossings(r layout.Result) int {
	type seg struct {
		edge           int
		x0, y0, x1, y1 float64
	}
	var segs []seg
	for i, e := range r.Edges {
		for k := 1; k < len(e.Pts); k++ {
			a, b := e.Pts[k-1], e.Pts[k]
			segs = append(segs, seg{i, math.Min(a[0], b[0]), math.Min(a[1], b[1]),
				math.Max(a[0], b[0]), math.Max(a[1], b[1])})
		}
	}
	n := 0
	for i, a := range segs {
		for _, b := range segs[i+1:] {
			if a.edge == b.edge {
				continue
			}
			horizA, horizB := a.y0 == a.y1, b.y0 == b.y1
			if horizA == horizB {
				continue // two parallel segments overlap rather than cross
			}
			h, v := a, b
			if horizB {
				h, v = b, a
			}
			if v.x0 > h.x0 && v.x0 < h.x1 && h.y0 > v.y0 && h.y0 < v.y1 {
				n++
			}
		}
	}
	return n
}

// longestPath returns the edges of the longest path, by edge count, in the graph
// with loop edges removed. Ties go to the path met first in row order.
func longestPath(r layout.Result) []layout.PlacedEdge {
	outs := map[string][]layout.PlacedEdge{}
	ins := map[string]int{}
	for _, e := range r.Edges {
		if !e.Back {
			outs[e.Src] = append(outs[e.Src], e)
			ins[e.Dst]++
		}
	}
	memo := map[string][]layout.PlacedEdge{}
	var best func(string) []layout.PlacedEdge
	best = func(id string) []layout.PlacedEdge {
		if p, ok := memo[id]; ok {
			return p
		}
		var out []layout.PlacedEdge
		for _, e := range outs[id] {
			if p := best(e.Dst); len(p)+1 > len(out) {
				out = append([]layout.PlacedEdge{e}, p...)
			}
		}
		memo[id] = out
		return out
	}
	var out []layout.PlacedEdge
	for _, it := range r.Items {
		if ins[it.ID] == 0 {
			if p := best(it.ID); len(p) > len(out) {
				out = p
			}
		}
	}
	return out
}

func main() {
	var total row
	fmt.Printf("%-34s %4s %4s %4s %4s %5s %5s %7s %9s %4s %7s\n", "diagram", "A", "B", "C", "D", "bends", "cross", "length", "size", "chk", "main")
	for _, p := range os.Args[1:] {
		data, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		res, err := flowcast.Build(flowcast.Source{Name: filepath.Base(p), Data: data}, flowcast.Options{})
		if err != nil || res.Layout == nil {
			fmt.Fprintf(os.Stderr, "%s: build failed: %v\n", p, err)
			os.Exit(1)
		}
		m := measure(*res.Layout, len(res.Findings))
		fmt.Printf("%-34s %4d %4d %4d %4d %5d %5d %7.0f %4.0fx%-4.0f %4d %4d/%-2d\n", filepath.Base(p),
			m.cases[0], m.cases[1], m.cases[2], m.cases[3], m.bends, m.crossings, m.length, m.w, m.h,
			m.findings, m.spineA, m.spineTotal)
		for i := range total.cases {
			total.cases[i] += m.cases[i]
		}
		total.bends += m.bends
		total.crossings += m.crossings
		total.length += m.length
		total.findings += m.findings
		total.spineA += m.spineA
		total.spineTotal += m.spineTotal
	}
	fmt.Printf("%-34s %4d %4d %4d %4d %5d %5d %7.0f %9s %4d %4d/%-2d\n", "total", total.cases[0], total.cases[1],
		total.cases[2], total.cases[3], total.bends, total.crossings, total.length, "", total.findings,
		total.spineA, total.spineTotal)
}
