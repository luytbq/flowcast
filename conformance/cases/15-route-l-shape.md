# Đi dây chữ L ra mặt bên rồi xuống

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Chuẩn bị | |
| E2 | edge | | gửi | from=A-2; to=B-1 |
| B-1 | task | B | Nhận | |
| E3 | edge | | | from=B-1; to=B-2 |
| B-2 | task | B | Ghi nhận | |
| E4 | edge | | trả về | from=B-2; to=A-3 |
| A-3 | end | A | Kết thúc | |
