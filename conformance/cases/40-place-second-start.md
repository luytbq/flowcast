# Start thứ hai vẫn về hàng 0 dù xếp sau

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | A | |
| B | lane | | B | |
| A-1 | start | A | Vào A | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Bước | |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | end | A | Xong A | |
| B-1 | start | B | Vào B, bắt đầu muộn | |
| E3 | edge | | | from=B-1; to=B-2 |
| B-2 | end | B | Xong B | |
