# ADR-0002: Mermaid is input only, never output

- Status: accepted
- Date: 2026-09-21

## Context

Output targets fall into two families. The first consumes coordinates: draw.io,
excalidraw, SVG, PDF. The second does its own layout and only needs the list of nodes
and edges: mermaid, graphviz, plantuml.

Taking the second family into scope would mean the writer seam has to exist in two
different places, one at LayoutResult and one at Table, because the second family
never uses LayoutResult.

More importantly: exporting to mermaid throws away exactly the 840 lines of layout that
are the tool's entire reason to exist.

At the same time, mermaid as input is the product thesis itself. Users already have
mermaid, and both mermaid and draw.io's import feature produce unsatisfying layouts.
What this tool has that they do not is deterministic layout.

## Decision

The writer seam lives only at LayoutResult, meaning it only serves the family that
consumes coordinates.

Mermaid is a source adapter, on par with markdown, csv and xlsx, reducing to the same
Table.

Anyone who wants mermaid output writes a side script that reads the Table, outside the
core.

## Consequences

- The core has exactly one reason to exist, and a single place to plug in writers.
- Conversion is one-way. There is no round-trip commitment from mermaid to mermaid.
- Mermaid syntax that cannot map to the 5-column table is lost. It must become an Issue
  with a code and a line number, never be silently dropped.
- If exporting to a target that does its own layout is ever truly needed, this ADR must
  be reopened, since it places the seam where it cannot serve that family.
