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
	"time"

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
	Trace   = layout.Trace
)

// Since formats the time a step took, for the end of a trace line.
func Since(t0 time.Time) string { return layout.Since(t0) }

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
	// Trace, when set, receives one line per processing step: what the step
	// did, with counts and timing. The CLI prints these lines under --verbose.
	Trace Trace
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
	t, err := readAndTrace(src, newBudget(opt.Limits), opt.Trace)
	if err != nil {
		return Result{}, err
	}
	issues := validateAndTrace(t, opt.Trace)
	return Result{Title: t.Title, Source: t.Source, Issues: issues}, nil
}

// readAndTrace parses the source and logs what it found.
func readAndTrace(src Source, b budget, trace Trace) (model.Table, error) {
	t0 := time.Now()
	t, err := parseWithin(src, b)
	if err != nil {
		trace.Log("source: %s: %v", src.Name, err)
		return t, err
	}
	var lanes, nodes, edges int
	for _, r := range t.Rows {
		switch r.Type {
		case "lane":
			lanes++
		case "edge":
			edges++
		default:
			nodes++
		}
	}
	trace.Log("source: read %d bytes of %s as %s, title %q, direction %q: %d rows (%d lanes, %d elements, %d edges), %d issues %s",
		len(src.Data), src.Name, t.Source, t.Title, t.Direction, len(t.Rows), lanes, nodes, edges, len(t.Issues), layout.Since(t0))
	return t, nil
}

// validateAndTrace validates a parsed table and logs the outcome.
func validateAndTrace(t model.Table, trace Trace) []Issue {
	t0 := time.Now()
	issues := append(append([]Issue{}, t.Issues...), validate.Validate(t.Rows)...)
	ne := 0
	for _, i := range issues {
		if i.Level == model.LevelError {
			ne++
		}
	}
	trace.Log("validate: %d errors, %d warnings %s", ne, len(issues)-ne, layout.Since(t0))
	return issues
}

// Build reads, validates and builds the diagram.
func Build(src Source, opt Options) (Result, error) {
	t0 := time.Now()
	b := newBudget(opt.Limits)
	t, err := readAndTrace(src, b, opt.Trace)
	if err != nil {
		return Result{}, err
	}
	r := Result{Title: t.Title, Source: t.Source}
	r.Issues = validateAndTrace(t, opt.Trace)
	if opt.Title != "" {
		r.Title = opt.Title
	}
	if r.HasErrors() {
		opt.Trace.Log("build: stopped, the table has errors")
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
	mode := "new"
	if opt.Previous != nil {
		mode = "merge"
	}
	opt.Trace.Log("config: direction %s, %s, optimize %d rounds", dir, mode, cfg.Optimize)
	if err := layoutAndWrite(&r, t.Rows, cfg, m, dir, opt.Previous, b, opt.Trace); err != nil {
		opt.Trace.Log("build: failed: %v", err)
		return Result{}, err
	}
	opt.Trace.Log("build: done: %d lanes, %d elements, %d edges, %.2fx%.2f px, %d bytes %s",
		r.Stats.Lanes, r.Stats.Items, r.Stats.Edges, r.Stats.W, r.Stats.H, len(r.Text), layout.Since(t0))
	return r, nil
}

// layoutAndWrite lays out a validated table and produces the target text.
//
// Merge and the writer can still panic on a broken invariant; that error is
// returned like any other, so programs using the library and the web service do
// not crash on an unusual file.
func layoutAndWrite(r *Result, rows []model.Row, cfg layout.Config, m *text.Metrics, dir string, prev *merge.Old, b budget, trace Trace) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = model.Errf("layout.internal", "internal layout error: %v", p)
		}
	}()
	laid, err := layout.Lay(rows, text.NewMeasure(m), layout.Options{
		Config: cfg, Direction: dir, Trace: trace, Checkpoint: b.check})
	if err != nil {
		return err
	}
	res := laid.Result
	r.Warnings, r.Findings, r.Layout = laid.Warnings, laid.Findings, &res
	t0 := time.Now()
	if prev != nil {
		extras, rep := merge.Apply(&res, prev)
		r.Merge = &rep
		trace.Log("merge: kept %d positions, placed %d new elements, kept %d wires, left %d wires to draw.io, kept %d freehand cells %s",
			len(rep.Pinned), len(rep.Placed), len(rep.KeptEdges), len(rep.Rerouted), len(rep.FreehandKept), layout.Since(t0))
		t0 = time.Now()
		r.Text = drawio.WriteMerged(res, r.Title, extras, prev.Pages)
	} else {
		r.Text = drawio.Write(res, r.Title)
	}
	trace.Log("write: %d bytes of draw.io XML %s", len(r.Text), layout.Since(t0))
	lanes := len(res.Lanes)
	if res.NoLanes {
		lanes = 0
	}
	r.Stats = Stats{lanes, len(res.Items), len(res.Edges), res.PoolW, res.PoolH}
	return nil
}
