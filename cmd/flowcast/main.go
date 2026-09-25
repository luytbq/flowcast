// Command flowcast reads a Flow Table and writes a .drawio file.
//
// Printed lines, exit codes and flag names are an interface that scripts and
// agents rely on: they read the exit code and copy the ERROR layout:, ERROR
// render: and merge: lines verbatim. They are therefore pinned by CLI
// transcripts and change only on purpose.
//
// Exit codes:
//
//	0  ok, possibly with warnings
//	1  the table has errors, nothing is drawn
//	2  the geometry self-check has errors
//	3  the rendered image deviates from the coordinates, or the drawio CLI failed
//	4  the target file exists and no mode was chosen, or the target file cannot be read
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unicode"

	"github.com/luytbq/flowcast"
	"github.com/luytbq/flowcast/internal/unistr"
	"github.com/luytbq/flowcast/merge"
	"github.com/luytbq/flowcast/model"
	"github.com/luytbq/flowcast/num"
	"github.com/luytbq/flowcast/render"
	"github.com/luytbq/flowcast/source"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(argv []string, stdin io.Reader, stdout, stderr io.Writer) int {
	a, err := parseArgs(argv)
	if errors.Is(err, errHelp) {
		fmt.Fprint(stdout, helpText())
		return 0
	}
	if err != nil {
		fmt.Fprintln(stderr, "flowcast:", err)
		fmt.Fprintln(stderr, "usage: flowcast check <file> | flowcast build <file> [-o out.drawio] [--mode merge|force] [--direction LR] ...")
		return 2
	}
	c := &cli{a: a, stdin: stdin, out: stdout}
	if a.cmd == "check" {
		return c.check()
	}
	return c.build()
}

type cli struct {
	a     *args
	stdin io.Reader
	out   io.Writer
}

func (c *cli) println(s ...any) { fmt.Fprintln(c.out, s...) }

// supported lists the readable file extensions, checked before opening the file,
// as the reference implementation does: an unknown extension is rejected even if
// the file does not exist.
var supported = map[string]bool{".md": true, ".markdown": true, ".txt": true,
	".csv": true, ".tsv": true, ".xlsx": true, ".xlsm": true, ".mmd": true, ".mermaid": true}

// read reads the input file into a Source. Returned errors are already in printable form.
func (c *cli) read() (flowcast.Source, error) {
	ext := unistr.Lower(source.Ext(c.a.file))
	if !supported[ext] {
		return flowcast.Source{}, fmt.Errorf("unsupported file extension \"%s\"; use .md, .csv, .xlsx or .mmd", ext)
	}
	data, err := os.ReadFile(c.a.file)
	if err != nil {
		return flowcast.Source{}, fmt.Errorf("cannot read %s: %s", c.a.file, strerror(err))
	}
	opts := map[string]string{}
	for k, v := range map[string]string{"sheet": c.a.sheet, "delimiter": c.a.delimiter, "encoding": c.a.encoding} {
		if v != "" {
			opts[k] = v
		}
	}
	return flowcast.Source{Data: data, Name: c.a.file, Options: opts}, nil
}

// printIssues prints errors first and warnings after, keeping the order within
// each group, and returns the number of errors.
func (c *cli) printIssues(issues []model.Issue) int {
	sorted := append([]model.Issue(nil), issues...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Level == model.LevelError && sorted[j].Level != model.LevelError
	})
	ne := 0
	for _, i := range sorted {
		c.println(i.String())
		if i.Level == model.LevelError {
			ne++
		}
	}
	c.println(fmt.Sprintf("check: %d errors, %d warnings", ne, len(issues)-ne))
	return ne
}

func (c *cli) check() int {
	src, err := c.read()
	if err != nil {
		c.println("ERROR   " + err.Error())
		return 1
	}
	r, err := flowcast.Check(src, flowcast.Options{Limits: &flowcast.CLILimits})
	if err != nil {
		c.println("ERROR   " + err.Error())
		return 1
	}
	c.println("check: read table from " + r.Source)
	if c.printIssues(r.Issues) > 0 {
		return 1
	}
	return 0
}

func (c *cli) build() int {
	src, err := c.read()
	if err != nil {
		c.println("ERROR   " + err.Error())
		return 1
	}
	chk, err := flowcast.Check(src, flowcast.Options{Limits: &flowcast.CLILimits})
	if err != nil {
		c.println("ERROR   " + err.Error())
		return 1
	}
	c.println("check: read table from " + chk.Source)
	if c.printIssues(chk.Issues) > 0 {
		c.println("build: stopped because the table has errors")
		return 1
	}

	out := c.a.output
	if out == "" {
		out = strings.TrimSuffix(c.a.file, source.Ext(c.a.file)) + ".drawio"
	}
	mode, ok := c.chooseMode(out)
	if !ok {
		c.println("build: stopped, no write mode chosen")
		return 4
	}
	var prev *merge.Old
	if mode == "merge" {
		data, err := os.ReadFile(out)
		if err == nil {
			prev, err = merge.Read(data, out)
		} else {
			err = fmt.Errorf("cannot read %s: %v", out, err)
		}
		if err != nil {
			c.println("ERROR   " + err.Error() + "; not overwriting. Use --mode force to regenerate everything")
			return 4
		}
	}

	r, err := flowcast.Build(src, flowcast.Options{Config: &c.a.cfg, Title: c.a.title, Previous: prev,
		Direction: c.a.direction, Limits: &flowcast.CLILimits})
	if err != nil {
		c.println("ERROR   " + err.Error())
		return 1
	}
	backup := ""
	if mode != "new" && !c.a.noBackup {
		backup = out + ".bak"
		if err := copyFile(out, backup); err != nil {
			c.println("ERROR   " + err.Error())
			return 4
		}
	}
	if err := os.WriteFile(out, []byte(r.Text), 0o644); err != nil {
		c.println("ERROR   " + err.Error())
		return 1
	}
	c.println("build: mode " + map[string]string{
		"new": "new", "force": "force, regenerate everything", "merge": "merge, keep manual edits"}[mode])
	s := r.Stats
	c.println(fmt.Sprintf("build: wrote %s (%d lanes, %d elements, %d edges, %sx%spx)",
		out, s.Lanes, s.Items, s.Edges, num.Fmt(s.W), num.Fmt(s.H)))
	if backup != "" {
		c.println("build: backup of the old file " + backup)
	}
	if c.a.layoutJSON != "" {
		if err := writeLayoutJSON(c.a.layoutJSON, r); err != nil {
			c.println("ERROR   " + err.Error())
			return 1
		}
	}
	for _, w := range r.Warnings {
		c.println("WARNING layout: " + w.Msg)
	}
	if r.Merge != nil {
		for _, l := range r.Merge.Lines() {
			c.println(l)
		}
		c.println("layout: merge does not self-check wires and labels; see the list above and the PNG image")
		return c.exportAndVerify(out, r, 0)
	}
	nerr := 0
	for _, f := range r.Findings {
		c.println(fmt.Sprintf("%-7s layout: %s", strings.ToUpper(f.Level), f.Msg))
		if f.Level == "error" {
			nerr++
		}
	}
	c.println(fmt.Sprintf("layout: %d errors, %d warnings", nerr, len(r.Findings)-nerr))
	code := 0
	if nerr > 0 {
		code = 2
	}
	return c.exportAndVerify(out, r, code)
}

// exportAndVerify exports the PNG and checks the render when asked. An export
// failure does not spoil the .drawio file already written; it only sets the exit
// code to 3 if no other code is set.
func (c *cli) exportAndVerify(out string, r flowcast.Result, code int) int {
	if c.a.png == "" && !c.a.verify {
		return code
	}
	exe := render.Bin()
	if exe == "" {
		c.println("WARNING drawio CLI not found, skipping image export and render check")
		return code
	}
	verify := c.a.verify
	if r.Merge != nil && verify {
		c.println("render: skipping render check in merge mode (wires are kept from the file or routed by draw.io itself)")
		verify = false
	}
	fail := func(err error) int {
		c.println("ERROR   " + err.Error())
		if code != 0 {
			return code
		}
		return 3
	}
	ctx := context.Background()
	if c.a.png != "" {
		png := c.a.png
		if png == "-" {
			png = strings.TrimSuffix(out, source.Ext(out)) + ".png"
		}
		if err := render.Export(ctx, exe, out, png, "png", 2); err != nil {
			return fail(err)
		}
		c.println("png: " + png)
	}
	if verify {
		dir, err := os.MkdirTemp("", "flowcast-verify")
		if err != nil {
			return fail(err)
		}
		defer os.RemoveAll(dir)
		svg := filepath.Join(dir, "render.svg")
		if err := render.Export(ctx, exe, out, svg, "svg", 0); err != nil {
			return fail(err)
		}
		data, err := os.ReadFile(svg)
		if err != nil {
			return fail(err)
		}
		probs, err := render.Verify(*r.Layout, data)
		if err != nil {
			return fail(fmt.Errorf("cannot read the SVG exported by drawio: %v", err))
		}
		for _, p := range probs {
			c.println("ERROR   render: " + p)
		}
		c.println(fmt.Sprintf("render: %d mismatches", len(probs)))
		if len(probs) > 0 && code == 0 {
			code = 3
		}
	}
	return code
}

// chooseMode returns "new", "force" or "merge". ok is false when it must stop: the
// target file exists, no mode was chosen, and there is nobody at a terminal to ask.
func (c *cli) chooseMode(out string) (string, bool) {
	if _, err := os.Stat(out); err != nil {
		return "new", true
	}
	if c.a.mode != "" {
		return c.a.mode, true
	}
	if !isTerminal(c.stdin) {
		c.println("ERROR   " + out + " already exists; choose --mode merge (keep manually edited positions, lanes, waypoints) " +
			"or --mode force (regenerate everything)")
		return "", false
	}
	fmt.Fprint(c.out, out+" already exists. [m]erge keep manual edits / [f]orce regenerate everything / [q]uit: ")
	line, _ := bufio.NewReader(c.stdin).ReadString('\n')
	switch unistr.Lower(unistr.Strip(line)) {
	case "m", "merge":
		return "merge", true
	case "f", "force":
		return "force", true
	}
	return "", false
}

func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

// strerror rebuilds the system error message in the style of C's strerror, which
// the reference implementation prints: "No such file or directory" rather than
// Go's "open x: no such file or directory".
func strerror(err error) string {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		msg := errno.Error()
		r := []rune(msg)
		r[0] = unicode.ToUpper(r[0])
		return string(r)
	}
	return err.Error()
}

// copyFile copies the old file before it is overwritten, keeping the original's
// permissions and modification time.
func copyFile(src, dst string) error {
	st, err := os.Stat(src)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, st.Mode().Perm()); err != nil {
		return err
	}
	return os.Chtimes(dst, st.ModTime(), st.ModTime())
}

// writeLayoutJSON writes the computed coordinates to JSON, for debugging the layout
// or for other tools to read. The key structure is stable; number formatting is not
// guaranteed.
func writeLayoutJSON(path string, r flowcast.Result) error {
	l := r.Layout
	lanes := []map[string]any{}
	for i, ln := range l.Lanes {
		lanes = append(lanes, map[string]any{"id": ln.ID, "x": l.LaneX[i], "w": l.LaneW[i]})
	}
	items := map[string]any{}
	for _, it := range l.Items {
		items[it.ID] = map[string]any{"kind": it.Kind, "row": it.Row, "col": it.Col,
			"x": it.X, "y": it.Y, "w": it.W, "h": it.H}
	}
	edges := map[string]any{}
	for _, e := range l.Edges {
		edges[e.ID] = map[string]any{"case": string(e.Case), "exit": string(e.ExitSide),
			"points": e.Pts, "label": e.Label}
	}
	b, err := json.MarshalIndent(map[string]any{
		"pool": map[string]any{"w": l.PoolW, "h": l.PoolH}, "lanes": lanes, "items": items, "edges": edges,
	}, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
