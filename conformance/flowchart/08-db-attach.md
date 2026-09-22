# Bảng dữ liệu bám node

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start |  | Vào |  |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task |  | Ghi đơn hàng |  |
| A-3 | db |  | DB.ORDER | attach=A-2 |
| E2 | edge | | | from=A-2; to=A-4 |
| A-4 | end |  | Xong |  |
