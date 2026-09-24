# ADR-0007: bản Go được phép khác bản tham chiếu, nhưng phải ghi lại từng chỗ

- Trạng thái: bị thay thế bởi ADR-0008
- Ngày: 2026-09-23

## Bối cảnh

Suốt lộ trình port, luật là: bản Go cho ra đúng từng byte, đúng từng dòng in ra
và đúng mã thoát như bản Python, trên toàn bộ bộ đối chiếu. Luật đó là thứ giữ
cho bản port không lặng lẽ sai.

Nhưng sản phẩm phải đi xa hơn bản gốc: đọc mermaid, vẽ bốn hướng, dựng bảng
không có lane thành flowchart. Có những chỗ hành vi cũ là sai theo nghĩa sản
phẩm, ví dụ bảng không có lane bị báo lỗi. Giữ luật tuyệt đối nghĩa là đóng băng
cả những chỗ đó.

Ngược lại, bỏ luật thì mất cổng chặn: một bản port lệch vì lỗi và một bản port
lệch vì cố ý trông giống hệt nhau trong diff.

## Quyết định

Bản Go được phép khác, với ba điều kiện:

1. Chỗ khác phải nằm trong `conformance/diverge.txt`, mỗi dòng một case hoặc
   một bản ghi CLI, kèm lý do. Test Go bỏ qua đúng những dòng đó khi so với đáp
   án của Python, và không bỏ qua gì khác.
2. Hành vi mới phải có cổng chặn riêng: case mới, golden do bản Go sinh, và
   golden đó chỉ được đổi khi có người xem ảnh và đọc diff.
3. Chỗ khác phải có trong README, ở danh sách những chỗ cố ý khác, viết cho
   người dùng chứ không cho người port.

## Hệ quả

- Diff của bộ đối chiếu vẫn có nghĩa: một golden đổi mà không có dòng tương ứng
  trong diverge.txt là một lỗi.
- Bản Python vẫn là đáp án cho mọi thứ còn lại, kể cả những thứ thêm sau như
  merge, nên nó vẫn được chạy trong `tools/check.sh` và trong CI.
- Cái giá: mỗi lần cố ý đổi hành vi là ba việc chứ không phải một. Đó là ý đồ,
  vì đổi hành vi phải đắt hơn sửa lỗi.
- Khi bản Python thôi được dùng nữa, `diverge.txt` là danh sách đầy đủ những
  chỗ hai bản đã tách nhau, tức là tài liệu bàn giao chứ không phải rác.
