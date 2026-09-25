# Long edges span many rows, different targets, stacked gutters

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Fork early? | |
| E2 | edge | | Yes | from=A-2; to=B-3 |
| E3 | edge | | No | from=A-2; to=A-3 |
| A-3 | condition | A | Fork mid? | |
| E4 | edge | | Yes | from=A-3; to=B-2 |
| E5 | edge | | No | from=A-3; to=A-4 |
| A-4 | task | A | Long path | |
| E6 | edge | | | from=A-4; to=B-1 |
| B-1 | task | B | Get last | |
| E7 | edge | | | from=B-1; to=B-2 |
| B-2 | task | B | Get middle | |
| E8 | edge | | | from=B-2; to=B-3 |
| B-3 | end | B | Finish | |
