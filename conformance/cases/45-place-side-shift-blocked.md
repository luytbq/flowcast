# Nhánh phụ dạt qua nhiều ô bị chặn rồi mới xuống hàng

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | A | |
| B | lane | | B | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Rẽ? | |
| E2 | edge | | phụ | from=A-2; to=A-5 |
| E3 | edge | | chính | from=A-2; to=A-3 |
| A-3 | task | A | Chính | |
| E4 | edge | | | from=A-3; to=B-1 |
| B-1 | end | B | Sang B | |
| A-5 | condition | A | Ba hướng? | |
| E5 | edge | | x | from=A-5; to=A-6 |
| E6 | edge | | y | from=A-5; to=A-7 |
| E7 | edge | | z | from=A-5; to=A-8 |
| A-6 | end | A | X | |
| A-7 | end | A | Y | |
| A-8 | end | A | Z | |
