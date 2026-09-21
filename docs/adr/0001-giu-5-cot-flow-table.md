# ADR-0001: Giữ đúng 5 cột cho Flow Table

- Trạng thái: đã chấp nhận
- Ngày: 2026-09-21

## Bối cảnh

Bảng Flow Table có 5 cột: id, type, parent, content, metadata. Trong đó cột
metadata chở cả những tham chiếu bắt buộc là from, to và attach, cạnh những key
tùy chọn như style và back.

Đếm trên ví dụ 22 dòng của đặc tả: id và type có mặt ở 100% số dòng, content
77%, parent 50%, from và to 36%, attach 9%.

Điều đó bộc lộ một bất đối xứng. Cột parent bỏ trống ở đúng những dòng edge,
trong khi from và to là tham chiếu bắt buộc lại bị giấu trong một chuỗi vốn
dành cho dữ liệu tùy chọn. Người dùng xlsx không lọc hay sắp xếp theo from và to
được, và việc kiểm tham chiếu treo phải đi qua lớp phân tích metadata.

Phép thử xóa bỏ cho thấy ranh giới cột so với metadata không phải vấn đề mô hình
hóa: bỏ cột nào cũng chỉ làm nội dung chuyển vào metadata, mô hình và luật kiểm
tra không đổi. Đây thuần túy là công thái học của người viết bảng.

Phương án đã cân nhắc và bị loại: lên 7 cột (id, type, parent, from, to,
content, metadata), kèm một reader tương thích nâng from và to từ metadata lên
cột để file cũ vẫn chạy.

## Quyết định

Giữ đúng 5 cột. Không thêm cột from và to.

Tính tường minh chuyển sang một lược đồ metadata do kind khai báo: theo từng
type, key nào bắt buộc, key nào là tham chiếu tới id khác, miền giá trị của từng
key.

## Hệ quả

- Số cột là một cam kết tương thích. Mọi bảng đã viết, mọi file .csv và .xlsx đã
  có, và prompt của subagent flowtable-drawio đều không phải đụng tới.
- Kiểm tham chiếu treo trở thành một vòng lặp trên các key đã đánh dấu is_ref
  trong lược đồ, thay vì viết tay từng trường hợp trong validate.
- Web trả được lỗi ở mức từng key trong ô metadata, nhờ lược đồ và nhờ Location
  có cấu trúc.
- Thêm kiểu sơ đồ mới không bao giờ cần thêm cột: kind mở rộng lược đồ, không mở
  rộng bảng.
- Đổi lại, người dùng xlsx vẫn không lọc được theo from và to. Đây là cái giá đã
  biết và đã chấp nhận.
