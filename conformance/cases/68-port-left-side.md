# Cổng trên mặt trái nằm ở mép trái

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| C | lane | | Lane C | |
| A-1 | start | A | A-1 x | |
| C-2 | task | C | C-2 x | |
| C-2.0 | db | C | d0 | attach=C-2 |
| E2 | edge | |  | from=C-2; to=B-3 |
| E3 | edge | |  | from=C-2; to=A-1; back=true |
| B-3 | end | B | B-3 x | |
