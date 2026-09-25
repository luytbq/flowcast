# Merge of two roots with no shared fork

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | A | |
| A-1 | start | A | Enter one | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Fork? | |
| E2 | edge | | side | from=A-2; to=A-3 |
| E3 | edge | | main | from=A-2; to=A-4 |
| A-3 | task | A | Side branch | |
| E4 | edge | | | from=A-3; to=A-9 |
| A-4 | end | A | Main | |
| A-5 | start | A | Enter two | |
| E5 | edge | | | from=A-5; to=A-6 |
| A-6 | task | A | Step | |
| E6 | edge | | | from=A-6; to=A-7 |
| A-7 | task | A | Deep step | |
| E7 | edge | | | from=A-7; to=A-9 |
| A-9 | end | A | Roots merge | |
