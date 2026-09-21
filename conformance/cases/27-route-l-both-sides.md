# Chữ L ra cả hai mặt bên

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| C | lane | | Lane C | |
| B-1 | start | B | Vào | |
| E1 | edge | | | from=B-1; to=B-2 |
| B-2 | condition | B | Hướng nào? | |
| E2 | edge | | trái | from=B-2; to=A-1 |
| E3 | edge | | phải | from=B-2; to=C-1 |
| A-1 | task | A | Việc trái | |
| E4 | edge | | | from=A-1; to=A-2 |
| A-2 | end | A | Xong trái | |
| C-1 | task | C | Việc phải | |
| E5 | edge | | | from=C-1; to=C-2 |
| C-2 | end | C | Xong phải | |
