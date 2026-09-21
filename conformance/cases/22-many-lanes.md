# Năm lane, lưu lượng chéo

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Người dùng | |
| B | lane | | Cổng | |
| C | lane | | Dịch vụ | |
| D | lane | | Hàng đợi | |
| E | lane | | Kho dữ liệu | |
| A-1 | start | A | Gửi yêu cầu | |
| E1 | edge | | | from=A-1; to=B-1 |
| B-1 | task | B | Xác thực | |
| E2 | edge | | | from=B-1; to=C-1 |
| C-1 | task | C | Xử lý nghiệp vụ | |
| E3 | edge | | đẩy sự kiện | from=C-1; to=D-1 |
| D-1 | task | D | Xếp hàng | |
| E4 | edge | | ghi | from=D-1; to=E-1 |
| E-1 | task | E | Lưu bản ghi | |
| E5 | edge | | xác nhận | from=E-1; to=A-2 |
| A-2 | end | A | Nhận kết quả | |
