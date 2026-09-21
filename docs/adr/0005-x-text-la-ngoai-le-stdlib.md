# ADR-0005: golang.org/x/text là ngoại lệ của luật core chỉ dùng stdlib

- Trạng thái: đã chấp nhận
- Ngày: 2026-09-21

## Bối cảnh

Core chỉ dùng thư viện chuẩn. Luật đó được chọn để binary nhỏ, build cho WASM
về sau nhẹ, và không phải theo dõi chuỗi cung ứng.

Nhưng việc đọc bảng cần chuẩn hóa Unicode về dạng NFC, và thư viện chuẩn của Go
không có.

Chuẩn hóa không phải để cho đẹp. Cùng một chữ nhìn thấy trên màn hình lưu được
theo nhiều dãy mã khác nhau: "Xử lý" ở dạng dựng sẵn là 5 ký tự Unicode, ở dạng
tách dấu là 8. Tool đo chữ bằng cách cộng bề rộng từng ký tự, nên hai dạng cho
ra 30.75px và 42.43px, lệch 38%. Bề rộng chữ quyết định bề rộng hộp, bề rộng hộp
quyết định bề rộng lane, và lane quyết định toạ độ mọi thứ.

Tiếng Việt dính nặng hơn hầu hết ngôn ngữ vì nhiều chữ chồng hai dấu: một dấu
tạo chữ cái như ư, ơ, ă, â, ê, ô, đ, cộng một dấu thanh.

Dạng tách dấu không phải chuyện giả định. macOS lưu tên file ở dạng đó, vài
trình soạn thảo và vài bộ gõ tiếng Việt sinh ra nó, và chép dán giữa một số ứng
dụng cũng vậy. Người dùng không biết file của mình ở dạng nào.

Hai phương án còn lại đã cân nhắc:

- **Tự sinh bảng hợp thành từ Python rồi commit**, đúng cách đã làm với bảng số
  đo font. Giữ được luật, nhưng phải tự viết cả phần sắp xếp thứ tự dấu kết
  hợp, tức là viết lại một phần thuật toán chuẩn hóa Unicode, và bảng chỉ đúng
  trong phạm vi ký tự đã sinh.
- **Không chuẩn hóa, chỉ kiểm và báo lỗi.** Làm được bằng stdlib, nhưng file
  chép từ macOS bị từ chối thay vì chạy được, và hành vi lệch hẳn bản tham
  chiếu.

## Quyết định

Core dùng `golang.org/x/text/unicode/norm`. Đây là ngoại lệ duy nhất được cấp
cho luật chỉ stdlib, và nó chỉ áp cho việc chuẩn hóa Unicode.

Phiên bản ghim ở v0.30.0, vì các bản mới hơn yêu cầu Go 1.26 còn mốc tương
thích của dự án là Go 1.25.

## Hệ quả

- Binary tăng khoảng 219 KB, từ 2331 lên 2551 KB. Chấp nhận được với CLI và
  server; cần đo lại nếu WASM vào phạm vi.
- Một phụ thuộc duy nhất, do chính đội Go duy trì.
- **Ngoại lệ này không tạo tiền lệ.** Thêm phụ thuộc khác vào core vẫn phải mở
  ADR riêng và phải chứng minh rằng stdlib không có đường nào, chứ không phải
  rằng thư viện ngoài tiện hơn. Lý do chấp nhận ở đây là viết lại chuẩn hóa
  Unicode cho đúng là bãi mìn, không phải là tiết kiệm công.
- `conformance/cases/30-input-nfd.md` giữ nhánh này sống. Trước case đó mọi
  bảng trong bộ đối chiếu đều đã ở dạng dựng sẵn, nên một bản port quên chuẩn
  hóa vẫn qua được mọi cổng.
