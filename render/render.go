// Package render calls the drawio CLI to export images, and checks whether
// draw.io draws the wires exactly at the computed coordinates.
//
// This package lives outside the core: it runs external processes and reads and
// writes files. The core only produces .drawio text; whoever needs images plugs
// this package in.
package render

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/luytbq/flowcast/layout"
)

// Error is an error that prevents exporting an image. The message is ready to be
// shown to the user.
type Error struct{ Msg string }

func (e *Error) Error() string { return e.Msg }

// Bin returns the path to the drawio CLI, or empty when the machine has none.
func Bin() string {
	for _, name := range []string{"drawio", "draw.io"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

// Timeout is the maximum duration of one drawio call. drawio is an Electron
// app, and the first run on a cold machine can take several tens of seconds.
var Timeout = 90 * time.Second

// Export exports src to out in the given format ("png" or "svg"). A scale of 0
// means no scale is set. On timeout it retries once.
func Export(ctx context.Context, exe, src, out, format string, scale int) error {
	if exe == "" {
		return &Error{"drawio CLI not found"}
	}
	args := []string{"-x", "-f", format, "-o", out}
	if scale != 0 {
		args = append(args, "-s", strconv.Itoa(scale))
	}
	args = append(args, src, "--no-sandbox")
	for attempt := 0; attempt < 2; attempt++ {
		cctx, cancel := context.WithTimeout(ctx, Timeout)
		var stdout, stderr bytes.Buffer
		cmd := exec.CommandContext(cctx, exe, args...)
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		timedOut := errors.Is(cctx.Err(), context.DeadlineExceeded)
		cancel()
		if timedOut {
			continue
		}
		if _, statErr := os.Stat(out); err != nil || statErr != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = strings.TrimSpace(stdout.String())
			}
			return &Error{"drawio export failed: " + msg}
		}
		return nil
	}
	return &Error{"drawio export timed out: " + strings.Join(append([]string{exe}, args...), " ")}
}

const svgNS = "http://www.w3.org/2000/svg"

// node is an SVG element, with just enough for the check: the tag in the svg
// namespace, the attributes, and the children in document order.
type node struct {
	tag      string // empty when the element is not in the svg namespace
	attrs    map[string]string
	children []*node
}

func (n *node) iter(f func(*node) bool) bool {
	if !f(n) {
		return false
	}
	for _, c := range n.children {
		if !c.iter(f) {
			return false
		}
	}
	return true
}

func parseSVG(data []byte) (*node, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	root := &node{}
	stack := []*node{root}
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
			n := &node{attrs: map[string]string{}}
			if t.Name.Space == svgNS {
				n.tag = t.Name.Local
			}
			for _, a := range t.Attr {
				if a.Name.Space == "" {
					n.attrs[a.Name.Local] = a.Value
				}
			}
			p := stack[len(stack)-1]
			p.children = append(p.children, n)
			stack = append(stack, n)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		}
	}
	return root, nil
}

var pathNum = regexp.MustCompile(`[-+]?\d*\.?\d+(?:e[-+]?\d+)?`)

func pathNums(d string) []float64 {
	var out []float64
	for _, s := range pathNum.FindAllString(d, -1) {
		v, _ := strconv.ParseFloat(s, 64)
		out = append(out, v)
	}
	return out
}

func attrFloat(n *node, k string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(n.attrs[k]), 64)
	return v
}

// Verify compares the wire paths in the SVG drawn by draw.io with the
// coordinates in r, with a tolerance of 2 pixels, and returns each mismatch.
//
// draw.io's SVG places the diagram at its own coordinate origin, so the offset
// is derived from the shape of the first element in the table. The last segment
// of each wire is compared only along its axis, because the arrowhead makes
// draw.io shorten that segment.
func Verify(r layout.Result, svg []byte) ([]string, error) {
	root, err := parseSVG(svg)
	if err != nil {
		return nil, err
	}
	cells := map[string]*node{}
	root.iter(func(n *node) bool {
		if id := n.attrs["data-cell-id"]; n.tag == "g" && id != "" {
			cells[id] = n
		}
		return true
	})
	if len(cells) == 0 {
		return []string{"SVG has no data-cell-id, skipping render check"}, nil
	}
	if len(r.Items) == 0 {
		return nil, nil
	}
	anchor := r.Items[0]
	for _, it := range r.Items[1:] {
		if it.Order < anchor.Order {
			anchor = it
		}
	}
	var rect *[2]float64
	if a := cells[anchor.ID]; a != nil {
		a.iter(func(n *node) bool {
			switch {
			case n.tag == "rect" && n.attrs["width"] != "":
				rect = &[2]float64{attrFloat(n, "x"), attrFloat(n, "y")}
			case n.tag == "ellipse":
				rect = &[2]float64{attrFloat(n, "cx") - attrFloat(n, "rx"), attrFloat(n, "cy") - attrFloat(n, "ry")}
			case n.tag == "path" && n.attrs["d"] != "":
				nums := pathNums(n.attrs["d"])
				mx, my := math.Inf(1), math.Inf(1)
				for i, v := range nums {
					if i%2 == 0 {
						mx = math.Min(mx, v)
					} else {
						my = math.Min(my, v)
					}
				}
				rect = &[2]float64{mx, my}
			default:
				return true
			}
			return false
		})
	}
	if rect == nil {
		return []string{"shape of " + anchor.ID + " not found in SVG, skipping render check"}, nil
	}
	dx, dy := rect[0]-anchor.X, rect[1]-anchor.Y
	const tol = 2.0
	var probs []string
	for _, e := range r.Edges {
		g := cells[e.ID]
		if g == nil {
			probs = append(probs, e.ID+": not in SVG")
			continue
		}
		var path *node
		g.iter(func(n *node) bool {
			if n.tag == "path" && n.attrs["d"] != "" && n.attrs["fill"] == "none" {
				path = n
				return false
			}
			return true
		})
		if path == nil {
			probs = append(probs, e.ID+": cannot read path")
			continue
		}
		nums := pathNums(path.attrs["d"])
		var got [][2]float64
		for i := 0; i+1 < len(nums); i += 2 {
			got = append(got, [2]float64{nums[i] - dx, nums[i+1] - dy})
		}
		want := e.Pts
		if len(got) != len(want) {
			probs = append(probs, fmt.Sprintf("%s: draw.io drew %d points, computed %d points", e.ID, len(got), len(want)))
			continue
		}
		for k := 0; k < len(got)-1; k++ {
			a, b := got[k], want[k]
			if math.Abs(a[0]-b[0]) > tol || math.Abs(a[1]-b[1]) > tol {
				probs = append(probs, fmt.Sprintf("%s: point %d off, drawn (%.0f,%.0f) instead of (%.0f,%.0f)",
					e.ID, k, a[0], a[1], b[0], b[1]))
				break
			}
		}
		n := len(want)
		lastG, lastW, prevW := got[n-1], want[n-1], want[n-2]
		var bad bool
		if math.Abs(lastW[0]-prevW[0]) < 0.01 {
			bad = math.Abs(lastG[0]-lastW[0]) > tol
		} else {
			bad = math.Abs(lastG[1]-lastW[1]) > tol
		}
		if bad {
			probs = append(probs, e.ID+": last segment off axis")
		}
	}
	return probs, nil
}
