# ADR-0002: Mermaid chỉ ở chiều vào, không ở chiều ra

- Trạng thái: đã chấp nhận
- Ngày: 2026-09-21

## Bối cảnh

Các đích xuất ra chia thành hai họ. Họ thứ nhất tiêu thụ toạ độ: draw.io,
excalidraw, SVG, PDF. Họ thứ hai tự làm layout của riêng chúng và chỉ cần danh
sách node và cạnh: mermaid, graphviz, plantuml.

Nếu nhận họ thứ hai vào phạm vi, writer seam phải tồn tại ở hai chỗ khác nhau,
một ở LayoutResult và một ở Table, vì họ thứ hai không dùng tới LayoutResult.

Quan trọng hơn: xuất ra mermaid là vứt bỏ đúng 840 dòng xếp hình vốn là toàn bộ
lý do tồn tại của công cụ.

Cùng lúc đó, mermaid ở chiều vào lại chính là luận đề sản phẩm. Người dùng có
sẵn mermaid, và cả mermaid lẫn chức năng import của draw.io đều cho bố cục không
đạt. Thứ công cụ này có mà họ không có là bố cục tất định.

## Quyết định

Writer seam chỉ nằm ở LayoutResult, tức chỉ phục vụ họ tiêu thụ toạ độ.

Mermaid là một source adapter, ngang hàng với markdown, csv và xlsx, quy về cùng
một Table.

Muốn xuất mermaid thì viết một script phụ đọc Table, nằm ngoài core.

## Hệ quả

- Core có đúng một lý do tồn tại, và một chỗ duy nhất để cắm writer.
- Chuyển đổi là một chiều. Không có cam kết khứ hồi mermaid ra mermaid.
- Cú pháp mermaid không ánh xạ được vào bảng 5 cột sẽ mất. Chúng phải là Issue
  có mã và có số dòng, không được im lặng bỏ qua.
- Nếu về sau thật sự cần xuất ra một đích tự làm layout, ADR này phải mở lại,
  vì nó dựng seam ở chỗ không phục vụ được họ đó.
