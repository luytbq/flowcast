# Nhãn chồng node bị phạt nặng hơn cắt dây

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Hệ thống A<br>thanh toán<br>nội bộ | |
| A-3 | task | A | A-3 x | |
| A-3.0 | text | A | d0 | attach=A-3 |
| E3 | edge | | trả kết quả kèm mã lỗi chi tiết | from=A-3; to=A-4 |
| E4 | edge | |  | from=A-3; to=A-5 |
| E5 | edge | | trả kết quả kèm mã lỗi chi tiết | from=A-3; to=A-4 |
| A-4 | task | A | A-4 x | |
| A-4.0 | db | A | d0 | attach=A-4 |
| E6 | edge | |  | from=A-4; to=A-5 |
| A-5 | task | A | A-5 x | |
