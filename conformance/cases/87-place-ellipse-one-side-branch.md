# An ellipse with one side branch still takes a side arrow

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=B-1 |
| B-1 | external | B | System X | |
| E2 | edge | | a | from=B-1; to=B-2 |
| E3 | edge | | b | from=B-1; to=B-3 |
| B-2 | end | B | Done a | |
| B-3 | end | B | Done b | |
