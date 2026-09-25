# db next to a node does not hug a side with wires

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | A-1 x | |
| A-1.0 | text | A | d0 | attach=A-1 |
| A-1.1 | text | A | d1 | attach=A-1 |
| E1 | edge | | send request | from=A-1; to=A-2 |
| E2 | edge | | Yes | from=A-1; to=A-3 |
| A-2 | end | A | A-2 x | |
| A-3 | task | A | A-3 x | |
