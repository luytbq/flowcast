# Points to a flow outside the diagram

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start |  | Enter |  |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition |  | Needs refund? |  |
| E2 | edge | | Yes | from=A-2; to=A-3 |
| E3 | edge | | No | from=A-2; to=A-4 |
| A-3 | external |  | Refund process |  |
| A-4 | end |  | Finish |  |
