# Warnings about the graph shape

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Fork? | |
| E2 | edge | | | from=A-2; to=A-3 |
| E3 | edge | | Yes | from=A-2; to=A-4 |
| A-3 | end | A | Ends early | |
| E4 | edge | | leave | from=A-3; to=A-4 |
| A-4 | end | A | The end | |
| E5 | edge | | loop to start | from=A-4; to=A-1 |
