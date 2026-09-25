# General routing via channels and gutters

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| C | lane | | Lane C | |
| A-1 | start | A | In | |
| E1 | edge | | | from=A-1; to=B-1 |
| B-1 | task | B | Dispatch | |
| E2 | edge | | | from=B-1; to=C-1 |
| C-1 | task | C | Deep process | |
| E3 | edge | | | from=C-1; to=C-2 |
| C-2 | task | C | Save result | |
| E4 | edge | | loop back | from=C-2; to=A-2 |
| A-2 | end | A | Get result | |
