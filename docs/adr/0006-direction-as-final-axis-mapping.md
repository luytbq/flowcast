# ADR-0006: Directions other than TD are done with a final axis swap, not by rewriting the engine per axis

- Status: accepted
- Date: 2026-09-23

## Context

The engine knows only one direction: the flow goes top to bottom, branches go to either
side. Section 7 of `docs/core-design.md` proposes a way to support other directions:
name the two axes flow and cross, write an `Axis` module with `ToXY(flow, cross)`, then
funnel every place that speaks in x and y into that module. Place and route are already
axis-neutral; pixels only appear in the geometry phase, in computing track coordinates,
in resolving path references, and in the writer.

When the work started there was a stronger constraint: TD diagrams had to keep matching
the Python reference implementation bit for bit. The geometry phase compares floating
point numbers to make decisions, so merely reordering an addition makes a label jump
and a golden change. Rewriting four phases per axis means touching exactly those lines.

## Decision

The engine still lays out in a virtual TD space. `layout/axis.go` does two things:

- Before layout, for LR and RL, swap the width and height of every element and every
  label. The rows of the virtual space then have exactly the thickness of the real
  columns.
- After layout and after the self-check, swap axes and flip the result: points, boxes,
  anchor points and label offsets.

Text is still measured and wrapped in the real orientation, because text does not rotate
with the diagram.

## Consequences

- Place, route, geometry and labels do not change by a single line. Every golden of the
  TD diagrams matches unchanged, and the conformance suite is still a byte-for-byte
  comparison against the reference implementation.
- The self-check runs on the virtual result. Swapping axes and flipping are
  transformations that preserve distances and overlap relations, so the self-check's
  conclusions do not change with direction.
- BT is the flipped image of TD and RL is the flipped image of LR, which tests can
  check: the same statistics, the same routing style for each edge.
- LR is not the transposed image of TD, and should not be expected to be: node sizes
  swap roles so the layout genuinely differs.
- What is lost: two places still speak TD language. The minimum lane thickness is taken
  from the text width of the lane name, so in a horizontal diagram a long lane name
  makes the band excessively thick. Merge also does not work yet with directions other
  than TD, because it reads the old file in the real coordinate system. Both are noted
  in section 15 of core-design.
- If a second kind with a different layout approach ever appears, or diagonal
  directions are needed, then the Axis module of the original design becomes worth
  doing. By then the port will no longer have to match the reference implementation
  bit for bit, so the main barrier disappears.
