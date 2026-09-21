# Đầu vào ở dạng NFD, dấu tách rời chữ

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Xử lý đơn hàng | |
| A-1 | start | A | Nhận yêu cầu từ người dùng | |
| E1 | edge | | đã xác thực | from=A-1; to=A-2 |
| A-2 | task | A | Kiểm tra tồn kho và giữ chỗ | |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | end | A | Trả kết quả | |
