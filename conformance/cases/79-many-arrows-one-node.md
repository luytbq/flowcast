# One element with many arrows in and out

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Customer | |
| B | lane | | Dispatcher | |
| C | lane | | Partner | |
| A-1 | start | A | Send request | |
| E1 | edge | | request | from=A-1; to=B-1 |
| B-1 | external | B | Dispatch gateway | |
| B-1.t | text | B | Takes many sources, returns many branches | attach=B-1 |
| E2 | edge | | valid | from=B-1; to=B-2 |
| E3 | edge | | needs partner | from=B-1; to=C-1 |
| E4 | edge | | rejected | from=B-1; to=A-2 |
| B-2 | condition | B | Within limit? | |
| E5 | edge | | no | from=B-2; to=A-2 |
| E6 | edge | | yes | from=B-2; to=B-3 |
| C-1 | task | C | Partner works | |
| E7 | edge | | result | from=C-1; to=B-3 |
| B-3 | task | B | Aggregate results | |
| E8 | edge | | | from=B-3; to=A-3 |
| A-2 | end | A | Rejected | |
| A-3 | end | A | Completed | |
