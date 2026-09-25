# Branch direction stops at the merge node

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | A | |
| B | lane | | B | |
| C | lane | | C | |
| B-1 | start | B | Enter | |
| E1 | edge | | | from=B-1; to=B-2 |
| B-2 | condition | B | Fork? | |
| E2 | edge | | side | from=B-2; to=B-3 |
| E3 | edge | | main | from=B-2; to=B-5 |
| B-3 | task | B | Side branch | |
| E4 | edge | | | from=B-3; to=B-4 |
| B-5 | task | B | Main branch | |
| E5 | edge | | | from=B-5; to=B-4 |
| B-4 | task | B | Merge | |
| E6 | edge | | | from=B-4; to=A-1 |
| A-1 | end | A | Exit left | |
