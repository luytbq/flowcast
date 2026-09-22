# Cổng trên mặt trái nằm ở mép trái

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| N0 | lane | | Lane mới 0 | |
| B | lane | | Lane B | |
| C | lane | | Lane C | |
| A-1 | start | A | A-1 x | |
| C-2 | task | N0 | C-2 x kèm thêm chữ cho dài ra |  |
| C-2.0 | db | C | d0 | attach=C-2 |
| E2 | edge | |  | from=C-2; to=B-3 |
| E3 | edge | |  | from=C-2; to=A-1; back=true |
| B-3 | end | B | B-3 x | |
| C-92 | task | C | Mới 2 | |
| E92 | edge | | | from=C-92; to=C-2 |
