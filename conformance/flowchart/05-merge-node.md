# Branches merge back to the fork's column

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start |  | Enter |  |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition |  | Fork? |  |
| E2 | edge | | No | from=A-2; to=A-3 |
| E3 | edge | | Yes | from=A-2; to=A-4 |
| A-3 | task |  | Branch one |  |
| E4 | edge | | | from=A-3; to=A-5 |
| A-4 | task |  | Branch two |  |
| E5 | edge | | | from=A-4; to=A-5 |
| A-5 | end |  | Merge point |  |
