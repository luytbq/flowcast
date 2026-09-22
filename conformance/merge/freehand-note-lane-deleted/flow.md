# Luồng đặt hàng

| id | type | parent | content | metadata |
|---|---|---|---|---|
| USR | lane | | Người dùng | |
| API | lane | | API | |
| USR-1 | start | USR | Gửi yêu cầu đặt hàng | |
| E1 | edge | | POST /orders | from=USR-1; to=API-1 |
| API-1 | task | API | Kiểm tra dữ liệu | |
| API-1.1 | text | API | Chỉ kiểm tra định dạng, chưa kiểm tồn kho | attach=API-1 |
| E2 | edge | | | from=API-1; to=API-2 |
| API-2 | condition | API | Dữ liệu hợp lệ? | style=highlight |
| E3 | edge | | No | from=API-2; to=API-3 |
| E4 | edge | | Yes | from=API-2; to=USR-2 |
| API-3 | task | API | Trả lỗi 400 | |
| E5 | edge | | | from=API-3; to=USR-2 |
| USR-2 | end | USR | Nhận kết quả | |
