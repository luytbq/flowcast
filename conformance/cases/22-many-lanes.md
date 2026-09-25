# Five lanes, cross traffic

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | End user | |
| B | lane | | Gateway | |
| C | lane | | Service | |
| D | lane | | Job queue | |
| E | lane | | Data store | |
| A-1 | start | A | Send request | |
| E1 | edge | | | from=A-1; to=B-1 |
| B-1 | task | B | Auth check | |
| E2 | edge | | | from=B-1; to=C-1 |
| C-1 | task | C | Business logic | |
| E3 | edge | | push event | from=C-1; to=D-1 |
| D-1 | task | D | Enqueue | |
| E4 | edge | | write | from=D-1; to=E-1 |
| E-1 | task | E | Save record | |
| E5 | edge | | confirm | from=E-1; to=A-2 |
| A-2 | end | A | Get results | |
