# Dây chỉ chạm mép hộp nhãn thì không tính là cắt

| id | type | parent | content | metadata |
|---|---|---|---|---|
| C | lane | | Lane C<br>dòng hai | |
| C-1 | start | C | C-1 x | |
| C-1.1 | db | C | d1 | attach=C-1 |
| C-1.2 | db | C | d2 | attach=C-1 |
| E1 | edge | | Có | from=C-1; to=C-2 |
| E2 | edge | | trả kết quả kèm mã lỗi chi tiết | from=C-1; to=C-3 |
| C-2 | task | C | C-2 x | |
| E3 | edge | |  | from=C-2; to=C-3 |
| C-3 | task | C | C-3 x | |
