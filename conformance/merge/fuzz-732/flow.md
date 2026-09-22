# Đi dây tổng quát qua kênh và máng

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| C | lane | | Lane C | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=B-1 |
| B-1 | task | B | Điều phối | |
| E2 | edge | | | from=B-1; to=C-1 |
| C-1 | task | C | Xử lý sâu | |
| E3 | edge | | | from=C-1; to=C-2 |
| C-2 | task | C | Ghi kết quả | |
| E4 | edge | | vòng về | from=C-2; to=A-2 |
| A-2 | end | A | Nhận kết quả | |
