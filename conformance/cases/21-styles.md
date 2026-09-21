# Tô nhấn và nét đứt

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Vào | style=highlight |
| E1 | edge | | chính | from=A-1; to=A-2; style=highlight |
| A-2 | task | A | Xử lý | |
| A-3 | text | A | Ghi chú được tô nhấn | attach=A-2; style=highlight |
| E2 | edge | | phụ | from=A-2; to=B-1; style=dashed |
| B-1 | end | B | Nhánh phụ | |
