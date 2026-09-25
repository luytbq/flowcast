# A db in the flow takes edges in and out, and can have a note attached

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | App | |
| B | lane | | Storage | |
| A-1 | start | A | Request | |
| E1 | edge | | save | from=A-1; to=B-1 |
| B-1 | db | B | ORDERS | |
| B-1.1 | text | B | Written in one transaction | attach=B-1 |
| E2 | edge | | read back | from=B-1; to=A-2 |
| A-2 | task | A | Build the reply | |
| E3 | edge | | | from=A-2; to=A-3 |
| A-3 | end | A | Done | |
