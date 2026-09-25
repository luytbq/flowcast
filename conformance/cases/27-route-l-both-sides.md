# L shapes out of both sides

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| C | lane | | Lane C | |
| B-1 | start | B | Enter | |
| E1 | edge | | | from=B-1; to=B-2 |
| B-2 | condition | B | Which way? | |
| E2 | edge | | left | from=B-2; to=A-1 |
| E3 | edge | | right | from=B-2; to=C-1 |
| A-1 | task | A | Left task | |
| E4 | edge | | | from=A-1; to=A-2 |
| A-2 | end | A | Left done | |
| C-1 | task | C | Right task | |
| E5 | edge | | | from=C-1; to=C-2 |
| C-2 | end | C | Right done | |
