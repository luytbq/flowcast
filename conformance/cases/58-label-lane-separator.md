# Nhãn tránh đường phân cách lane

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | A-1 x | |
| E1 | edge | |  | from=A-1; to=B-2 |
| E2 | edge | | gửi yêu cầu | from=A-1; to=B-3 |
| B-2 | external | B | B-2 x | |
| B-3 | end | B | B-3 x | |
