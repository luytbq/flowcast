# Branches merge back to the fork's column

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Fork? | |
| E2 | edge | | No | from=A-2; to=A-3 |
| E3 | edge | | Yes | from=A-2; to=A-4 |
| A-3 | task | A | Branch one | |
| E4 | edge | | | from=A-3; to=A-5 |
| A-4 | task | A | Branch two | |
| E5 | edge | | | from=A-4; to=A-5 |
| A-5 | end | A | Merge point | |
