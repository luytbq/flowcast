# Năm lane, lưu lượng chéo

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start |  | Gửi yêu cầu |  |
| E1 | edge | | | from=A-1; to=B-1 |
| B-1 | task |  | Xác thực |  |
| E2 | edge | | | from=B-1; to=C-1 |
| C-1 | task |  | Xử lý nghiệp vụ |  |
| E3 | edge | | đẩy sự kiện | from=C-1; to=D-1 |
| D-1 | task |  | Xếp hàng |  |
| E4 | edge | | ghi | from=D-1; to=E-1 |
| E-1 | task |  | Lưu bản ghi |  |
| E5 | edge | | xác nhận | from=E-1; to=A-2 |
| A-2 | end |  | Nhận kết quả |  |
