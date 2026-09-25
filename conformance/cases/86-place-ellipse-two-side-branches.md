# An ellipse with two side branches takes no side arrow, keeping both sides

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=B-1 |
| B-1 | external | B | System X | |
| E2 | edge | | a | from=B-1; to=B-2 |
| E3 | edge | | b | from=B-1; to=B-3 |
| E4 | edge | | c | from=B-1; to=B-4 |
| B-2 | end | B | Done a | |
| B-3 | end | B | Done b | |
| B-4 | end | B | Done c | |
