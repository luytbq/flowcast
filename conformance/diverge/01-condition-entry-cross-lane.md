# condition nhận dây từ lane khác qua đỉnh, không qua mặt bên

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Vào | |
| E1 | edge | | gửi yêu cầu | from=A-1; to=B-1 |
| B-1 | condition | B | Hợp lệ? | |
| B-2 | text | B | kiểm chữ ký và hạn | attach=B-1 |
| E2 | edge | | không | from=B-1; to=A-2 |
| E3 | edge | | có | from=B-1; to=B-3 |
| B-3 | task | B | Xử lý | |
| E4 | edge | | | from=B-3; to=A-2 |
| A-2 | end | A | Trả kết quả | |
