# Ghi chú bám node

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Kiểm tra dữ liệu | |
| A-3 | text | A | Chỉ kiểm định dạng, chưa kiểm tồn kho | attach=A-2 |
| E2 | edge | | | from=A-2; to=A-4 |
| A-4 | end | A | Xong | |
