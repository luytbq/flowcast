# Hợp nhánh về node rẽ gần nhất khi rẽ lồng nhau

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | A | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Ngoài? | |
| E2 | edge | | phụ | from=A-2; to=A-3 |
| E3 | edge | | chính | from=A-2; to=A-9 |
| A-3 | condition | A | Trong? | |
| E4 | edge | | p | from=A-3; to=A-4 |
| E5 | edge | | q | from=A-3; to=A-5 |
| A-4 | task | A | P | |
| E6 | edge | | | from=A-4; to=A-6 |
| A-5 | task | A | Q | |
| E7 | edge | | | from=A-5; to=A-6 |
| A-6 | end | A | Hợp trong | |
| A-9 | end | A | Chính | |
