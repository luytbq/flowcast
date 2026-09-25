# ADR-0004: The web service does not expose merge

- Status: accepted
- Date: 2026-09-21

## Context

merge regenerates a diagram while keeping manual edits: element positions, lane widths,
waypoints, and even cells the user drew on their own. It exists to serve the loop of
generating a file, dragging things by hand in draw.io, then regenerating.

That loop assumes the user keeps their own .drawio file across many runs. On the web,
each request is a separate event and no file is kept.

Exposing merge on the web would mean a second upload field, an extra API branch, and a
piece of UI just to display the list of places needing manual fixes in the MergeReport.

## Decision

The web service only exposes the fresh generation path.

The core still has full merge, and the overrides parameter is in the build signature
from the start. The CLI uses it. The web service simply does not expose it.

## Consequences

- The web API has exactly one shape: one file up, one file down.
- Opening it up later costs nothing in design. The build signature is ready; only the
  UI is missing.
- merge must still be made correct in the core, including rerunning check after
  applying Overrides. This decision limits the scope of the interface, not the quality
  of merge.
- Web users cannot keep manual edits between two builds. For that loop, use the CLI.
