# Data table attached to a node

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start |  | Enter |  |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task |  | Write order |  |
| A-3 | db |  | DB.ORDER | attach=A-2 |
| E2 | edge | | | from=A-2; to=A-4 |
| A-4 | end |  | Done |  |
