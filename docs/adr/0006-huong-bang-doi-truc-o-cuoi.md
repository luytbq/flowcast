# ADR-0006: hướng khác TD làm bằng đổi trục ở cuối, không viết lại engine theo trục

- Trạng thái: đã chấp nhận
- Ngày: 2026-09-23

## Bối cảnh

Engine chỉ biết một hướng: luồng đi từ trên xuống, nhánh rẽ sang hai bên. Mục 7
của `docs/core-design.md` đề xuất cách làm hướng khác: đặt tên hai trục là flow
và cross, viết một module `Axis` với `ToXY(flow, cross)`, rồi dồn mọi chỗ đang
nói bằng x và y về module đó. Place và route vốn đã trung lập về trục; pixel chỉ
xuất hiện ở pha hình học, ở phần tính toạ độ track, ở phần giải tham chiếu
đường đi, và ở writer.

Khi bắt tay làm thì có một ràng buộc mạnh hơn: sơ đồ TD phải tiếp tục khớp bản
tham chiếu Python tới từng bit. Pha hình học so sánh số thực để ra quyết định,
nên chỉ cần đổi thứ tự một phép cộng là một nhãn nhảy chỗ và một golden đổi.
Viết lại bốn pha theo trục nghĩa là đụng vào đúng những dòng đó.

## Quyết định

Engine vẫn xếp trong một không gian TD ảo. `layout/axis.go` làm hai việc:

- Trước khi xếp, với LR và RL, hoán đổi bề rộng và bề cao của mọi phần tử và
  mọi nhãn. Hàng của không gian ảo khi đó có đúng độ dày của cột thật.
- Sau khi xếp và sau khi tự kiểm, đổi trục và lật kết quả: điểm, hộp, điểm neo
  và độ lệch nhãn.

Chữ vẫn được đo và ngắt dòng theo chiều thật, vì chữ không xoay theo sơ đồ.

## Hệ quả

- Place, route, geometry và labels không đổi một dòng. Toàn bộ golden của sơ đồ
  TD khớp nguyên, và bộ đối chiếu vẫn là một phép so từng byte với bản tham
  chiếu.
- Tự kiểm chạy trên kết quả ảo. Đổi trục và lật là phép biến đổi bảo toàn khoảng
  cách và quan hệ chồng lấn, nên kết luận của tự kiểm không đổi theo hướng.
- BT là ảnh lật của TD và RL là ảnh lật của LR, kiểm được bằng test: cùng số
  liệu, cùng kiểu đi dây cho từng cạnh.
- LR không phải ảnh chuyển trục của TD, và không nên mong nó là: kích thước node
  đổi vai trò nên cách xếp khác thật.
- Cái mất: hai chỗ vẫn nói bằng ngôn ngữ TD. Bề dày tối thiểu của lane lấy theo
  bề rộng chữ của tên lane, nên trong sơ đồ đi ngang, tên lane dài làm băng dày
  quá mức. Merge cũng chưa chạy được với hướng khác TD, vì nó đọc file cũ trong
  hệ toạ độ thật. Cả hai được ghi ở mục 15 của core-design.
- Nếu sau này có kind thứ hai xếp hình theo cách khác, hoặc cần hướng chéo, thì
  module Axis như thiết kế gốc mới đáng làm. Lúc đó bản port đã không còn phải
  khớp bản tham chiếu từng bit nữa, nên rào cản chính biến mất.
