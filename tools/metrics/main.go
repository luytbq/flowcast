// Lệnh metrics đo chất lượng bố cục trên một bộ sơ đồ, để heuristic xếp hình
// được đánh giá bằng số liệu chứ không bằng cảm nhận.
//
//	go run ./tools/metrics conformance/mermaid/*.mmd conformance/flowchart/*.md
//
// Mỗi sơ đồ in ra: số dây mỗi kiểu đi dây (A thẳng đứng, B thẳng ngang, C chữ
// L, D đi vòng qua kênh), tổng số điểm gấp, số chỗ hai dây cắt nhau, tổng chiều
// dài dây, kích thước, số
// phát hiện tự kiểm, và số cạnh của đường chính được vẽ thẳng đứng trên tổng
// số cạnh của đường chính. Đường chính là đường dài nhất từ một điểm đầu tới
// một điểm cuối, không đi qua cạnh vòng lặp.
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

// crossings đếm số cặp đoạn dây của hai cạnh khác nhau cắt nhau. Một đoạn dọc
// và một đoạn ngang cắt nhau là chỗ người đọc phải dừng lại để lần xem dây nào
// đi đâu, nên càng ít càng tốt.
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
				continue // hai đoạn cùng phương thì chồng nhau chứ không cắt nhau
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

// longestPath trả về các cạnh của đường dài nhất theo số cạnh trên đồ thị đã
// bỏ cạnh vòng lặp. Hòa thì lấy đường gặp trước theo thứ tự dòng.
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
	fmt.Printf("%-34s %4s %4s %4s %4s %5s %5s %7s %9s %4s %7s\n", "sơ đồ", "A", "B", "C", "D", "gấp", "cắt", "dài", "cỡ", "tk", "chính")
	for _, p := range os.Args[1:] {
		data, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		res, err := flowcast.Build(flowcast.Source{Name: filepath.Base(p), Data: data}, flowcast.Options{})
		if err != nil || res.Layout == nil {
			fmt.Fprintf(os.Stderr, "%s: không dựng được: %v\n", p, err)
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
	fmt.Printf("%-34s %4d %4d %4d %4d %5d %5d %7.0f %9s %4d %4d/%-2d\n", "tổng", total.cases[0], total.cases[1],
		total.cases[2], total.cases[3], total.bends, total.crossings, total.length, "", total.findings,
		total.spineA, total.spineTotal)
}
