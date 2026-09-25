# flowcast glossary

This document defines the terms used throughout the code, tests and design docs.
A concept has exactly one name. Seeing a different name in the code means the code
needs fixing, not that the docs need another synonym.

## Domain: tables and diagrams

**Flow Table** - a flow diagram written as a table with 5 columns id, type, parent,
content, metadata. The full specification is in flow-table-format.md. The number of
columns is a compatibility commitment, not an implementation detail.

**Source** - unparsed input: bytes, a display name, a format hint. The CLI builds a
Source from a path, the web service builds one from an upload. The core only accepts
a Source, never a path.

**Table** - the result of parsing a Source: title, a list of Rows, a list of Issues.
Every input format reduces to this, mermaid included.

**Row** - a table row already split into cells, not yet interpreted in terms of
diagram semantics.

**kind** - the diagram type. A kind owns three things: the metadata schema, the
validation rules, and the layout algorithm. There is currently a single kind,
activity-swimlane, and a flowchart is that same kind with lanes made optional.

**metadata schema** - a kind's declaration of the valid keys in the metadata column:
per type, which keys are required, which keys are references to other ids, and the
value range of each key. Because the metadata column carries required references too
(from, to, attach), this schema is the only place that keeps them explicit.

**lane** - a band holding elements. A diagram may have no lanes at all, in which case
it is a flowchart. A lane is optional, not a privileged concept.

**main branch** - among a node's outgoing edges, the branch that stays in that node's
column. The other branches are side branches and drift to either side. The Flow Table
lets the author designate the main branch by row order; for input without that
convention, the main branch is inferred from the path depth to an end node.

## Axes

**flow** - the axis along which the flow advances. In a TD diagram this is the
vertical axis, in LR it is the horizontal axis.

**cross** - the branching axis, perpendicular to flow. Columns within a lane and the
order of lanes are both measured on this axis.

The placement and routing algorithms speak only in terms of flow and cross. Only a
single module is allowed to map that pair to screen x and y. That is why LR, BT and RL
are variants of the same mapping rather than separate code paths.

**gutter** - the empty space between two adjacent cross positions, where wire segments
run along flow.

**channel** - the empty space between two adjacent flow steps, where wire segments run
across along cross.

**track** - a wire lane inside a gutter or a channel. Several edges share a gutter by
being assigned different tracks; edges with the same target share a track so they join
into a single line.

## Results and outer layers

**LayoutResult** - plain data describing a laid-out diagram: pool size, lanes, elements
with coordinates and primitive shapes, edges with waypoints and label positions. It
speaks in primitive shapes (rectangle, diamond, ellipse, double ellipse, dashed
ellipse, cylinder, note) rather than semantic types, so that a writer does not need to
know about kinds.

**geometry declaration** - for each element type, its primitive shape and its wiring
rules, such as only accepting incoming wires at the top. The engine and writers only
read this declaration, they do not compare type names.

**writer** - a function that turns a LayoutResult into the text of a target format.
A writer only reads the LayoutResult, it does not call the algorithm.

**renderer** - the thing that turns target text into an image. A renderer calls an
external process, so it lives outside the core.

**Overrides** - the manual edits extracted from an existing target file, keyed by id:
positions, sizes, waypoints, plus the cells the user drew on their own, kept in sealed
form. Reading the target file is the job of that format's adapter; applying Overrides
to a LayoutResult is the core's shared job.

**merge** - regenerating a diagram while keeping Overrides. After Overrides are
applied, the LayoutResult must still go through check().

**check** - the geometry self-check on a LayoutResult: wires crossing nodes, two
overlapping wires, overlapping nodes. Because it runs on the LayoutResult, every kind
and every generation path, merge included, gets checked.

**verify** - comparing the image drawn by a renderer against the computed coordinates.
It differs from check in that it needs an external process, so it is a testing tool
rather than a step of build.

## Configuration and errors

**CoreConfig** - kind-independent configuration: font, resource limits, strictness
level.

**kind config** - layout parameters declared by a kind, with defaults and value
ranges. The CLI generates flags and the web service generates a form from that same
declaration, so the two cannot drift apart.

**Issue** - a finding returned to the caller, carrying a stable machine code, a
severity, a structured location and a human-readable message. The core does not print
Issues; the caller presents them itself.

**BuildResult** - what build() returns: the target text, the list of Issues, the merge
report if any, and statistics. It contains no exit code and no preformatted string
for printing.
