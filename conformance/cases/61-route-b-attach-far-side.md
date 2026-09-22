# db nằm xa ở mặt đối diện vẫn chặn mũi tên ngang

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | A-1 x | |
| E1 | edge | |  | from=A-1; to=A-3 |
| E2 | edge | |  | from=A-1; to=A-2 |
| A-2 | end | A | A-2 x | |
| A-3 | end | A | A-3 x | |
| A-3.0 | text | A | d0 | attach=A-3 |
| A-3.1 | db | A | d1 | attach=A-3 |
| A-3.2 | text | A | d2 | attach=A-3 |
