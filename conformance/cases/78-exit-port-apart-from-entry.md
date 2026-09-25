# Exit wire apart from entry on one side

| id | type | parent | content | metadata |
|---|---|---|---|---|
| P | lane | | Lane P | |
| Q | lane | | Lane Q | |
| P-1 | start | P | Enter | |
| E1 | edge | | send | from=P-1; to=Q-1 |
| Q-1 | external | Q | External system | |
| Q-1.t | text | Q | a note | attach=Q-1 |
| E2 | edge | | next | from=Q-1; to=Q-2 |
| E3 | edge | | error | from=Q-1; to=P-2 |
| Q-2 | end | Q | Done | |
| P-2 | end | P | Error | |
