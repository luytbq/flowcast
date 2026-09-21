# Mũi tên ngang sang lane kế

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Bắt đầu | |
| E1 | edge | | gọi | from=A-1; to=B-1 |
| B-1 | task | B | Nhận | |
| E2 | edge | | | from=B-1; to=B-2 |
| B-2 | end | B | Xong | |
