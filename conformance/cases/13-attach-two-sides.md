# Node has edges on both sides, db must move out

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| C | lane | | Lane C | |
| B-1 | start | B | Enter | |
| E1 | edge | | | from=B-1; to=B-2 |
| B-2 | condition | B | Fork? | |
| B-3 | db | B | DB.LOG | attach=B-2 |
| E2 | edge | | left | from=B-2; to=A-1 |
| E3 | edge | | right | from=B-2; to=C-1 |
| A-1 | end | A | Left branch | |
| C-1 | end | C | Right branch | |
