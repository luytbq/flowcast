package flowcast

import (
	"time"

	"github.com/luytbq/flowcast/model"
	"github.com/luytbq/flowcast/source"
)

// Limits caps the resources one build may use. A value of 0 means no cap.
//
// The row count is the primary cap, because it also bounds run time: build time
// grows roughly with the square of the element count, 1000 elements take about
// 0.1 seconds and 3000 elements about 0.9 seconds. Timeout is a secondary cap,
// only checked between phases rather than interrupting a phase.
type Limits struct {
	MaxBytes    int   // input size
	MaxUnzipped int64 // total bytes decompressed from one xlsx file
	MaxRows     int   // table row count, including lanes and edges
	MaxEdges    int
	Timeout     time.Duration
}

// Two built-in profiles. The CLI runs on the user's own machine, so it is
// loose; the web service accepts files from strangers, so it is strict.
var (
	CLILimits = Limits{MaxBytes: 64 << 20, MaxUnzipped: 512 << 20, MaxRows: 50000, MaxEdges: 50000}
	WebLimits = Limits{MaxBytes: 1 << 20, MaxUnzipped: 16 << 20, MaxRows: 1000, MaxEdges: 1000,
		Timeout: 10 * time.Second}
)

// budget tracks one build against Limits.
type budget struct {
	lim   Limits
	start time.Time
}

func newBudget(l *Limits) budget {
	if l == nil {
		return budget{start: time.Now()}
	}
	return budget{lim: *l, start: time.Now()}
}

func (b budget) source(src Source) (Source, error) {
	if b.lim.MaxBytes > 0 && len(src.Data) > b.lim.MaxBytes {
		return src, model.Errf("limit.bytes", "file is %d bytes, over the limit of %d bytes", len(src.Data), b.lim.MaxBytes)
	}
	if b.lim.MaxUnzipped > 0 && (src.MaxUnzipped == 0 || src.MaxUnzipped > b.lim.MaxUnzipped) {
		src.MaxUnzipped = b.lim.MaxUnzipped
	}
	return src, nil
}

func (b budget) table(t model.Table) error {
	if b.lim.MaxRows > 0 && len(t.Rows) > b.lim.MaxRows {
		return model.Errf("limit.rows", "table has %d rows, over the limit of %d rows", len(t.Rows), b.lim.MaxRows)
	}
	edges := 0
	for _, r := range t.Rows {
		if r.Type == "edge" {
			edges++
		}
	}
	if b.lim.MaxEdges > 0 && edges > b.lim.MaxEdges {
		return model.Errf("limit.edges", "table has %d edges, over the limit of %d edges", edges, b.lim.MaxEdges)
	}
	return b.check()
}

// check reports a timeout. Call it between phases.
func (b budget) check() error {
	if b.lim.Timeout > 0 && time.Since(b.start) > b.lim.Timeout {
		return model.Errf("limit.timeout", "build exceeded %v, stopped", b.lim.Timeout)
	}
	return nil
}

func parseWithin(src Source, b budget) (model.Table, error) {
	src, err := b.source(src)
	if err != nil {
		return model.Table{}, err
	}
	t, err := source.Parse(src)
	if err != nil {
		return model.Table{}, err
	}
	return t, b.table(t)
}
