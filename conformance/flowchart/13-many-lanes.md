# Five lanes, cross traffic

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start |  | Send request |  |
| E1 | edge | | | from=A-1; to=B-1 |
| B-1 | task |  | Auth check |  |
| E2 | edge | | | from=B-1; to=C-1 |
| C-1 | task |  | Business logic |  |
| E3 | edge | | push event | from=C-1; to=D-1 |
| D-1 | task |  | Enqueue |  |
| E4 | edge | | write | from=D-1; to=E-1 |
| E-1 | task |  | Save record |  |
| E5 | edge | | confirm | from=E-1; to=A-2 |
| A-2 | end |  | Get results |  |
