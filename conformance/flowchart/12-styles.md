# Highlight and dashed lines

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start |  | Enter | style=highlight |
| E1 | edge | | main | from=A-1; to=A-2; style=highlight |
| A-2 | task |  | Handle |  |
| A-3 | text |  | A highlighted note | attach=A-2; style=highlight |
| E2 | edge | | side | from=A-2; to=B-1; style=dashed |
| B-1 | end |  | Side branch |  |
