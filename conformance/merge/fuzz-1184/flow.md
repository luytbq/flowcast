# Port on left side sits at left edge

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| N0 | lane | | New lane 0 | |
| B | lane | | Lane B | |
| C | lane | | Lane C | |
| A-1 | start | A | A-1 x | |
| C-2 | task | N0 | C-2 x plus extra text to pad it |  |
| C-2.0 | db | C | d0 | attach=C-2 |
| E2 | edge | |  | from=C-2; to=B-3 |
| E3 | edge | |  | from=C-2; to=A-1; back=true |
| B-3 | end | B | B-3 x | |
| C-92 | task | C | New 2 | |
| E92 | edge | | | from=C-92; to=C-2 |
