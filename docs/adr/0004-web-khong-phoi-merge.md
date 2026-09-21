# ADR-0004: Dịch vụ web không phơi merge

- Trạng thái: đã chấp nhận
- Ngày: 2026-09-21

## Bối cảnh

merge sinh lại sơ đồ mà giữ các chỉnh sửa tay: vị trí phần tử, bề rộng lane,
waypoint, và cả những cell người dùng tự vẽ. Nó tồn tại để phục vụ vòng lặp sinh
file, kéo tay trong draw.io, rồi sinh lại.

Vòng lặp đó giả định người dùng giữ file .drawio của chính họ qua nhiều lần
chạy. Trên web, mỗi request là một lần rời rạc và không có file nào được giữ.

Phơi merge ra web nghĩa là thêm một ô upload thứ hai, một nhánh API, và một phần
giao diện chỉ để hiển thị danh sách chỗ cần sửa tay trong MergeReport.

## Quyết định

Dịch vụ web chỉ phơi đường sinh mới.

Core vẫn có merge đầy đủ, và tham số overrides vẫn nằm trong chữ ký build ngay
từ đầu. CLI dùng nó. Web chỉ không phơi ra.

## Hệ quả

- API web có đúng một hình dạng: một file lên, một file xuống.
- Không tốn gì về thiết kế để mở sau này. Chữ ký build đã sẵn sàng, chỉ thiếu
  phần giao diện.
- merge vẫn phải được làm cho đúng trong core, gồm cả việc chạy lại check sau
  khi áp Overrides. Quyết định này giới hạn phạm vi giao diện, không giới hạn
  chất lượng của merge.
- Người dùng web không giữ được chỉnh sửa tay giữa hai lần dựng. Muốn vòng lặp
  đó thì dùng CLI.
