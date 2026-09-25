# Highlight and dashed lines

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Enter | style=highlight |
| E1 | edge | | main | from=A-1; to=A-2; style=highlight |
| A-2 | task | A | Handle | |
| A-3 | text | A | A highlighted note | attach=A-2; style=highlight |
| E2 | edge | | side | from=A-2; to=B-1; style=dashed |
| B-1 | end | B | Side branch | |
