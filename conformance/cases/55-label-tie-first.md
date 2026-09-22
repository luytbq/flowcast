# Hai ứng viên nhãn cùng chi phí thì lấy ứng viên trước

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Bộ phận xử lý nghiệp vụ A kéo dài | |
| B | lane | | Lane B | |
| C | lane | | Hệ thống C<br>thanh toán<br>nội bộ | |
| B-1 | start | B | B-1 x | |
| E1 | edge | | gửi yêu cầu | from=B-1; to=B-2 |
| B-2 | external | B | B-2 x | |
