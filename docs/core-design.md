# Core design

This document describes the flowcast core: what it promises its callers, how data
flows through it, and why it is cut the way it is. Vocabulary is defined in
[CONTEXT.md](../CONTEXT.md). Code locations are in [structure.md](structure.md),
the layout algorithm in [algorithm.md](algorithm.md). Decisions and their
rationale are in [adr/](adr/).

Goal: one core serving two very different callers, a CLI on a machine with a
filesystem and an HTTP request that has only bytes, without either side having
to rewrite the other's policy.

## 1. The rule that shapes everything

The core does not touch the filesystem, does not call external processes, does
not read environment variables, does not print to stdout. Bytes go in; the
target document plus diagnostic data come out.

The following work therefore lives outside the core: reading and writing files
(CLI), calling drawio to export images (the render package), and turning results
into printed lines or exit codes (CLI and web).

Testing consequence: every core test runs without temporary files, and every
error is either a returned Issue or a model.Error with a code. There is no third
path. Even when the engine violates one of its own invariants and panics, Build
recovers and returns an error with code layout.internal.

## 2. Technical constraints

Go, module github.com/luytbq/flowcast, with the current stable Go release as the
compatibility baseline.

The core uses only the standard library, with one exception. archive/zip and
encoding/xml are enough to read xlsx; font metrics are embedded so no font
library is needed. The only exception is golang.org/x/text/unicode/norm for NFC
normalization, because the standard library lacks it and reimplementing Unicode
normalization correctly is error-prone; the full rationale is in ADR-0005.

WASM is not in scope yet, but the design does not close the door: the core does
not touch I/O, so building for js/wasm is a matter of adding a target, not a
redesign.

## 3. Public interface

The core exposes the functions below. Everything else is data types.

```go
func Build(src Source, opt Options) (Result, error) // read, check, lay out, generate the file
func Check(src Source, opt Options) (Result, error) // only read and check the table
```

```go
type Source struct {
    Data        []byte
    Name        string            // fallback title, and for guessing the format; never used to open a file
    Format      string            // empty means guess from Name
    Options     map[string]string // format-specific parameters: sheet, delimiter, encoding
    MaxUnzipped int64
}

type Options struct {
    Config    *layout.Config // layout parameters; nil means defaults
    Title     string         // replaces the title taken from the source
    Previous  *merge.Old     // old .drawio file whose manual edits are kept; nil means generate fresh
    Direction string         // TD, BT, LR, RL; empty means follow the source, then TD
    Limits    *Limits        // resource limits; nil means no limits
}

type Result struct {
    Title, Source string
    Issues   []Issue          // findings about the input table
    Text     string           // the .drawio file; empty when the table has errors
    Warnings []Warning        // engine warnings, with codes
    Findings []Finding        // geometry self-check results, with codes
    Layout   *layout.Result   // computed coordinates
    Merge    *merge.Report    // merge report when Previous is set
    Stats    Stats            // counts of lanes, elements, edges, pool size
}
```

Result carries no exit code and no preformatted string to print. The CLI maps to
exit codes itself, the web maps to HTTP status itself. This is the only place
where the two front ends are allowed to differ in policy.

## 4. Error model

Every finding has a stable machine code, made of the domain, a dot, then the
symptom. Codes are part of the public interface: changing a code is a breaking
change.

| Prefix | Domain | Form |
|---|---|---|
| source. | reading and decoding the input | Issue or Error |
| table. | table structure, header, cells | Issue |
| ref. | id references: from, to, attach, parent | Issue |
| schema. | metadata keys missing, unknown, out of domain; layout parameters out of domain | Issue or Error |
| order. | row order per the specification | Issue |
| mermaid. | mermaid syntax with nowhere to go | Issue or Error |
| config. | parameters of the build, such as direction | Error |
| merge. | merge could not be done | Error |
| limit. | resource limit exceeded | Error |
| layout. | engine warnings; layout.internal is an internal error | Warning or Error |
| check. | geometry self-check | Finding |

An Issue is about the content of the table the user wrote, and carries a
structured location (row, cell, sheet, line in mermaid) so the web can point at
the right spot. model.Error is for things that stop the build from continuing and
cannot be traced to a single row: format cannot be guessed, corrupt file, limit
exceeded, bad parameter. A Warning is where the engine can still build but has to
accept a worse option. A Finding is a self-check result.

Message text lives in the core, next to where the problem is detected.
Supporting multiple languages later means looking up by code, without touching
the detection sites.

## 5. Table and metadata schema

The table keeps exactly 5 columns id, type, parent, content, metadata (ADR-0001).
Since the metadata column also carries required references, namely from, to,
attach, explicitness has to live elsewhere: the metadata schema in the schema
package, declaring for each type which keys it accepts, which keys are required,
which keys point to other ids, and the value domains.

validate consists of a generic pass driven by the schema, catching missing keys,
unknown keys, out-of-domain values and dangling references; plus diagram-specific
rules the schema cannot express, such as a condition needing at least two
outgoing edges and outgoing edges having to come right after the node.

parent is a single-parent containment tree, not just "the id of a lane". This
definition covers lanes, mermaid subgraphs, and later subprocesses, at no extra
cost today.

## 6. Configuration

Configuration has two tiers. Resource limits do not depend on the diagram kind,
so they live in the root package, as two data profiles: CLILimits loose,
WebLimits tight. Profiles are data, not if branches in the core.

Layout parameters belong to the engine and are declared as a list of Fields:
name, default, value domain, explanation. The CLI generates flags from this list,
the web generates its form and validates values from the same list, so the two
sides cannot drift apart.

An out-of-domain value is an error that stops the build, code
schema.out_of_range, not an Issue: an Issue is about table content, while this is
a parameter the caller passed in, and there is no partial result worth
returning. Domains are kept wide, because a config that produces an ugly layout
is still the user's right and the self-check will report it; only values that
make the engine misbehave are blocked.

## 7. Direction

The engine lays out in only one direction, top-down. Other directions are built
by axis-swapping the result at the end (ADR-0006). The placement, routing,
geometry and label phases have no code branch for each direction.

Element boxes do not flip with the axes: text still reads left to right, box
width is still capped by the maximum width, height still grows with the number of
lines. Only the grid flips. For horizontal directions, width and height are
swapped before placement so that a row in the virtual space has exactly the
thickness of a real column.

## 8. Layout result

layout.Result is plain data: pool size, lanes, elements with coordinates, edges
with bend points, ports and label positions. The self-check, merge and the
writer only read Result and never call into the engine.

Result speaks in primitive shapes (rectangle, diamond, ellipse, double ellipse,
dashed ellipse, cylinder, note), not in semantic kinds. Each element kind
declares its primitive shape and its wiring rules once in layout.Kinds; the
engine and the writer only read that declaration. If writers had to know semantic
kinds, every writer would have to know every kind, and the amount of work would
be the product of the two numbers instead of their sum. The semantic kind is
still carried in Result for tools and for debugging, but writers do not read it.

The self-check runs on Result. That way it checks every generation path, and it
can also check hand-built broken geometry, which is the only way to test the
self-check itself.

## 9. Merge

Merge regenerates the diagram while keeping what the user edited by hand in the
old .drawio file: node positions, lane widths, wire bend points, and user-drawn
cells. The detailed design is in [merge-design.md](merge-design.md).

Reading the old file is the job of the merge package, which takes bytes, not a
path. Only the CLI uses merge, because only the CLI has an old file sitting next
to the table; the web service does not accept an old file (ADR-0004). Merge
supports only the top-down direction and reports the error merge.direction for
other directions.

An invariant guards merge: merging onto a freshly generated file that nobody has
edited must not change a single byte. The file only stores numbers rounded to two
decimals, so every comparison between a number read back and a computed number
must round exactly as when writing.

## 10. Resource limits

Caps on the number of rows and edges are the main measure, because they also cap
running time: build time grows roughly with the square of the number of elements.
A timeout is a secondary measure, checked at the boundaries between phases,
never cutting into a phase. For xlsx the total unzipped byte count is also
capped, because a zip file of a few KB can unpack to several GB.

Exceeding a limit is an error with code limit.*, not an Issue, because there is
no partial result worth returning.

## 11. Fonts and determinism

Commitment: the same input, the same config, the same flowcast version produce
the same byte sequence, on every machine.

The tool does not read font files at run time. It measures text with a table of
advance widths per codepoint, generated once from the font file by the metrics
extraction tool in tools and embedded in the binary. That way the distributed
binary carries no font and does not depend on whether the target machine has
Verdana installed.

A risk worth stating plainly: Verdana is a commercial Microsoft font.
Redistributing a table of metrics is quite different from redistributing the
font, but if this service goes public someone who knows the law should look at
it. Fallback plan: change the fontFamily in the output to a free font and
generate the metrics table from that font.

### Floating point must be deterministic down to the bit

The geometry phase compares floating-point numbers to make decisions: which
candidate a label goes to depends on which total cost is smaller, which bend
point is dropped depends on whether two coordinates are less than 0.01 apart. A
difference of one unit in the last bit is enough to flip a decision, so results
must be deterministic down to the bit, not just to two decimal places.

The places prone to drift, each already guarded by a check:

- **Rounding.** num.Round rounds on the true binary value, half to even exactly,
  by going through the decimal string. math.Round is wrong for numbers like 2.675
  and 0.125.
- **Multiply then add.** The Go specification allows fusing a*b + c into one FMA
  instruction that rounds only once, and fusing is allowed even across
  statements. On arm64 this happens in roughly a quarter of a*1.42 + c
  operations. Only an explicit conversion prevents fusing, so every
  floating-point multiplication must be wrapped in float64(), unless the result
  goes straight into another multiplication or division. A static test in
  internal/lint checks this rule across the whole module on every go test.
- **Addition order.** Floating-point addition is not associative, so the order in
  which coordinates and costs are accumulated is part of the result.
- **Sign of zero.** max and min in the engine return the first number when the
  two are equal, because math.Max and math.Min change the sign of zero.

## 12. Mermaid input

mermaid maps to the same Table as the three table formats (ADR-0002). Mapping:

| mermaid | Flow Table |
|---|---|
| node with bracket syntax | one row, type inferred from the shape: [] task, {} condition, ([]) start or end, [()] db, (()) external |
| node label | content |
| subgraph | one lane row, and the parent of the nodes inside |
| edge with label | one edge row, content is the label, from and to go into metadata |
| dashed, thick, no arrow | the values dashed, bold, noarrow of the style key |
| classDef and class | resolved at read time into concrete style values on each row |

ids are taken directly from mermaid so they are stable, and merge by id works
with mermaid input with nothing extra.

The boundary of the metadata vocabulary: a key is accepted only if it changes
what is drawn, has a faithful equivalent in draw.io style, and does not reference
another row. By that boundary, arrow kinds are just new values of the existing
style key. Colors set directly on a node are not accepted, because the style key
carries a semantic role, not a color; a color matching the accent palette maps to
highlight, otherwise it produces a warning. icon, click, href do not change the
layout so they are not accepted. Nested subgraphs are flattened to the outermost
subgraph with the warning mermaid.nested_subgraph.

Everything skipped is an Issue with a code and a line number in the source, so
the web can show the list of what was skipped.

## 13. Flowcharts and the main branch

flowchart is not a new diagram kind. It is activity-swimlane with lanes made
optional: a table with no lane rows is built on a hidden lane, the title bar and
lane name bars are zero, and no pool is drawn.

One spot needs special handling. The engine places side branches toward the lane
the branch eventually leads to. In a single-lane diagram no branch leaves the
lane, so that signal is always empty and every branch falls back to the rule of
alternating right then left. Also, the Flow Table specification lets the table
author choose the main branch by row order, while mermaid has no such convention.

So for diagrams without lanes:

- The main branch is the branch with the longest path to an end node, measured on
  the graph with back edges removed, stopping at merge nodes.
- The spine is the chain of main branches starting from each starting point. A
  merge node off the spine keeps the column of its deepest source instead of
  returning to the column of the branching node, so that a step that only
  receives error branches does not pull the main flow around it.

Diagrams with lanes keep the Flow Table convention.

Measurements when these two rules were introduced, using go run ./tools/metrics on
the 28 diagrams in the mermaid and flowchart suites at the time:

| Metric | Before | After |
|---|---|---|
| wires detouring through channels (kind D) | 38 | 33 |
| bend points | 126 | 112 |
| total wire length | 31874 | 29362 |
| edges of the longest path drawn straight down | 117/152 | 135/152 |
| self-check findings | 0 | 0 |

## 14. Things considered and set aside

The items below were considered, with reasons, so next time nobody has to think
them through from scratch.

**Reducing wire crossings.** At measurement time there were 33 crossings across
the 28 diagrams of the mermaid and flowchart suites, the worst being the CI
diagram with 7. Cause: detouring wires run vertically in the gutter right next
to the source column, and that gutter is where every horizontal arrow leaving
that column passes through. Quick experiments tried and dropped:

| Experiment | Crossings | Length | Self-check |
|---|---|---|---|
| at measurement time | 33 | 29362 | 0 |
| branches with no clear direction always shift right | 33 | 29379 | 0 |
| detouring wires always use the outermost gutter | 27 | 32530 | 5 |
| as above, but fall back when two horizontal segments hit a node | 31 | 32066 | 2 |

The last two made wires nearly 10% longer and produced real geometry errors.
Doing it properly means choosing the gutter and the track together, with a cost
function that includes the number of crossings, rather than choosing the gutter
first and assigning tracks afterwards as it is done now.

**Self-check after merge.** Worth less than it seems: wires kept by the user are
correct by definition, wires left for draw.io to route have no coordinates to
check, and labels are not recomputed after nodes move. The only meaningful part
left is overlapping elements, which the merge report already lists. Doing this
properly means a second routing phase running on geometry the user placed.

**Merge for directions other than top-down.** The cheapest way is to axis-swap
the old file when reading and swap back when writing, but user-drawn cells are
copied verbatim so they would also have to be axis-swapped, and that is where it
is easy to get wrong. For now merge reports clearly that it is not supported.

**Lanes nested in lanes.** The model can handle it because parent is already a
tree, but the engine cannot lay it out yet.

**Lane thickness in horizontal diagrams.** Lane names are rotated vertically, but
the engine still takes the text width as the minimum thickness of the band, so
long names make the band too thick. The proper fix is to separate the two
numbers: thickness from text height, length from text width.

**WASM.** The core does not touch I/O, so running directly in the browser is a
packaging matter, not an architectural one. Not done because the web service is
good enough.
