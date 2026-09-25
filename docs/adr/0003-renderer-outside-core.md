# ADR-0003: The renderer lives outside the core

- Status: accepted
- Date: 2026-09-21

## Context

Today's export function calls an external process: it finds the drawio binary, builds a
command line with the no-sandbox flag, runs it with a 90 second timeout and retries
once. verify_svg also only works when that binary is present.

Running an Electron process inside a web request is wrong on several fronts at once:
time, memory, and attack surface.

The core has to serve both the CLI and HTTP without either side rewriting the other's
policy. The rule that makes that possible is that the core does not call external
processes, does not touch the filesystem, does not read environment variables, and does
not print to stdout.

## Decision

The core returns the target text. Turning that text into an image lives in the
flowtable_render package, outside the core.

The CLI calls flowtable_render directly. The web service runs it in a worker separate
from the request, or does not run it.

verify is a testing tool, not a step of build.

## Consequences

- The whole core test suite runs without the drawio CLI and without temporary files.
- The web service can be deployed without Electron in the image of the process serving
  requests.
- Image previews on the web become asynchronous work, not part of the build call. This
  is a known cost.
- The renderer seam currently has only one adapter, so it is a hypothetical seam. Do
  not build an abstract renderer interface until a second adapter actually exists.
