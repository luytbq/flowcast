# Two-branch condition

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | In | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Valid? | |
| E2 | edge | | No | from=A-2; to=A-3 |
| E3 | edge | | Yes | from=A-2; to=A-4 |
| A-3 | end | A | Decline | |
| A-4 | end | A | Accept | |
