# Cạnh quay ngược tạo vòng lặp

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start |  | Vào |  |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task |  | Thử gửi |  |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | condition |  | Thành công? |  |
| E3 | edge | | Chưa | from=A-3; to=A-2; back=true |
| E4 | edge | | Rồi | from=A-3; to=A-4 |
| A-4 | end |  | Xong |  |
