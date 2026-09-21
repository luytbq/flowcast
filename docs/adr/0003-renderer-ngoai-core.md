# ADR-0003: Renderer nằm ngoài core

- Trạng thái: đã chấp nhận
- Ngày: 2026-09-21

## Bối cảnh

Hàm export hôm nay gọi tiến trình ngoài: tìm binary drawio, dựng dòng lệnh kèm
cờ no-sandbox, chạy với timeout 90 giây và thử lại một lần. verify_svg cũng chỉ
chạy được khi có binary đó.

Chạy một tiến trình Electron bên trong một web request là sai về nhiều mặt cùng
lúc: thời gian, bộ nhớ, và diện tích tấn công.

Core phải phục vụ được cả CLI lẫn HTTP mà không bên nào phải viết lại chính sách
của bên kia. Luật khiến điều đó khả thi là core không gọi tiến trình ngoài,
không chạm filesystem, không đọc biến môi trường, không in ra stdout.

## Quyết định

Core trả về văn bản đích. Việc biến văn bản đó thành ảnh nằm trong package
flowtable_render, ngoài core.

CLI gọi flowtable_render trực tiếp. Dịch vụ web chạy nó trong worker tách khỏi
request, hoặc không chạy.

verify là công cụ kiểm thử, không phải một bước của build.

## Hệ quả

- Toàn bộ test của core chạy không cần drawio CLI và không cần file tạm.
- Dịch vụ web triển khai được mà không cần Electron trong image của tiến trình
  phục vụ request.
- Xem trước bằng ảnh trên web trở thành việc bất đồng bộ, không phải một phần
  của lời gọi build. Đây là cái giá đã biết.
- Seam renderer hiện chỉ có một adapter, nên nó là seam giả định. Không xây
  interface renderer trừu tượng cho tới khi có adapter thứ hai thật sự.
