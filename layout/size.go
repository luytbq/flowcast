package layout

import (
	"github.com/luytbq/flowcast/internal/unistr"
	"strings"

	"github.com/luytbq/flowcast/num"
	"github.com/luytbq/flowcast/text"
)

// WrapBudget is the wrap width of an element type.
//
// It is split out as a named function because behavior cannot pin it down: Wrap
// shrinks to the smallest width that keeps the same line count, so a budget off
// by a few pixels only shows when a character boundary falls exactly within
// that gap. Pinning the number directly does not depend on luck.
//
// Unrecognized types use the note budget.
func WrapBudget(cfg Config, kind string) float64 {
	switch kind {
	case "task":
		// Subtract the left and right padding of the task box.
		return float64(cfg.TaskMaxW - 32)
	case "condition":
		return float64(cfg.CondWrap)
	case "start", "end", "external":
		return float64(cfg.TermWrap)
	case "db":
		return float64(cfg.DBWrap)
	}
	return float64(cfg.TextWrap)
}

// SizeItem wraps an element's content and returns the size of its box.
//
// It depends only on the element type, content and config, not on where the
// element sits in the diagram. That is why it runs before any placement step.
func SizeItem(tm *text.Measure, cfg Config, kind string, lines []string) (wrapped []string, w, h float64) {
	switch kind {
	case "task":
		wl := tm.Wrap(lines, WrapBudget(cfg, kind))
		tw, th := tm.Box(wl)
		return wl, float64(num.Rnd(max(float64(cfg.TaskMinW), tw+32))), float64(num.Rnd(max(50, th+20)))

	case "condition":
		wl := tm.Wrap(lines, WrapBudget(cfg, kind))
		tw, th := tm.Box(wl)
		// Text fits inside the diamond when tw/W + th/H <= 1.
		return wl, float64(num.Rnd(max(100, tw/0.6+10))), float64(num.Rnd(max(60, th/0.4+10)))

	case "start", "end", "external":
		wl := tm.Wrap(lines, WrapBudget(cfg, kind))
		tw, th := tm.Box(wl)
		// end has an extra outer border, so it needs more room for the same content.
		pad := 0.0
		if kind == "end" {
			pad = 12
		}
		return wl, float64(num.Rnd(max(80, float64(tw*1.42)+24+pad))), float64(num.Rnd(max(44, float64(th*1.42)+16+pad)))

	case "db":
		wl := tm.Wrap(lines, WrapBudget(cfg, kind))
		tw, th := tm.Box(wl)
		return wl, float64(num.Rnd(max(100, tw+24))), float64(num.Rnd(max(60, th+34)))
	}

	wl := tm.Wrap(lines, WrapBudget(cfg, kind))
	tw, th := tm.Box(wl)
	return wl, float64(num.Rnd(tw + 20)), float64(num.Rnd(th + 12))
}

// SizeLabel measures an edge's label.
//
// An edge without a label still gets a size, not zero: the width equals the
// padding on both sides and the height is 2. The reference implementation adds
// the padding outside the empty-string check, so an empty edge comes out 8 x 2
// rather than 0 x 0.
func SizeLabel(tm *text.Measure, cfg Config, lines []string) (wrapped []string, w, h float64) {
	var wl []string
	if HasText(lines) {
		wl = tm.Wrap(lines, float64(cfg.LabelWrap))
	}
	var lw, lh float64
	if len(wl) > 0 {
		lw, lh = tm.Box(wl)
	}
	return wl, lw + float64(2*cfg.LabelPad), lh + 2
}

// HasText reports whether a table row's content has real text: join the lines
// with newlines, then trim whitespace.
func HasText(lines []string) bool {
	return unistr.Strip(strings.Join(lines, "\n")) != ""
}
