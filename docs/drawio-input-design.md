# Design: re-laying out existing draw.io files

Status: proposal, not implemented. Three decisions are still open, listed at the
end of this document.

This document is for the person who will implement draw.io and PlantUML input. It
assumes the vocabulary of [CONTEXT.md](../CONTEXT.md) and the package map of
[structure.md](structure.md).

## What problem does this solve?

A user already has a .drawio file, drawn by hand or by another tool. It may hold one
or several pages, and one or several diagrams on the same page. The layout is messy:
wires cut through nodes, labels overlap, branches wander. The user wants flowcast to
read the file, understand the diagrams in it, and lay them out again with the
existing engine, while keeping everything that is not geometry: colors, fonts, icons,
custom properties, freehand notes.

A second, smaller goal: accept PlantUML activity diagrams as input, the same way
mermaid is accepted today.

## Why can draw.io input not go through Build like mermaid?

Mermaid carries only content, so reading it into a Table and generating a fresh
.drawio file loses nothing. A user's .drawio file carries far more than a Table can
hold: styles, rich text, images, properties on UserObject cells, decorations, extra
pages. The current writer derives every style from the primitive shape, so a round
trip through Table and the writer would throw all of that away.

So draw.io input uses geometry patching instead of regeneration: the original file is
the base, and the tool only replaces positions, sizes, waypoints, wire anchors and
label positions. Everything else stays as it was.

This is the mirror image of merge. Merge keeps the old geometry and regenerates the
styles; relayout keeps the old styles and regenerates the geometry.

Table remains the intermediate form. validate, layout.Lay and layout.Check are used
unchanged. ADR-0001 and ADR-0002 still hold: the Table is the internal
representation, and the writer seam stays at the layout Result.

## How does a relayout run?

```
file.drawio
  --> doc        read every page (compressed ones too): cell tree, ordered styles,
                 absolute coordinates
  --> segment    split each page into independent diagrams plus decoration
  --> ingest     each diagram --> a Table plus a map from Table id to original cell
  --> validate, layout.Lay    unchanged
  --> pack       place the Results of one page back without overlapping
  --> patch      write the new geometry into the original cells
  --> new .drawio text
```

Build stays as it is for tables, mermaid and PlantUML. Relayout is a new entry point
in the root package, sharing validate, layout, check and render with Build. Like
Build, it takes bytes and returns text, so it belongs to the core.

## What are the pieces?

### doc: a draw.io document model

The draw.io reading code currently lives inside merge (reading pages, decompressing,
parsing styles, computing absolute boxes) and serves only merge. It moves into its own
package and grows to cover:

- every page, not just the first;
- layers, groups, nested containers, object and UserObject wrappers;
- edges without a source or target, edges connected to edges, edge labels stored as
  child cells;
- editing a single style key without reordering the others.

Invariant: reading a file and writing it back without edits yields an equivalent
file. Merge then switches to this package.

### segment: finding the diagrams on a page

- A diagram is a connected component of the vertex and edge graph, joined with
  container membership: everything inside one pool belongs to one diagram.
- A note or free text standing close to a node, with no wire, is attached to the
  nearest node (it becomes an attach), within a distance threshold.
- Everything else (page titles, legends, images, lone vertices, dangling edges) is
  decoration. Decoration is not laid out; it moves along with its nearest diagram so
  it does not get covered.
- A page with no flow-shaped diagram is copied unchanged.

### ingest: one diagram --> a Table

Element types are inferred from the style and from the graph:

| draw.io | Table |
|---|---|
| rhombus with 2 or more outgoing wires | condition; with fewer it becomes a task, with a warning |
| ellipse with no incoming wire, or no outgoing wire | start, or end |
| any other ellipse | external |
| cylinder, datastore | db |
| text or note next to a node | text with attach |
| any other shape (actor, cloud, image, unknown shapes) | task; its original style is kept by patch |
| swimlane container | lane; a pool holding lanes is the pool |

The Flow Table needs information that draw.io does not declare. The user's current
layout supplies it as hints:

- Direction: if most wires go down, TD; if most go right, LR.
- Lane order follows the current positions.
- Main branch: among a node's outgoing wires, the one whose target is best aligned
  with the node along the flow axis is the main branch. It is written first in the
  Table, which is the existing Flow Table convention.
- Back edges: a depth-first traversal from the start nodes, visiting in position
  order.

Anything that cannot be mapped (containers nested deeper than pool and lane, groups,
edges connected to edges, rotated shapes) becomes an Issue with a code and the cell
id, as mermaid input does.

### Text and sizes: the main risk

The engine measures text with the Verdana 12 width table. Real files carry HTML rich
text, arbitrary font sizes and families. Proposal:

- Strip the HTML value into plain lines for measuring, but keep the original value
  when patching, instead of overwriting it with pre-wrapped lines as the writer does.
- Measure as Verdana and scale by fontSize / 12. Verdana is wider than most fonts, so
  boxes come out slightly too large rather than too small.
- Turn on whiteSpace=wrap on patched cells so draw.io wraps inside the computed box.
- layout needs a way to receive a per-element font scale or minimum size. This is the
  only change inside the engine.

### pack and patch

- pack: each diagram keeps the top-left corner of its original bounding box. When the
  new sizes make two diagrams overlap, diagrams are shifted in reading order (top to
  bottom, left to right), deterministically.
- patch: move cells into their new lane when needed, write coordinates relative to
  the parent, set exit and entry anchors in the style, use orthogonalEdgeStyle with
  waypoints, and reposition edge labels, including labels stored as child cells. Most
  of the geometry writing already exists in writer/drawio.
- layout.Check runs on every diagram, and --verify works through render as it does
  today.

### CLI, web service and docs

- A new command, for example flowcast relayout file.drawio. Input and output share a
  format, so the default output path needs its own rule, see the open decisions.
- The web service can accept a .drawio upload: it still reads and writes no files on
  the server.
- A new ADR records "draw.io input patches geometry, it does not regenerate".
  CONTEXT.md gains the terms Document, Page, Diagram, Decoration and Patch.

## How is it tested?

Two strong invariants come almost for free:

- A file generated by flowcast, relaid out, must have the same geometry as building
  from its original table, since the ids already match. Every conformance case
  becomes a relayout case.
- Relaying out twice gives the same result as relaying out once (a fixed point), on
  top of the usual determinism.

A corpus of real hand-drawn .drawio files is also needed: several pages, several
diagrams per page, varied shapes and fonts.

## What about PlantUML?

PlantUML is much simpler than draw.io: it is a text source adapter like mermaid,
goes through Build, and needs no patching.

- Only activity diagrams (the current syntax) are accepted:

| PlantUML | Table |
|---|---|
| :action; | task |
| if / elseif / else / endif | condition, with branch labels |
| start, stop, end | start, end |
| \|Lane\| | lane |
| note | text with attach |
| repeat, while | back edge |
| database | db |

- endif produces no node: the branches connect straight to the next step.
- fork and join need a new element type (a horizontal bar). That is one entry in
  schema.Elements plus one in layout.Geometry, as the architecture intends.
- PlantUML actions have no ids, so ids are generated in order (A1, A2, ...). Merge by
  id is therefore unstable when the user inserts a step in the middle; the user
  documentation must say so.
- Sequence, class and component diagrams are not layered flows and are outside the
  engine's scope. They produce an Issue.

## In what order should it be built?

1. The doc package with its round-trip test; merge switches to it.
2. Relayout of files generated by flowcast: one page, one diagram, the patch writer.
   The "same as build" invariant runs on the whole conformance suite right away.
3. Hand-drawn files: type inference, lanes from containers, HTML text, font scaling,
   Issues for what cannot be mapped.
4. Several diagrams on one page: segment, pack, decoration.
5. Several pages.
6. The PlantUML activity adapter. It is independent of steps 1 to 5 and can be done
   at any time.

## Open decisions

- Default output: a new file next to the input (proposed: file.relayout.drawio), or
  overwrite in place with a .bak copy?
- Box sizes: measure again from the text (proposed, optionally never smaller than the
  original size), or keep the sizes the user drew?
- Test corpus: which real hand-drawn files, with several pages, form the first set of
  cases?
