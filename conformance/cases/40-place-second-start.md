# A second start still goes to row 0 though listed later

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | A | |
| B | lane | | B | |
| A-1 | start | A | Enter A | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Step | |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | end | A | Done A | |
| B-1 | start | B | Enter B, starts late | |
| E3 | edge | | | from=B-1; to=B-2 |
| B-2 | end | B | Done B | |
