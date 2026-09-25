# General routing through channels and gutters

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| C | lane | | Lane C | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=B-1 |
| B-1 | task | B | Dispatch | |
| E2 | edge | | | from=B-1; to=C-1 |
| C-1 | task | C | Deep work | |
| E3 | edge | | | from=C-1; to=C-2 |
| C-2 | task | C | Write result | |
| E4 | edge | | loop back | from=C-2; to=A-2 |
| A-2 | end | A | Get results | |
