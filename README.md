# flowcast

flowcast turns a flow diagram written as a table or in mermaid into a draw.io file
with the layout already done. Every coordinate is computed by the tool, the same input
always produces the same file, and nobody has to drag nodes around or edit XML by hand.

Automatic diagram tools often produce ugly layouts: wires cutting through nodes,
overlapping labels, messy branches. flowcast has its own layout and routing engine,
self-checks the geometry before writing the file, and can even check whether draw.io
actually draws what it computed.

## What it does

- Reads flow tables written in markdown, csv or xlsx, and mermaid flowcharts.
- Draws in four directions: top-down (TD), bottom-up (BT), left-to-right (LR),
  right-to-left (RL). In a diagram with lanes, the lanes are vertical or horizontal
  bands depending on the direction. A table that declares no lane is drawn as a
  flowchart.
- Regenerates a diagram while keeping manual edits made in draw.io: node positions,
  lane widths, wire waypoints, and shapes the user drew on their own.
- Self-checks the geometry before writing: wires crossing nodes, two overlapping
  wires, overlapping labels.
- Exports PNG, and compares the wires draw.io draws against the computed coordinates.
- Usable from the command line, as a web service, and as a Go library.

## Build

Requires Go 1.25 or later.

```
go build -o flowcast ./cmd/flowcast      # command line
go build -o flowcastd ./cmd/flowcastd    # web service
```

Image export and render checking also need the drawio CLI (the draw.io desktop app)
on the machine. Without it every other feature still works.

## Running from the command line

```
./flowcast check table.md                    # only check the table, do not draw
./flowcast build table.md                    # write table.drawio next to the input file
./flowcast build diagram.mmd                 # mermaid flowchart, or a mermaid block in a .md file
./flowcast build table.md --direction LR
./flowcast build table.md -o out.drawio --png --verify
```

The input table format is described in [docs/flow-table-format.md](docs/flow-table-format.md).

### Common options

| Option | Meaning |
|---|---|
| -o, --output FILE | The .drawio file to write. Defaults to the input file name with the extension changed. |
| --title T | Diagram title. Defaults to the title in the source file. |
| --direction D | TD, BT, LR or RL. Defaults to the source; if the source says nothing, TD. |
| --mode merge or force | Used when the target file already exists. merge keeps manual edits, force regenerates everything. If not given, the tool asks, or stops if not running in a terminal. |
| --no-backup | Do not write a .bak copy of the old file before overwriting. |
| --png [FILE] | Export a PNG image with the drawio CLI. Without FILE the image is placed next to the .drawio file. |
| --verify | Export SVG with the drawio CLI and compare each wire against the computed coordinates. |
| --layout-json FILE | Write the computed coordinates to JSON, for other tools to read. |
| --sheet S | For xlsx: name of the sheet holding the table. |
| --delimiter D | For csv: the delimiter, when you do not want the tool to guess. |
| --encoding E | For csv: the encoding, when you do not want the tool to guess. |

### Layout options

Units are pixels. Run flowcast --help to see the value range of each option.

| Option | Default | Meaning |
|---|---|---|
| --task-min-w | 120 | Minimum width of a task box |
| --task-max-w | 240 | Maximum width of a task box |
| --cond-wrap | 150 | Wrap width inside a diamond |
| --term-wrap | 170 | Wrap width of start, end and external |
| --db-wrap | 130 | Wrap width of db |
| --text-wrap | 260 | Wrap width of notes |
| --label-wrap | 180 | Wrap width of labels on wires |
| --track-gap | 12 | Distance between two parallel wires |
| --gutter-margin | 15 | Margin from the edge of the vertical gap between two columns to the first wire |
| --channel-margin | 12 | Margin from the edge of the horizontal gap between two rows to the first wire |
| --min-gutter | 24 | Minimum space between two columns |
| --min-channel | 30 | Minimum space between two rows |
| --attach-gap | 40 | Distance from a db or note to the node it attaches to |
| --lane-header | 30 | Thickness of the lane name bar |
| --pool-header | 30 | Thickness of the diagram title bar |
| --min-lane-w | 120 | Minimum thickness of a lane |
| --label-pad | 4 | Padding around the text of a label on a wire |

### Exit codes

| Code | When |
|---|---|
| 0 | Success |
| 1 | Could not read the file, the table has errors, or could not write the file |
| 2 | The file was written but the geometry self-check reported errors |
| 3 | The file was written but image export failed, or draw.io drew something that deviates from the computed coordinates |
| 4 | Nothing written: the target file already exists and no mode was chosen, or the old file could not be read for merging |

Tables over 50000 rows or files over 64 MB are rejected.

## Running the web service

```
./flowcastd -addr :8080
curl -F file=@table.md 'localhost:8080/api/build?download=1' -o table.drawio
```

Opening the service's root address in a browser shows the upload page.

| Path | What it does |
|---|---|
| POST /api/build | Builds the diagram, returns JSON with the .drawio file and the findings. Add download=1 to download the .drawio file directly. |
| POST /api/check | Only checks the table. |
| GET /api/fields | List of layout options with defaults and value ranges. |
| GET /healthz | Checks that the service is alive. |

Both POST paths accept multipart with a file field, plus the optional fields title,
direction, sheet, delimiter, encoding and every layout option (without the two leading
dashes). The upload page and the command line build their option lists from the same
declaration, so the two always match.

The -concurrent flag sets the number of builds running at the same time, defaulting
to the number of CPUs. A request that has to wait more than 5 seconds for a slot gets
status 503.

The web service is stricter than the command line: files up to 1 MB, tables up to
1000 rows and 1000 edges, each build up to 10 seconds. Exceeding a limit gets status
413. A table with errors gets 422, invalid options get 400. The service does not merge,
does not read or write files on the server, and does not call drawio.

## Documentation

| To learn | Read |
|---|---|
| How to write the input table | [docs/flow-table-format.md](docs/flow-table-format.md) |
| What the terms used in code and docs mean | [CONTEXT.md](CONTEXT.md) |
| Where the code lives, which packages the data passes through | [docs/structure.md](docs/structure.md) |
| How the engine computes coordinates and routes wires | [docs/algorithm.md](docs/algorithm.md) |
| Running checks, regenerating goldens, adding cases | [docs/testing.md](docs/testing.md) |
| Overall design of the core | [docs/core-design.md](docs/core-design.md) |
| How regenerating while keeping manual edits works | [docs/merge-design.md](docs/merge-design.md) |
| Reading csv and xlsx | [docs/input-formats-design.md](docs/input-formats-design.md) |
| Settled decisions and their reasons | [docs/adr/](docs/adr/) |

## License

MIT, see [LICENSE](LICENSE).

The tool measures text with a character width table generated from the Verdana font,
a commercial font from Microsoft, and embeds this table in the binary. A table of
measurements is not a font file, but someone who understands the law should review it
before public distribution. Why the table has to be embedded, and the fallback plan of
switching to a free font, are in section 11 of
[docs/core-design.md](docs/core-design.md).
