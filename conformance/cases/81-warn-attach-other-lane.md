# Note attached to a node in another lane

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=B-1 |
| B-1 | task | B | Process | |
| A-2 | text | A | Note declared in lane A | attach=B-1 |
| E2 | edge | | | from=B-1; to=B-2 |
| B-2 | end | B | Done | |
