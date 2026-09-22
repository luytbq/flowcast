# Cổng mặt bên sắp theo hàng của đích

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | A-1 x | |
| A-2 | task | A | A-2 x | |
| E2 | edge | |  | from=A-2; to=A-4 |
| E3 | edge | |  | from=A-2; to=A-1; back=true |
| A-3 | external | A | A-3 x | |
| A-4 | end | A | A-4 x | |
