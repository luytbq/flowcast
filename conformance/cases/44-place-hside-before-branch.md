# A side with a horizontal arrow pushes the side branch to the other side

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | A | |
| B | lane | | B | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Fork? | |
| E2 | edge | | across | from=A-2; to=B-1 |
| E3 | edge | | side | from=A-2; to=A-3 |
| E4 | edge | | main | from=A-2; to=A-4 |
| B-1 | end | B | Across | |
| A-3 | end | A | Side | |
| A-4 | end | A | Main | |
