# Layout and routing algorithm

This document is for people who modify the layout engine. After reading it, you
can trace which phase causes a layout bug, and fix it without breaking
determinism: the same input always produces the same file.

Read [CONTEXT.md](../CONTEXT.md) first, especially the terms lane, main branch,
flow, cross, gutter, channel and track. The location of each engine file is in
[structure.md](structure.md).

## Which phases does the engine run through?

The engine does not compute pixels right away. It first places everything on an
abstract grid: each element has a lane, a row and a column within the lane. Each
wire segment lies in a gutter or a channel, on some track. Only once the grid is
done does the engine convert the grid to pixels.

The engine knows only one direction, top-down. Other directions are built in a
virtual top-down space and then axis-swapped at the end.

The phases in order:

1. Measure the size of each element.
2. Placement: assign lane, row, column.
3. Routing: pick a wire kind for each edge, set ports, assign wire segments to tracks.
4. Geometry: convert the grid to pixels, build polylines.
5. Repeat phase 4 once, after learning how much extra room labels need.
6. Place labels.
7. Geometry self-check.
8. Axis swap according to the direction.

Phase 1 runs while building the initial state, phases 7 and 8 run in Build. The
remaining phases are in the Run function.

## 1. Measuring

Each element is wrapped to the maximum width of its kind (task, condition,
start, end, external, db, note), then measured with the character width table
embedded in the binary. Size depends only on kind, content and config, not on
position. That is why this phase runs before every other phase, and every later
phase treats sizes as constants.

Text is wrapped up front and written to the file with line break tags. The
output file does not enable draw.io's automatic wrapping, because draw.io
re-wrapping could produce a different number of lines than the computed size.

## 2. Placement

### Traversal order

The engine visits nodes in topological order, ignoring edges marked back (loop
edges). Two nodes at the same level come out in the order they are written in
the table. Nodes inside a loop that is not marked back have no topological
order. Those nodes are appended at the end in table order, with a warning.

db and notes with attach do not take part in this order. They are placed right
after the node they attach to.

### Row

- Node whose sources are already placed: its row is the deepest row among the
  sources, plus one.
- Start node with no source: row 0.
- Other node with no source: the row right below the deepest row used so far.

A node with exactly one source is tried on the same row as the source, so the
arrow runs straight across. Conditions:

- the source is in a different lane, or the node is a side branch of the source;
- the cells between source and target on that row are empty, and no already
  placed horizontal arrow runs through them;
- the node is not a condition. A condition only accepts incoming wires at its
  top, so it always sits below its source;
- the node is not a non-rectangular shape with two or more side branches,
  because it needs both sides for the branches.

### Column and branches

A node's outgoing edges within the same lane are split into a main branch and
side branches.

- Diagram with lanes: the main branch is the edge written last in the table.
  Table authors are told to put the longest continuing branch last.
- Diagram without lanes (usually from mermaid): the main branch is the deepest
  branch, measured by the number of nodes on the longest path starting from the
  head of the branch. On a tie, the edge written later wins.

The main branch keeps the column of the branching node. Each side branch shifts
to one side:

- A branch that eventually leaves the lane shifts toward the lane it leads to.
  The engine walks along the branch to the first edge that leaves the lane, and
  stops at a merge node because from there on it is the shared flow. This way a
  branch that returns an error to the lane on the left sits on the left, and its
  wire does not have to go around other nodes.
- A branch that does not leave the lane has no clear direction. Such branches
  alternate right, then left.
- If one side of the node already has a horizontal arrow, all side branches go
  to the other side.

A node with several same-lane branches coming in (a merge node) returns to the
column of the nearest branching node that every branch passes through. The
shared flow after the merge is therefore in line with where it branched. Diagrams
without lanes apply this only to nodes on the spine, that is, the path that
follows the main branch from each starting point.

If the intended cell is already taken, a side branch tries shifting further
sideways up to three columns before settling for the row below.

### db and notes

db and notes sit on the same row as the node they attach to, on the side of the
node that has no connected edge. If both sides have edges, the right side is
chosen. The engine tries the adjacent cell on that side, then the adjacent cell
on the other side, then the cell two columns away, and finally backs off further
with a warning.

At the end of the phase, the columns and gutters of all lanes are numbered
continuously along the horizontal axis, so later phases can compare horizontal
positions across lanes.

## 3. Routing

### Wire kinds

Rules specific to each element kind, such as a condition only accepting wires at
its top or which shape has only one connection point per side, are read from
that kind's geometry declaration table, not written as conditions scattered
through the code.

The engine scans all edges in four passes, one wire kind per pass, from simple
to general. An edge takes the first kind whose path is still free.

| Kind | Shape | Condition |
|---|---|---|
| A | Straight down from the bottom of the source to the top of the target | Same column, target on a lower row, the cells in between empty or occupied only by other wires with the same target, and the bottom of the source has no outgoing wire yet |
| B | Straight across from a side of the source to a side of the target | Same row, different column, both of those sides have no wire and no db or note next to them, the cells in between empty, target is not a condition |
| C | L-shape: out of a side of the source, turn down into the top of the target | Target on a lower row and a different column, the exit side free, both the horizontal and vertical segments free |
| D | General: goes through gutters and channels | All remaining edges, including loop edges |

The pass order is deliberate: straight wires reserve their space first, so
complex wires must avoid them and not the other way around.

Wires of kind A, C, D always enter the top of the target. Only kind B wires enter
a side.

### Path of a kind D wire

First the engine picks the exit side. The order tried is the side facing the
target, then the bottom, then the opposite side. If the target is on the same
row or above, the bottom is not tried. The engine takes the first free side. A
side is not free when:

- a db or note sits next to that side;
- that side already has an incoming wire;
- the node is not a rectangle and that side already has an outgoing wire,
  because diamonds and ellipses have only one connection point in the middle of
  each side;
- the node is a rectangle and that side already has a kind B or C wire, because
  those two kinds exit exactly at the middle of the side.

If no side is free, the engine exits on the side facing the target, and the port
placement step separates the port from the incoming wires on the same side.

After that, the path depends on the exit side:

- Exit on a side: run vertically in the gutter next to that side, to the channel
  right above the target row, then run horizontally in the channel to the target
  column and go down into the top of the target.
- Exit at the bottom, target on the row right below: run horizontally in the
  channel right below the source to the target column.
- Exit at the bottom, target column free all the way down from the source row:
  run horizontally in the channel right below the source, then go straight down
  along the target column.
- Exit at the bottom, otherwise: run horizontally in the channel right below the
  source to the gutter next to the target column, down the gutter to the channel
  above the target row, then across into the target column.

Each horizontal or vertical segment is recorded as an interval on a gutter or a
channel. Its coordinates are not known yet. The engine only records which
segment connects to which segment and to which node, so the geometry phase can
fill in the pixels later.

### Port placement

A port is the point where a wire touches a node, recorded as a ratio of the
node's width and height.

- Incoming wires always connect at the middle of a side.
- Outgoing wires on the same side share a source, so they are allowed to join.
  Diamonds and ellipses let them exit together exactly at the middle of the
  side. Rectangles keep A, B, C wires at the middle of the side and divide the
  rest of the side among the D wires. A wire takes a port on the side it turns
  toward, so the first segments do not cross.
- If a side has both outgoing and incoming wires, the outgoing wire must move
  away from the middle of the side, because the incoming wire differs in both
  source and target, so overlapping it is wrong. With one outgoing wire, the
  port shifts a quarter of the side toward the direction the wire will turn. For
  diamonds and ellipses, the shifted port lies on the shape's outline rather than
  on its bounding box, and is rounded to two decimals as when writing the file.

### Track assignment

Each gutter and each channel is a resource that holds many wire segments. The
engine assigns segments to tracks (wire lanes) by interval coloring, greedily in
the order segments were created:

- Two overlapping segments on the same resource must be on different tracks.
- Two segments with the same target may share a track, and are drawn as one
  shared line.
- If two segments have legs connecting to the same position from two sides, the
  segment coming from the lower side must be on the smaller track. Otherwise the
  two legs overlap.

If no track can take a segment, the engine inserts a new track at a position
that satisfies the ordering constraints. If no position satisfies them, the new
track goes at the end, with a warning.

Because the algorithm is greedy in segment creation order, that order is part of
the result. Changing the edge scan order or the segment creation order changes
the layout.

## 4. Geometry

This phase converts the grid to pixels.

Each column is as wide as the widest element in it. Each gutter's width is the
sum of:

- the number of tracks times the track spacing, plus margins on both sides, no
  smaller than the minimum gap between two columns;
- room reserved for the labels of wires exiting on a side;
- room reserved for a db or note sitting next to a node.

If a lane's name is wider than the total width of its columns and gutters, the
shortfall is split evenly between the two outermost gutters.

The vertical axis works the same way. A row is as tall as the tallest element in
it. A channel's height follows the number of tracks plus margins, plus room
reserved for the labels of wires exiting at the bottom, and is no smaller than
the minimum gap between two rows.

Elements are centered in their cell. A db or note sitting next to a node is
placed exactly the attach distance from the node's edge. They sit next to the
node only when that side of the node has exactly one such element and no wire
entering or leaving that side. Otherwise they are centered in the neighboring
cell like any other element.

### Building polylines

Each wire starts at the exit port and ends at the entry port. The points in
between take their horizontal coordinate from the track of the vertical segment,
and their vertical coordinate from the track of the horizontal segment, or from
the center of the target column. Then the engine drops duplicate points and
points lying between two collinear segments, and rounds every coordinate to two
decimal places.

## 5. Second geometry pass

The first pass does not know how long the first horizontal segment of B and C
wires is, so it does not know whether their labels fit. After the first pass,
the engine measures those segments, reserves extra room in the gutters for
labels that do not fit, and reruns all of phase 4.

## 6. Label placement

For each wire with a label, the engine lists the candidate positions, in order of
preference:

- segments near the source first;
- on each segment long enough, try near the start of the segment, then the middle;
- each position tries both sides of the line.

Each position has a cost: the area overlapping nodes, already placed labels and
the title bar, plus a fixed amount for each other wire or lane divider crossing
it. The first position with zero cost is taken immediately. If there is none,
the cheapest one is taken and a warning is recorded.

Labels placed earlier take space from labels placed later, so the order of wires
in the table affects label positions.

## 7. Self-check

The self-check runs on Result and does not read engine state. It reports an
error when:

- two elements overlap;
- an element spills outside its lane;
- a wire segment is neither horizontal nor vertical;
- a wire crosses an element, except for the first and last segments touching
  its own source and target;
- two wires lie on top of each other on the same line, unless they share a
  source or a target, since then they are allowed to join.

It reports a warning when a label overlaps an element, another label, or another
wire.

A correct engine never produces an error here. A self-check error means the
engine has a bug, not that the input table has an error.

## 8. Axis swap

For left-to-right and right-to-left directions, the width and height of every
element and label are swapped before placement. This way a row in the virtual
space has exactly the thickness of a real column. Text is still wrapped in the
real orientation.

Phases 2 through 7 run exactly as for a top-down diagram. After the self-check,
every coordinate, port ratio and label box is converted to the real direction:

- left to right: swap the two axes;
- bottom to top: flip the content area vertically;
- right to left: swap the two axes, then flip.

When flipping, the diagram title bar and the lane name bars stay at the head of
the band; only the content reverses. Axis swaps and flips preserve every
overlap relation and distance, so a self-check on the virtual space still holds
for the real picture.

## How do you trace which phase caused a bug?

Run build with the --layout-json flag to write the coordinates to JSON. This file
has the row and column of each element, and the wire kind, exit side and points
of each wire. Then follow the symptom:

| Symptom | Look at |
|---|---|
| Node in the wrong row, column or lane | Placement: row, main branch, side branch shift direction |
| Wire takes a detour even though a straight path exists | Routing: the chosen wire kind and why the simpler kinds were rejected |
| Wire exits on the wrong side | Routing: exit side trial order and the side-free conditions |
| Two wires overlap | Port placement (same connection point) or track assignment (same lane) |
| Gap too wide or too narrow | Geometry: room reserved for labels and for a db or note sitting alongside |
| Label placed somewhere odd | Label placement: candidate list and costs |
| Wrong only in directions other than top-down | Axis swap |

After fixing, check with images: build with the --png flag to look at it, and
the --verify flag to have draw.io confirm it draws at the computed coordinates.

## How is determinism kept?

The same input must produce the same file on every machine. The rules below keep
it that way. Breaking a rule usually does not turn tests red on your own machine
right away; it only shows up on another machine or with another Go version.

- Do not let map iteration order decide the result. Iterate in table order, or
  sort the keys before iterating.
- Do not change the order of a floating-point accumulation. Floating-point
  addition is not associative, and coordinates are accumulated in a fixed order.
- Wrap every floating-point product in float64() before adding or subtracting
  further. Otherwise the compiler may fuse the multiply and add into one FMA
  instruction on some architectures, giving a different result in the last bit.
  A static test in the repo catches this.
- When two numbers are equal, max and min in the engine return the first one;
  do not use math.Max and math.Min. Those two functions change the sign of zero,
  and that sign can leak into the output file.
- Round coordinates to two decimals at the end, before writing. Any value that
  is compared with a number read back from the file, like the port ratios merge
  compares, must be rounded exactly as when writing.
- Measure text with the embedded width table, never with fonts installed on the
  machine.
