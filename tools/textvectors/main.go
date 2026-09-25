// Command textvectors generates the text measuring and wrapping vectors.
//
//	go run ./tools/textvectors conformance/text-vectors.json
//
// The diagram case suite pins the text module too loosely. Wrap shrinks to the
// smallest width that keeps the same line count, so a budget a few pixels off
// usually leaves the output unchanged. The vectors here sweep the width one
// pixel at a time, so any deviation, even of one pixel, flips at least one row.
//
// The expected values come from the text and layout packages themselves: the
// vectors pin the current behavior against unintended change, they do not
// prove it right.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luytbq/flowcast/layout"
	"github.com/luytbq/flowcast/num"
	"github.com/luytbq/flowcast/text"
)

var samples = []string{
	"Run it",
	"Match the sale to the payment gate and then log the end result in our own ledger",
	"https://api.example.com/v1/transactions/reconcile?from=2026-01-01",
	"DBPAYMENTRECONCILIATIONFOREXTRADE",
	"Line one\nLine two | with a | bar",
	"a::b::c::d_e.f,g(h)i{j}k=l&m?n-o/p",
	"Validate the input data",
	"AAAA BBBB CCCC DDDD EEEE FFFF GGGG HHHH",
	// Characters outside the metrics table, to pin the fallback to the advance
	// of the .notdef glyph.
	"Task state 状態 プロセス",
}

// kinds are the element kinds whose wrap budget is pinned. The budget is a
// constant in SizeItem that the wrap shrink hides, so a ladder of growing
// lengths crosses every line-count boundary and moves the flip point as soon
// as the budget is off by even 2px.
var kinds = []string{"task", "condition", "start", "end", "external", "db", "text"}

// unknownKind checks the fallback budget for a kind no rule names.
const unknownKind = "unknown kind"

const lo, hi = 20, 300

func ladder() []string {
	var out []string
	for n := 1; n <= 30; n++ {
		out = append(out, strings.Repeat("abc ", n-1)+"abc")
	}
	for n := 1; n <= 20; n++ {
		out = append(out, strings.Repeat("ledger ", n-1)+"ledger")
	}
	for n := 1; n <= 60; n++ {
		out = append(out, strings.Repeat("x", n))
	}
	return out
}

// Fields are declared in key order so the output matches a sorted-key dump.
type width struct {
	H     string   `json:"h"`
	Hard  bool     `json:"hard"`
	Lines []string `json:"lines"`
	MaxW  int      `json:"maxw"`
	W     string   `json:"w"`
}

type sample struct {
	Text   string  `json:"text"`
	Widths []width `json:"widths"`
}

type sizeRow struct {
	H     string   `json:"h"`
	Lines []string `json:"lines"`
	Text  string   `json:"text"`
	W     string   `json:"w"`
}

type sizeKind struct {
	Kind string    `json:"kind"`
	Rows []sizeRow `json:"rows"`
}

type doc struct {
	Budgets map[string]int `json:"budgets"`
	LineH   float64        `json:"line_h"`
	Samples []sample       `json:"samples"`
	Size    float64        `json:"size"`
	Sizes   []sizeKind     `json:"sizes"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: textvectors OUT.json")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR", err)
		os.Exit(1)
	}
}

func run(out string) error {
	m, err := text.LoadMetrics(filepath.Join("data", "verdana.json"))
	if err != nil {
		return fmt.Errorf("cannot read data/verdana.json (run from the repo root): %w", err)
	}
	tm := text.NewMeasure(m)
	cfg := layout.DefaultConfig()
	d := doc{Budgets: map[string]int{}, LineH: tm.LineH, Size: tm.Size}

	nw := 0
	for _, s := range samples {
		lines := strings.Split(s, "\n")
		sv := sample{Text: s}
		for maxw := lo; maxw <= hi; maxw++ {
			got := tm.Wrap(lines, float64(maxw))
			w, h := tm.Box(got)
			sv.Widths = append(sv.Widths, width{
				MaxW: maxw, Lines: got, Hard: tm.Hard(), W: num.Fmt(w), H: num.Fmt(h),
			})
		}
		nw += len(sv.Widths)
		d.Samples = append(d.Samples, sv)
	}

	ns := 0
	for _, k := range kinds {
		sk := sizeKind{Kind: k}
		for _, content := range ladder() {
			got, w, h := layout.SizeItem(tm, cfg, k, []string{content})
			sk.Rows = append(sk.Rows, sizeRow{Text: content, Lines: got, W: num.Fmt(w), H: num.Fmt(h)})
		}
		ns += len(sk.Rows)
		d.Sizes = append(d.Sizes, sk)
	}

	for _, k := range append(append([]string{}, kinds...), unknownKind) {
		d.Budgets[k] = int(layout.WrapBudget(cfg, k))
	}
	d.Budgets["__label__"] = cfg.LabelWrap

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(d); err != nil {
		return err
	}
	if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Printf("%s: %d wrap vectors, %d size vectors, %d budgets\n", out, nw, ns, len(d.Budgets))
	return nil
}
