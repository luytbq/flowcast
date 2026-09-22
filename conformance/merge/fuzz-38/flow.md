# Đoạn dài từ 12 điểm ảnh đã nhận nhãn

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Hệ thống A<br>thanh toán<br>nội bộ | |
| B | lane | | Bộ phận xử lý nghiệp vụ B kéo dài | |
| C | lane | | Lane C<br>dòng hai | |
| A-1 | start | A | A-1 x | |
| E1 | edge | | Có | from=A-1; to=C-2 |
| C-2 | task | A | C-2 x |  |
| E2 | edge | | gửi yêu cầu | from=C-2; to=B-4 |
| E3 | edge | | Có | from=C-2; to=B-3 |
| B-3 | end | B | B-3 x | |
| B-4 | task | B | B-4 x | |
