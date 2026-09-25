# An edge cannot end at a db that is attached to an element

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Save | |
| A-2.1 | db | A | ORDERS | attach=A-2 |
| E2 | edge | | | from=A-2; to=A-2.1 |
