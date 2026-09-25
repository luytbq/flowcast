Fake drawio CLIs for the CLI transcripts that use --png and --verify. A transcript
carrying the line "# drawio: <name>" makes the test run the CLI with a PATH that
holds only the <name> directory here, so the real drawio on the machine cannot
leak in.

| name | behavior |
|---|---|
| ok | writes the command line it received into the PNG file, or an SVG without data-cell-id |
| stdout-fail | prints an error to stdout, exit code 1 |
| fail | prints an error to stderr, exit code 1 |
| silent | exit code 0 but writes no file |
| none | no drawio |

The SVG comparison part of the render check is pinned separately, on real SVG
exported by drawio, through verify-vectors.json.
