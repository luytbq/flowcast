# ADR-0008: chuẩn đúng là golden do flowcast sinh và các bất biến

- Trạng thái: đã chấp nhận
- Ngày: 2026-09-24
- Thay thế: ADR-0007

## Bối cảnh

flowcast từng lấy việc khớp từng byte với một bản cài đặt khác làm chuẩn đúng,
và so cả từng bước trung gian của engine: bảng đã đọc, kích thước chữ, hàng và
cột của từng phần tử, kiểu dây, track của từng đoạn. Cách đó bắt được mọi sai
lệch, nhưng nó khóa luôn cấu trúc bên trong engine. Đổi cách chia hàng hay cách
xếp track là đỏ hàng loạt test, kể cả khi file đầu ra vẫn đúng hoặc tốt hơn.
Mỗi lần cố ý đổi hành vi lại phải ghi vào một danh sách ngoại lệ.

## Quyết định

Chuẩn đúng gồm hai phần, cả hai chỉ nhìn vào đầu ra:

1. Golden do chính flowcast sinh cho từng case: file .drawio, báo cáo phát hiện
   (issue của bảng, cảnh báo của engine, kết quả tự kiểm), sơ đồ ở các hướng
   khác, và bản ghi phiên làm việc của CLI. Golden chỉ được đổi bằng cờ -update,
   và người đổi phải xem ảnh và đọc diff.
2. Các bất biến chạy trên mọi case hợp lệ:
   - tự kiểm hình học không báo lỗi;
   - merge trên file vừa sinh, chưa sửa, không làm đổi file;
   - dây vào condition nối vào đỉnh, dây ra chỉ ở trái, phải, đáy;
   - các hướng khác là ảnh đổi trục hoặc lật của hướng từ trên xuống.

Test đơn vị của từng package vẫn được phép kiểm chi tiết bên trong package đó.
Không test nào ngoài package được so trạng thái trung gian của engine.

## Hệ quả

- Tái cấu trúc engine mà giữ nguyên đầu ra thì không test nào đỏ.
- Đổi hành vi có chủ đích là một lần chạy -update và một diff golden cần đọc.
  Không còn danh sách ngoại lệ.
- Golden không tự chứng minh là đúng. Bất biến và việc người duyệt xem ảnh là
  hai thứ giữ cho golden không trôi.
