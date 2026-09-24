# Một phần tử nhiều mũi tên vào ra

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Khách hàng | |
| B | lane | | Điều phối | |
| C | lane | | Đối tác | |
| A-1 | start | A | Gửi yêu cầu | |
| E1 | edge | | yêu cầu | from=A-1; to=B-1 |
| B-1 | external | B | Cổng điều phối | |
| B-1.t | text | B | Nhận từ nhiều nguồn, trả về nhiều nhánh | attach=B-1 |
| E2 | edge | | hợp lệ | from=B-1; to=B-2 |
| E3 | edge | | cần đối tác | from=B-1; to=C-1 |
| E4 | edge | | từ chối | from=B-1; to=A-2 |
| B-2 | condition | B | Đủ hạn mức? | |
| E5 | edge | | không | from=B-2; to=A-2 |
| E6 | edge | | có | from=B-2; to=B-3 |
| C-1 | task | C | Đối tác xử lý | |
| E7 | edge | | kết quả | from=C-1; to=B-3 |
| B-3 | task | B | Tổng hợp kết quả | |
| E8 | edge | | | from=B-3; to=A-3 |
| A-2 | end | A | Bị từ chối | |
| A-3 | end | A | Hoàn tất | |
