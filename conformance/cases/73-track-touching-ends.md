# Hai đoạn chạm đầu vẫn tính là chồng

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | A-1 x | |
| E1 | edge | |  | from=A-1; to=A-3 |
| A-2 | task | A | A-2 x | |
| A-2.0 | db | A | d0 | attach=A-2 |
| E2 | edge | |  | from=A-2; to=A-4 |
| A-3 | end | A | A-3 x | |
| A-4 | external | A | A-4 x | |
