# Nhánh phụ bị chặn dạt qua nhiều cột trước khi xuống hàng

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-2 | task | A | A-2 x | |
| E2 | edge | |  | from=A-2; to=A-6 |
| A-3 | condition | A | A-3 x | |
| E3 | edge | |  | from=A-3; to=A-4 |
| E4 | edge | | có | from=A-3; to=A-5 |
| E5 | edge | | nhánh dài hơn | from=A-3; to=A-6 |
| A-4 | condition | A | A-4 x | |
| E7 | edge | |  | from=A-4; to=A-6 |
| E8 | edge | |  | from=A-4; to=A-5 |
| A-5 | end | A | A-5 x | |
| A-5.1 | db | A | d4 | attach=A-5 |
| A-6 | task | A | A-6 x | |
