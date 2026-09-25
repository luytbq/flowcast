# Case suite

The input case suite, the output goldens and the tests that run over the whole case
suite. How to use it, when to regenerate goldens and how to add a case are in
[docs/testing.md](../docs/testing.md).

| Path | Contents |
|---|---|
| cases/ | Markdown, csv, xlsx tables; each file pins down one branch of the algorithm. The 9x numbers are deliberately invalid tables |
| flowchart/ | Tables without lanes |
| mermaid/ | Realistic mermaid flowcharts, also the benchmark for the main branch heuristic |
| golden/cases, golden/flowchart, golden/mermaid | The .drawio file and findings report for each case |
| golden/direction/ | Diagrams in the BT, LR, RL directions |
| merge/ | Merge scenarios: an edited table together with an old, manually edited .drawio file |
| cli/ | Transcripts of command line sessions, merge included |
| drawio-fakes/ | A fake drawio CLI for transcripts with --png and --verify |
| *-vectors.json | Fixed regression data for the unit tests of each package |
| golden_test.go | Golden comparison, and the invariants run over every case |
| direction_test.go | Goldens and invariants for the directions |
