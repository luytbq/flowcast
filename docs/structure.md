# Project structure

This document is for someone about to change flowcast code for the first time. After
reading it, you know which package a change belongs in, in what order data passes
through the packages, and what each package is allowed to do.

Terms such as lane, kind, gutter, channel, track are defined in
[CONTEXT.md](../CONTEXT.md). It is best to read that document first.

## Where does a single diagram build go?

Every entry point (command line, web service, library) calls the same Build function
in the root package. Build takes bytes and returns the .drawio text along with
diagnostic data. It does not read or write files, does not call external processes and
prints nothing.

```
input bytes
  --> source     read markdown, csv, xlsx or mermaid, reduce to a Table
  --> validate   check the Table against the metadata schema and the diagram rules
  --> layout     measure text, placement, routing, compute coordinates, place labels
  --> layout     geometry self-check on the result
  --> layout     axis swap if the direction is not top-down
  --> merge      (only when there is an old file) apply manual edits from the old file
  --> writer     generate the .drawio text
```

If the table has errors, Build stops after the validate step and returns only the
list of errors. The Check function in the root package runs exactly the first two
steps.

The result of the layout step is a plain data type named Result: the coordinates of
each element, the waypoints of each wire, the position of each label. Self-check,
merge and writer all read only the Result and never touch the engine's internal state.

## What does each package do?

| Directory | Role | Change it when |
|---|---|---|
| repo root | Build, Check, Options, Result, and the resource limit profiles for the command line and the web | Changing the library's public interface, changing resource limits |
| model | Shared data types: Row, Table, Issue, Location, Error | Adding fields to Issue or to error locations |
| num | Number rounding and printing rules, shared by every layer | Changing how numbers are written to files |
| source | Reads each input format: markdown, csv, xlsx, mermaid; guesses format and encoding | Adding an input format, fixing file reading bugs |
| schema | Declares the valid metadata keys for each row type | Adding a metadata key, adding a row type |
| validate | Checks a parsed Table: valid references, duplicate ids, diagram-specific rules | Adding a table validation rule |
| text | Measures and wraps text using the character width table | Changing how text wraps |
| data | The font's character width table, embedded in the binary | Changing the measuring font |
| layout | The layout and routing engine, geometry self-check, axis swap | Any layout change, see the table below |
| writer/drawio | Generates the .drawio file from a Result | Changing shapes, styles, the output XML structure |
| merge | Reads the old .drawio file, applies manual edits onto the new Result, builds the report | Changing what is kept when regenerating |
| render | Calls the drawio CLI to export PNG and SVG, compares the wires draw.io draws against the computed coordinates | Changing image export or render checking |
| cmd/flowcast | Command line: reads flags, reads and writes files, chooses the write mode, prints results, exit codes | Adding flags, changing printed lines |
| cmd/flowcastd | Web service and upload page | Adding APIs, changing the upload page |
| internal/etree | Reads and writes XML trees, used by writer and merge | Rarely |
| internal/unistr | Shared definitions of whitespace and lowercase | Rarely |
| internal/lint | Static tests that run over the source code itself, for example catching floating point multiplications that could be fused into FMA | Adding a source code check rule |
| conformance | The input case suite, output goldens, and the tests comparing goldens along with the invariants run over every case | Adding cases, updating goldens, see [testing.md](testing.md) |
| tools | Scripts for repo-wide checks, end-to-end acceptance checks with the real drawio, mutation testing, measuring layout quality, extracting measurements from font files | Adding development tools |
| docs | The table format specification, designs, settled decisions | |

### Inside layout

The engine splits its files by phase. Each phase reads the result of the previous
one. How the phases work together is described in [algorithm.md](algorithm.md).

| File | Contents |
|---|---|
| config.go | Layout parameters, defaults and value ranges. Both the command line and the web build their option lists from here |
| model.go | The state of one layout run: elements, edges, grid; engine warnings |
| shape.go | The geometry declaration of each element type: primitive shape, wiring rules, points on the outline |
| size.go | Measures and wraps each element, before any placement step |
| topo.go | Topological order of the nodes |
| branch.go | Chooses the main branch, assigns columns to side branches, finds the join column |
| place.go | Placement: assigns lane, row and column to every element |
| route.go | Chooses the routing style for each edge and records wire segments into gutters and channels |
| tracks.go | Places ports on node faces, assigns wire segments to tracks |
| seg.go | Data types for wire segments and the resources holding wires |
| geometry.go | Converts the grid to pixels, builds the polyline for each wire |
| labels.go | Reserves space and places labels for wires, plus the Run function that runs all phases |
| axis.go | Axis swap for the LR, RL, BT directions |
| result.go | Result, the plain data type returned to the outside |
| check.go | Geometry self-check on a Result |

## What is each package allowed to do?

The following boundaries keep the same code path able to serve the command line, the
web and the library:

- The core consists of the root package plus model, num, source, schema, validate,
  text, data, layout, writer, merge and internal. The core does not read or write files
  and does not call external processes. Input is bytes, output is text.
- render calls the drawio process so it lives outside the core. Only the command line
  and tests use it.
- merge belongs to the core and also takes bytes, but only the command line uses it,
  because only the command line has an old file sitting next to the table. The web
  service does not accept old files.
- model and num are separate from the root package because every input-reading package
  needs them, while the root package needs those packages. The root package re-exports
  the model types through aliases.
- Self-check, merge and writer only read the Result. That way the self-check can check
  geometry that the user edited by hand or that a test deliberately broke.
- Only axis.go knows about direction. The other phases always compute as if the
  diagram goes top-down.

Dependency direction between packages:

```
root package --> source, validate, layout, merge, writer/drawio, text, data, model
source       --> model
validate     --> schema, model
layout       --> text, schema, model, num
writer       --> layout, num
merge        --> layout, num
render       --> layout
```

All of them may use internal. No package in the core may import render or cmd.
