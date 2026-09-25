// Package flowcast turns a flow diagram written as a table into a draw.io file
// with a deterministic layout.
//
// Build is the only entry point. It does not touch the filesystem, does not run
// external processes and prints nothing: bytes in, target text plus diagnostics
// out. That way the same code path serves both the CLI reading files and the web
// service receiving uploads. See docs/core-design.md.
package flowcast

import (
	"sync"

	"github.com/luytbq/flowcast/data"
	"github.com/luytbq/flowcast/layout"
	"github.com/luytbq/flowcast/merge"
	"github.com/luytbq/flowcast/model"
	"github.com/luytbq/flowcast/source"
	"github.com/luytbq/flowcast/text"
	"github.com/luytbq/flowcast/validate"
	"github.com/luytbq/flowcast/writer/drawio"
)

type (
	Source  = source.Source
	Issue   = model.Issue
	Finding = layout.Finding
	Warning = layout.Warning
)

// Options are the parameters of one build.
type Options struct {
	// Config holds the layout parameters. The zero value means layout.DefaultConfig().
	Config *layout.Config
	// Title replaces the title taken from the source. Empty keeps the source title.
	Title string
	// Previous is the .drawio file generated last time, possibly with manual
	// edits. When non-nil, the new diagram keeps those edits, see package merge.
	// Previous can only be used once: its pages are moved to the new file.
	Previous *merge.Old
	// Direction is the diagram direction: TD, BT, LR or RL. Empty follows the
	// direction the source declares, and TD if the source declares none.
	Direction string
	// Limits caps resources, see CLILimits and WebLimits. nil means no caps.
	// Exceeding a limit returns an error whose model.Error code starts with
	// "limit.".
	Limits *Limits
}

// Stats is the size of the built diagram.
type Stats struct {
	Lanes int     `json:"lanes"`
	Items int     `json:"items"`
	Edges int     `json:"edges"`
	W     float64 `json:"width"`
	H     float64 `json:"height"`
}

// Result is the result of one build.
type Result struct {
	Title  string
	Source string // format that was read, e.g. "markdown"
	// Issues are findings about the input table, from both the reading and the
	// validation layers.
	Issues []Issue
	// Text is the content of the .drawio file. Empty when the table has errors.
	Text string
	// Warnings are the layout engine's warnings, Findings are the results of the
	// geometry self-check. Both are empty when the table has errors.
	Warnings []Warning
	Findings []Finding
	Layout   *layout.Result
	// Merge is the merge report when Options.Previous is non-nil. Findings are
	// then the self-check of the freshly computed layout, before merge adjusts it.
	Merge *merge.Report
	Stats Stats
}

// HasErrors reports whether the table has errors that block the build.
func (r Result) HasErrors() bool {
	for _, i := range r.Issues {
		if i.Level == model.LevelError {
			return true
		}
	}
	return false
}

var (
	metricsOnce sync.Once
	metrics     *text.Metrics
	metricsErr  error
)

func defaultMetrics() (*text.Metrics, error) {
	metricsOnce.Do(func() { metrics, metricsErr = text.ParseMetrics(data.Verdana) })
	return metrics, metricsErr
}

// Check reads and validates the table without building the diagram.
//
// The returned error is one that stops reading from continuing, such as a
// missing header. Errors of individual rows are in Result.Issues.
func Check(src Source, opt Options) (Result, error) {
	t, err := parseWithin(src, newBudget(opt.Limits))
	if err != nil {
		return Result{}, err
	}
	issues := append(append([]Issue{}, t.Issues...), validate.Validate(t.Rows)...)
	return Result{Title: t.Title, Source: t.Source, Issues: issues}, nil
}

// Build reads, validates and builds the diagram.
func Build(src Source, opt Options) (Result, error) {
	b := newBudget(opt.Limits)
	t, err := parseWithin(src, b)
	if err != nil {
		return Result{}, err
	}
	r := Result{Title: t.Title, Source: t.Source}
	r.Issues = append(append([]Issue{}, t.Issues...), validate.Validate(t.Rows)...)
	if opt.Title != "" {
		r.Title = opt.Title
	}
	if r.HasErrors() {
		return r, nil
	}
	m, err := defaultMetrics()
	if err != nil {
		return Result{}, err
	}
	cfg := layout.DefaultConfig()
	if opt.Config != nil {
		cfg = *opt.Config
	}
	if err := layout.ValidateConfig(cfg); err != nil {
		return Result{}, err
	}
	if err := b.check(); err != nil {
		return Result{}, err
	}
	dir := opt.Direction
	if dir == "" {
		dir = t.Direction
	}
	if dir == "" {
		dir = layout.DirTD
	}
	if !layout.ValidDirection(dir) {
		return Result{}, model.Errf("config.direction", "invalid direction %q; use TD, BT, LR or RL", dir)
	}
	if opt.Previous != nil && dir != layout.DirTD {
		return Result{}, model.Errf("merge.direction",
			"merge does not support direction %s yet; use --mode force to regenerate everything", dir)
	}
	if err := layoutAndWrite(&r, t.Rows, cfg, m, dir, opt.Previous, b); err != nil {
		return Result{}, err
	}
	return r, nil
}

// layoutAndWrite lays out, self-checks and produces the target text for a
// validated table.
//
// The engine only panics when it violates one of its own invariants. That error
// is returned like any other, so programs using the library and the web service
// do not crash on an unusual table.
func layoutAndWrite(r *Result, rows []model.Row, cfg layout.Config, m *text.Metrics, dir string, prev *merge.Old, b budget) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = model.Errf("layout.internal", "internal layout error: %v", p)
		}
	}()
	lay := layout.New(rows, cfg, text.NewMeasure(m))
	lay.SetDirection(dir)
	l := lay.Run()
	if err := b.check(); err != nil {
		return err
	}
	res := l.Result()
	r.Warnings = l.Warnings
	r.Findings = layout.Check(res)
	res = layout.Orient(res)
	r.Layout = &res
	if prev != nil {
		extras, rep := merge.Apply(&res, prev)
		r.Merge = &rep
		r.Text = drawio.WriteMerged(res, r.Title, extras, prev.Pages)
	} else {
		r.Text = drawio.Write(res, r.Title)
	}
	lanes := len(res.Lanes)
	if res.NoLanes {
		lanes = 0
	}
	r.Stats = Stats{lanes, len(res.Items), len(res.Edges), res.PoolW, res.PoolH}
	return nil
}
