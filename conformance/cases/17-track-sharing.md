# Edges with the same target share a track

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Type? | |
| E2 | edge | | One | from=A-2; to=A-3 |
| E3 | edge | | Two | from=A-2; to=A-4 |
| E4 | edge | | Three | from=A-2; to=A-5 |
| A-3 | task | A | Task one | |
| E5 | edge | | | from=A-3; to=A-6 |
| A-4 | task | A | Task two | |
| E6 | edge | | | from=A-4; to=A-6 |
| A-5 | task | A | Task three | |
| E7 | edge | | | from=A-5; to=A-6 |
| A-6 | task | A | Gather | |
| E8 | edge | | | from=A-6; to=A-7 |
| A-7 | end | A | Done | |
