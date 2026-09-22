# Hợp hai gốc không chung node rẽ

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | A | |
| A-1 | start | A | Vào một | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Rẽ? | |
| E2 | edge | | phụ | from=A-2; to=A-3 |
| E3 | edge | | chính | from=A-2; to=A-4 |
| A-3 | task | A | Nhánh phụ | |
| E4 | edge | | | from=A-3; to=A-9 |
| A-4 | end | A | Chính | |
| A-5 | start | A | Vào hai | |
| E5 | edge | | | from=A-5; to=A-6 |
| A-6 | task | A | Bước | |
| E6 | edge | | | from=A-6; to=A-7 |
| A-7 | task | A | Bước sâu | |
| E7 | edge | | | from=A-7; to=A-9 |
| A-9 | end | A | Hợp hai gốc | |
