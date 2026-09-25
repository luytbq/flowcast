# A port skips the side center if it has a fixed edge

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| B-1 | start | B | B-1 x | |
| E1 | edge | |  | from=B-1; to=A-2 |
| A-2 | task | A | A-2 x | |
| E2 | edge | |  | from=A-2; to=A-3 |
| E3 | edge | |  | from=A-2; to=A-3 |
| A-3 | task | A | A-3 x | |
