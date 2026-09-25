# Note attached to a node

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start |  | Enter |  |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task |  | Check input data |  |
| A-3 | text |  | Checks format only, not stock levels | attach=A-2 |
| E2 | edge | | | from=A-2; to=A-4 |
| A-4 | end |  | Done |  |
