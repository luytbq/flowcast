# Out the side, then down onto the target node's top

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Step one | |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | task | A | Step two | |
| E3 | edge | | forward | from=A-3; to=B-1 |
| B-1 | task | B | Step three | |
| E4 | edge | | | from=B-1; to=B-2 |
| B-2 | condition | B | Done? | |
| E5 | edge | | No | from=B-2; to=B-1; back=true |
| E6 | edge | | Yes | from=B-2; to=B-3 |
| B-3 | end | B | Finish | |
