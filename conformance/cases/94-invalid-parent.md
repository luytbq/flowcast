# Element whose parent is not a lane

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A-1 | Bad parent | |
