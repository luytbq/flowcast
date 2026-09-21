# Nhánh phụ dẫn sang lane bên phải

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Cần lưu? | |
| E2 | edge | | Có | from=A-2; to=A-3 |
| E3 | edge | | Không | from=A-2; to=A-4 |
| A-3 | task | A | Chuẩn bị bản ghi | |
| E4 | edge | | lưu | from=A-3; to=B-1 |
| B-1 | end | B | Đã lưu | |
| A-4 | end | A | Bỏ qua | |
