# Nhánh phụ dẫn sang lane bên trái

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| B-1 | start | B | Vào | |
| E1 | edge | | | from=B-1; to=B-2 |
| B-2 | condition | B | Lỗi? | |
| E2 | edge | | Có | from=B-2; to=B-3 |
| E3 | edge | | Không | from=B-2; to=B-4 |
| B-3 | task | B | Dựng thông báo lỗi | |
| E4 | edge | | trả lỗi | from=B-3; to=A-1 |
| A-1 | end | A | Nhận lỗi | |
| B-4 | end | B | Tiếp tục | |
