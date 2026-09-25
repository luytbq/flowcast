# Condition with two branches

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start |  | Enter |  |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition |  | Valid? |  |
| E2 | edge | | No | from=A-2; to=A-3 |
| E3 | edge | | Yes | from=A-2; to=A-4 |
| A-3 | end |  | Reject |  |
| A-4 | end |  | Approve |  |
