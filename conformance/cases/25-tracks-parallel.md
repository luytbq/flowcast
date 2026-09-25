# Edges with different targets run in parallel in one gutter

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| B-1 | start | B | Enter | |
| E1 | edge | | | from=B-1; to=B-2 |
| B-2 | condition | B | Block one? | |
| E2 | edge | | Yes | from=B-2; to=A-1 |
| E3 | edge | | No | from=B-2; to=B-3 |
| A-1 | end | A | Error one | |
| B-3 | condition | B | Block two? | |
| E4 | edge | | Yes | from=B-3; to=A-2 |
| E5 | edge | | No | from=B-3; to=B-4 |
| A-2 | end | A | Error two | |
| B-4 | condition | B | Block three? | |
| E6 | edge | | Yes | from=B-4; to=A-3 |
| E7 | edge | | No | from=B-4; to=B-5 |
| A-3 | end | A | Error three | |
| B-5 | end | B | All pass | |
