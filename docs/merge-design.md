# Design: regenerating while keeping manual edits

Status: implemented in the merge package (merge.Read reads the old file, merge.Apply adjusts the fresh layout to it) and wired into flowcast.Build through Options.Previous; drawio.WriteMerged writes the result. Tested by the CLI transcripts that run the scenarios in conformance/merge, plus unit tests in merge/. Usage: README.md, section on regenerating when the .drawio file already exists.

## Problem

Workflow: from a Flow Table file xxx.md, generate xxx.drawio; the user edits the drawio file by hand, then edits the table (adds, removes, changes nodes and edges) and regenerates. The regeneration must keep as many of the manual edits worth keeping as possible.

Manual edits worth keeping are usually:
- moving an element;
- adjusting an edge: adding, removing, moving bend points.

## Settled decisions

### 1. Two generation modes: force and merge

- force: regenerate everything, discard the old file.
- merge: regenerate, but keep some geometric properties taken from the existing file, matched by element id (the id in the table is also the id of the cell in drawio).
- Chosen by the --mode flag. If the target file already exists and no flag is given, ask when running in a terminal. If not running in a terminal (for example when called by an agent), stop with an error asking for an explicit choice.
- Merge exists only in the CLI, where the old file sits next to the table. The web service does not accept an old file (ADR-0004).
- Merge supports only the TD direction. Asking to merge a diagram in another direction is an error (merge.direction) that points to --mode force.

### 2. Scope of merge

Kept from the existing file, by id:
- element position;
- lane position and width;
- edge waypoints and anchor points (exit/entry);
- cells the user drew (section 7).

Everything else is regenerated from the table: text, style, shape kind, the lane containing the element, adding or deleting elements and edges.

The original of the previous generation is not stored. Merge is two-way: an element present in the existing file takes its geometry from there, regardless of whether the user moved it or not.

An element whose lane differs between the table and the file (the table moved it, or the user dragged it into another lane) does not keep its position; it is placed again like a new element and listed in the report.

### 3. Placing new elements by pinned neighbors

Old elements stay where they are, so the coordinates from the freshly computed layout no longer match their surroundings.

- New flow elements are placed in topological order, so a new element placed earlier can serve as the anchor of a later one.
- The anchor of a new node is the nearest node whose position is already fixed: first the sources of its incoming edges, then the targets of its outgoing edges (forward edges before back edges, each in table order), then ring by ring outward through the graph.
- The vertical offset from the anchor is kept as in the fresh layout. When the anchor is in the same lane, the horizontal offset is kept too; otherwise the node keeps its fresh offset from the left edge of its lane, pulled inside the lane when the lane is wide enough.
- A node with no fixed node anywhere in its graph is placed below everything already placed, one minimum channel lower.
- If the node, with a 10px margin, overlaps an already placed element or a user-drawn shape, it moves down in 10px steps until it is clear. Old elements are never moved. Nodes that had to move are listed in the report.
- A new db/text is placed after the flow elements, anchored to the node it attaches to by the same rule.
- Edges:
  - the edge exists in the file with the same source and target, and both ends kept their old position: keep waypoints and anchor points;
  - otherwise (it connects to a new node, a node that was re-placed, or from/to changed): write no waypoints, let draw.io route an orthogonal wire itself, and list it in the report so the user can adjust by hand.

### 4. Lanes also keep their geometry

Node coordinates in drawio are relative to the lane. If a lane's width is regenerated, old nodes drift, waypoints no longer line up, and nodes may spill out of the lane. So a lane is treated like an element:

- Old lane: keep its width from the file. The file stores widths rounded to two decimals; a width equal to the fresh width after rounding counts as unedited and the unrounded fresh width is used, so an unchanged table yields an unchanged file.
- A lane widens only when some element (old or new) does not fit inside it with a 10px margin. Widening on the left shifts the lane's own content right; any widening shifts the lanes to the right, their elements, the waypoints lying in them and the user-drawn cells in them by the same amount.
- New lane: inserted according to table order, width from the freshly computed layout, lanes to the right shift over.
- Lane removed from the table: dropped, lanes to the right shift left (waypoints follow). Waypoints and user-drawn cells that lay inside it follow the next remaining lane, or the last lane when none remains after it.
- Lane order always follows the table.
- A diagram without lanes keeps its nodes directly on the draw.io default layer, which plays the role of the single hidden lane.
- If a node or waypoint was dragged above the top of the lanes, the whole diagram is pushed down so it sits below the headers again.
- The pool resizes itself to enclose all content; the title comes from the source file or --title.

### 5. Sizes regenerated, anchored at the center

- Node size is always recomputed from the text; sizes the user dragged by hand are not kept.
- The kept position is the node's center. This way wires into the top/bottom still pass through the midpoint of the side, wires into a side keep their height, and old waypoints stay orthogonal to the node.
- If a node grows and overlaps a neighboring element, nothing is pushed; the overlapping pair is only listed in the report.

### 6. Edge labels are outside the scope of merge, except for unedited edges

- Label text is always regenerated from the table.
- Merge does not keep label positions from the file. Label positions are computed by the tool based on the tool's own wire path, so they cannot be applied to a wire path kept from the file.
- An edge whose kept wire path matches the path the tool just computed is considered unedited and uses the label position the tool computed. Matching means the same number of waypoints, each within 2px on both axes, and the same exit/entry ratios after rounding to two decimals as they are written.
- The remaining kept edges get no label position; draw.io places the label at the middle of the line. Edges with a label in this situation are listed in the report. Edges routed by draw.io get no label position either.
- Force mode is unchanged: labels are placed in free space as they are now.

### 7. Keeping cells the user drew

- Every cell generated by the tool carries the marker flowtable=1 in its style. A marked cell whose id is no longer in the table is deleted and listed in the report.
- A cell without the marker that is not the pool or a lane is a user-drawn cell: copied as is into the new file (style, text, geometry). If its parent lane still exists it stays in that lane and moves with it; if the lane was deleted it moves into the pool, keeping its absolute position. A user-drawn cell whose parent is some other removed cell is dropped and listed in the report.
- A user-drawn edge connected to a node that no longer exists: that end is replaced by a free point at the old center of the node, and the edge becomes a dangling wire. Each such end is listed in the report.
- The report lists the user-drawn cells that were kept.
- Force mode discards all user-drawn cells.
- A file generated by a tool version that did not add markers (no cell has a marker): the pool, the swimlane cells directly inside the pool, ids still present in the table, and ids that have the form of a table id (LANE-number, E+number, optionally with a decimal suffix) are treated as generated by the tool.

### 8. The target file name only changes the extension

- The default target file is the input file (.md, .csv, .xlsx or .mmd) with its extension changed to .drawio, in the same directory (xxx.md --> xxx.drawio). There is no other naming rule.
- The -o flag overrides the target path.
- The CLI always prints the target file and the mode being run (new, force, merge).

### 9. Back up the old file before overwriting

- Before overwriting (merge or force), copy the existing target file to <file>.drawio.bak in the same directory. Only one copy is kept; the next run overwrites it.
- If the target file does not exist yet, no .bak is created.
- The --no-backup flag turns off the backup.
- The CLI prints the path of the .bak file.

## Checks after merge

- Merge does not rerun the geometry self-check: wires are either kept from the old file or routed by draw.io, so the self-check has nothing to say about them. The findings in the result are those of the fresh layout before merge adjusted it, and the CLI says so.
- --verify skips the render comparison in merge mode for the same reason.

## Implementation notes

- The tool reads a plain mxGraphModel file or an mxfile with pages. The first page may be compressed by draw.io (base64 of raw deflate data holding URL-encoded XML). Cells wrapped in object/UserObject are supported. The other pages of the file are copied as is into the new file.
- A duplicate id in the old file: the later cell replaces the earlier one but keeps the earlier one's position in the order.
- A file that cannot be read (no pages, no root, first page cannot be decompressed) stops the build without overwriting, and the CLI suggests --mode force.
- The pool height after merge is the lowest bottom of the nodes, waypoints and user-drawn shapes plus the minimum channel (--min-channel), and never less than the headers plus one minimum channel, so an unchanged table yields an unchanged file.
