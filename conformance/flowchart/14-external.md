# Trỏ sang luồng ngoài sơ đồ

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start |  | Vào |  |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition |  | Cần hoàn tiền? |  |
| E2 | edge | | Có | from=A-2; to=A-3 |
| E3 | edge | | Không | from=A-2; to=A-4 |
| A-3 | external |  | Luồng hoàn tiền |  |
| A-4 | end |  | Kết thúc |  |
