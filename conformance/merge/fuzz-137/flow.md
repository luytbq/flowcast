# Side branch stays in lane, right then left

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| N2 | lane | | New lane 2 | |
| N1 | lane | | New lane 1 | |
| A-1 | start | N1 | In |  |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Three way? | |
| E2 | edge | | One | from=A-2; to=A-3 |
| E3 | edge | | Two | from=A-2; to=A-4 |
| E4 | edge | | Three | from=A-2; to=A-5 |
| A-3 | task | A | Task one |  |
| E5 | edge | | | from=A-3; to=A-6 |
| A-4 | task | N2 | Task two |  |
| E6 | edge | | | from=A-4; to=A-6 |
| A-5 | task | A | Task three | |
| E7 | edge | | | from=A-5; to=A-6 |
| A-6 | end | A | Merge | |
