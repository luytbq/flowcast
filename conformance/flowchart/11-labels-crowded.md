# Nhiều nhãn cạnh tranh chỗ

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start |  | Vào |  |
| E1 | edge | | yêu cầu khởi tạo phiên | from=A-1; to=B-1 |
| B-1 | condition |  | Phiên hợp lệ? |  |
| E2 | edge | | phiên hết hạn, cấp lại | from=B-1; to=B-2 |
| E3 | edge | | phiên còn hiệu lực | from=B-1; to=B-3 |
| B-2 | task |  | Cấp phiên mới |  |
| E4 | edge | | | from=B-2; to=B-3 |
| B-3 | task |  | Trả dữ liệu |  |
| E5 | edge | | kết quả kèm mã phiên | from=B-3; to=A-2 |
| A-2 | end |  | Nhận |  |
