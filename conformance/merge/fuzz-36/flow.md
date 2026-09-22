# Mặt đã có mũi tên ngang đẩy nhánh phụ sang mặt kia

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | A | |
| B | lane | | B | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Rẽ? | |
| E2 | edge | | ngang | from=A-2; to=B-1 |
| E3 | edge | | phụ | from=A-2; to=A-3 |
| E4 | edge | | chính | from=A-2; to=A-4 |
| B-1 | end | B | Ngang | |
| A-3 | end | A | Phụ |  |
| A-4 | end | A | Chính | |
| B-90 | task | B | Mới 0 | |
| E90 | edge | | | from=B-90; to=A-3 |
