# Side branch leads to the right lane

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Save it? | |
| E2 | edge | | Yes | from=A-2; to=A-3 |
| E3 | edge | | No | from=A-2; to=A-4 |
| A-3 | task | A | Prepare record | |
| E4 | edge | | save | from=A-3; to=B-1 |
| B-1 | end | B | Saved | |
| A-4 | end | A | Ignore | |
