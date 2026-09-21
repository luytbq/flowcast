# Ghi chú gắn vào node ở lane khác

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=B-1 |
| B-1 | task | B | Xử lý | |
| A-2 | text | A | Ghi chú khai ở lane A | attach=B-1 |
| E2 | edge | | | from=B-1; to=B-2 |
| B-2 | end | B | Xong | |
