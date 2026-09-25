package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/luytbq/flowcast/layout"
)

// args holds the arguments of one command: flags may come before or after the
// file name, and both "--flag value" and "--flag=value" are accepted. Go's
// standard flag package stops at the first positional argument, so it cannot be
// used.
type args struct {
	cmd        string
	file       string
	output     string
	title      string
	direction  string
	layoutJSON string
	mode       string
	png        string // empty: no export; "-" means the default path
	verify     bool
	noBackup   bool
	sheet      string
	delimiter  string
	encoding   string
	cfg        layout.Config
}

// configFlags maps command-line flags to layout.Config fields. Flag names are an
// interface that scripts calling flowcast rely on, so change them only on purpose.
func configFlags(c *layout.Config) map[string]*int {
	out := map[string]*int{}
	for _, f := range layout.Fields() {
		out[f.Name] = f.Get(c)
	}
	return out
}

// errHelp signals that the user asked for --help; the caller prints usage and exits 0.
var errHelp = errors.New("help")

// helpText builds the usage text. The layout parameter section is generated from
// the declarations in layout.Fields, so adding a parameter there makes it appear here.
func helpText() string {
	var b strings.Builder
	b.WriteString(`flowcast: turn a flow table or mermaid flowchart into a draw.io file

usage:
  flowcast check <file> [--sheet S] [--delimiter D] [--encoding E]
  flowcast build <file> [flags]

build flags:
  -o, --output FILE    .drawio file to write; defaults to next to the input file
  --title T            title; defaults to the one in the source
  --direction D        TD, BT, LR or RL; defaults to the source, or TD if the source has none
  --mode merge|force   when the target file exists: merge keeps manual edits, force regenerates
  --no-backup          do not write a .bak file on merge or force
  --png [FILE]         export a PNG image with the drawio CLI
  --verify             export SVG and compare the wires draw.io draws with the computed coordinates
  --layout-json FILE   write the computed coordinates to JSON

layout parameters, in pixels:
`)
	for _, f := range layout.Fields() {
		fmt.Fprintf(&b, "  --%-15s %s; default %d, range %d..%d\n", f.Name, f.Help, f.Default, f.Lo, f.Hi)
	}
	return b.String()
}

func parseArgs(argv []string) (*args, error) {
	if len(argv) == 0 {
		return nil, fmt.Errorf("missing command; use check or build")
	}
	if argv[0] == "--help" || argv[0] == "-h" {
		return nil, errHelp
	}
	a := &args{cmd: argv[0], cfg: layout.DefaultConfig()}
	if a.cmd != "check" && a.cmd != "build" {
		return nil, fmt.Errorf("unknown command %q; use check or build", a.cmd)
	}
	ints := configFlags(&a.cfg)
	strs := map[string]*string{"sheet": &a.sheet, "delimiter": &a.delimiter, "encoding": &a.encoding}
	if a.cmd == "build" {
		strs["output"], strs["title"], strs["layout-json"] = &a.output, &a.title, &a.layoutJSON
		strs["mode"] = &a.mode
		strs["direction"] = &a.direction
	}
	rest := argv[1:]
	for i := 0; i < len(rest); i++ {
		tok := rest[i]
		if tok == "-o" {
			tok = "--output"
		}
		if !strings.HasPrefix(tok, "--") {
			if a.file != "" {
				return nil, fmt.Errorf("unexpected extra argument %q", tok)
			}
			a.file = tok
			continue
		}
		name, val, hasVal := strings.Cut(tok[2:], "=")
		next := func() (string, error) {
			if hasVal {
				return val, nil
			}
			if i+1 >= len(rest) {
				return "", fmt.Errorf("--%s needs a value", name)
			}
			i++
			return rest[i], nil
		}
		switch {
		case name == "help":
			return nil, errHelp
		case a.cmd == "build" && name == "png":
			// The value is optional: take the next argument as the output path if
			// it is not a flag, otherwise use the default path.
			a.png = "-"
			if hasVal {
				a.png = val
			} else if i+1 < len(rest) && !strings.HasPrefix(rest[i+1], "-") {
				i++
				a.png = rest[i]
			}
		case a.cmd == "build" && name == "verify":
			a.verify = true
		case a.cmd == "build" && name == "no-backup":
			a.noBackup = true
		case strs[name] != nil:
			v, err := next()
			if err != nil {
				return nil, err
			}
			*strs[name] = v
		case a.cmd == "build" && ints[name] != nil:
			v, err := next()
			if err != nil {
				return nil, err
			}
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("--%s needs an integer, got %q", name, v)
			}
			*ints[name] = n
		default:
			return nil, fmt.Errorf("unknown flag --%s for command %s", name, a.cmd)
		}
	}
	if a.file == "" {
		return nil, fmt.Errorf("missing input file")
	}
	if a.mode != "" && a.mode != "merge" && a.mode != "force" {
		return nil, fmt.Errorf("--mode accepts only merge or force, got %q", a.mode)
	}
	if a.direction != "" && !layout.ValidDirection(a.direction) {
		return nil, fmt.Errorf("--direction accepts only TD, BT, LR or RL, got %q", a.direction)
	}
	return a, nil
}
