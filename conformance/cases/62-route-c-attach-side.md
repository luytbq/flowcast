# Chữ L không ra mặt đang có db

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| B-1 | start | B | B-1 x | |
| E1 | edge | |  | from=B-1; to=A-4 |
| A-4 | task | A | A-4 x | |
| A-4.0 | text | A | d0 | attach=A-4 |
| A-4.1 | text | A | d1 | attach=A-4 |
| A-4.2 | text | A | d2 | attach=A-4 |
| E2 | edge | |  | from=A-4; to=A-7 |
| A-6 | task | A | A-6 x | |
| E5 | edge | |  | from=A-6; to=A-7 |
| E6 | edge | |  | from=A-6; to=A-8 |
| A-7 | end | A | A-7 x | |
| A-8 | task | A | A-8 x | |
