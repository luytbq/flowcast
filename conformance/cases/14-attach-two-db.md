# Một node có hai bảng dữ liệu

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Đối soát | |
| A-3 | db | A | DB.ORDER | attach=A-2 |
| A-4 | db | A | DB.PAYMENT | attach=A-2 |
| E2 | edge | | | from=A-2; to=A-5 |
| A-5 | end | A | Xong | |
