# A note on a condition goes opposite its side branch, even when the input comes from the lane on that side

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Web | |
| B | lane | | Service | |
| A-1 | task | A | Open the link | |
| E1 | edge | | | from=A-1; to=B-1 |
| B-1 | condition | B | Payment found? | |
| B-1.1 | text | B | Missing, wrong flow or a read error | attach=B-1 |
| E2 | edge | | No | from=B-1; to=B-2 |
| E3 | edge | | Yes | from=B-1; to=B-3 |
| B-2 | end | B | Not found page | |
| B-3 | task | B | Continue the payment | |
| E4 | edge | | | from=B-3; to=B-4 |
| B-4 | end | B | Done | |
