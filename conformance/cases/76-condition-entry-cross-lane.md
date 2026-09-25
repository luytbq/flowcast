# A condition takes a wire from another lane at the top, not a side

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Enter | |
| E1 | edge | | send request | from=A-1; to=B-1 |
| B-1 | condition | B | Valid? | |
| B-2 | text | B | check sig and expiry | attach=B-1 |
| E2 | edge | | no | from=B-1; to=A-2 |
| E3 | edge | | yes | from=B-1; to=B-3 |
| B-3 | task | B | Process | |
| E4 | edge | | | from=B-3; to=A-2 |
| A-2 | end | A | Return result | |
