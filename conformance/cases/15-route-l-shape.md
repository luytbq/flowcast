# L-shaped route out the side then down

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Prepare | |
| E2 | edge | | send | from=A-2; to=B-1 |
| B-1 | task | B | Take | |
| E3 | edge | | | from=B-1; to=B-2 |
| B-2 | task | B | Record it | |
| E4 | edge | | return | from=B-2; to=A-3 |
| A-3 | end | A | Finish | |
