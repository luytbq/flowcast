# from và to cùng hỏng, id trùng lật thứ tự

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Đích | |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | end | A | Xong | |
| E9 | edge | | | from=KHONG1; to=KHONG2 |
| A-1 | task | A | Trùng id A-1, đứng sau A-2 | |
