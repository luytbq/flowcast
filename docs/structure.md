# Cấu trúc project

Tài liệu này dành cho người sắp sửa code flowcast lần đầu. Đọc xong, bạn biết
một thay đổi nằm ở package nào, dữ liệu đi qua các package theo thứ tự nào, và
package nào được phép làm gì.

Các từ như lane, kind, máng, kênh, track được định nghĩa trong
[CONTEXT.md](../CONTEXT.md). Nên đọc tài liệu đó trước.

## Một lần dựng sơ đồ đi qua những đâu?

Mọi đường vào (dòng lệnh, dịch vụ web, thư viện) đều gọi cùng một hàm Build ở
package gốc. Build nhận bytes và trả về văn bản .drawio kèm dữ liệu chẩn đoán.
Nó không đọc ghi file, không gọi tiến trình ngoài và không in gì ra.

```
bytes đầu vào
  --> source     đọc markdown, csv, xlsx hoặc mermaid, quy về một Table
  --> validate   kiểm Table theo lược đồ metadata và luật của sơ đồ
  --> layout     đo chữ, xếp chỗ, đi dây, tính toạ độ, đặt nhãn
  --> layout     tự kiểm hình học trên kết quả
  --> layout     đổi trục nếu hướng không phải từ trên xuống
  --> merge      (chỉ khi có file cũ) áp chỉnh sửa tay từ file cũ
  --> writer     sinh văn bản .drawio
```

Bảng có lỗi thì Build dừng sau bước validate và chỉ trả về danh sách lỗi. Hàm
Check ở package gốc chạy đúng hai bước đầu.

Kết quả của bước xếp hình là một kiểu dữ liệu thuần tên Result: toạ độ từng phần
tử, điểm gấp từng dây, vị trí từng nhãn. Tự kiểm, merge và writer đều chỉ đọc
Result, không đụng vào trạng thái bên trong engine.

## Package nào làm gì?

| Thư mục | Vai trò | Sửa ở đây khi |
|---|---|---|
| gốc repo | Build, Check, Options, Result, và các hồ sơ giới hạn tài nguyên cho dòng lệnh và web | Đổi giao diện công khai của thư viện, đổi giới hạn tài nguyên |
| model | Kiểu dữ liệu chung: Row, Table, Issue, Location, Error | Thêm trường cho Issue hay vị trí lỗi |
| num | Quy tắc làm tròn và in số, dùng chung cho mọi tầng | Đổi cách số được ghi vào file |
| source | Đọc từng định dạng đầu vào: markdown, csv, xlsx, mermaid; đoán định dạng và bảng mã | Thêm định dạng đầu vào, sửa lỗi đọc file |
| schema | Khai báo các key metadata hợp lệ cho từng loại dòng | Thêm key metadata, thêm loại dòng |
| validate | Kiểm một Table đã đọc: tham chiếu hợp lệ, id trùng, luật riêng của sơ đồ | Thêm luật kiểm bảng |
| text | Đo chữ và ngắt dòng bằng bảng độ rộng ký tự | Đổi cách ngắt dòng |
| data | Bảng độ rộng ký tự của font, nhúng vào binary | Đổi font đo chữ |
| layout | Engine xếp hình và đi dây, tự kiểm hình học, đổi trục | Mọi thay đổi về bố cục, xem bảng bên dưới |
| writer/drawio | Sinh file .drawio từ Result | Đổi hình vẽ, style, cấu trúc XML đầu ra |
| merge | Đọc file .drawio cũ, áp chỉnh sửa tay lên Result mới, lập báo cáo | Đổi những gì được giữ khi sinh lại |
| render | Gọi drawio CLI để xuất PNG và SVG, so dây draw.io vẽ với toạ độ đã tính | Đổi cách xuất ảnh hay cách kiểm render |
| cmd/flowcast | Dòng lệnh: đọc cờ, đọc ghi file, chọn chế độ ghi, in kết quả, mã thoát | Thêm cờ, đổi dòng in ra |
| cmd/flowcastd | Dịch vụ web và trang upload | Thêm API, đổi trang upload |
| internal/etree | Đọc ghi cây XML, dùng cho writer và merge | Hiếm khi |
| internal/unistr | Định nghĩa khoảng trắng và chữ thường dùng chung | Hiếm khi |
| internal/lint | Test tĩnh chạy trên chính mã nguồn, ví dụ bắt phép nhân số thực có thể bị gộp thành FMA | Thêm luật kiểm mã nguồn |
| conformance | Bộ case đầu vào, golden đầu ra, và các test so golden cùng các bất biến chạy trên mọi case | Thêm case, cập nhật golden, xem [testing.md](testing.md) |
| tools | Script kiểm tra toàn repo, nghiệm thu với drawio thật, thử đột biến, đo chất lượng bố cục, trích số đo từ file font | Thêm công cụ phát triển |
| docs | Đặc tả định dạng bảng, thiết kế, quyết định đã chốt | |

### Bên trong layout

Engine chia file theo pha. Mỗi pha đọc kết quả của pha trước. Cách các pha phối
hợp với nhau nằm ở [algorithm.md](algorithm.md).

| File | Nội dung |
|---|---|
| config.go | Tham số xếp hình, mặc định và miền giá trị. Dòng lệnh và web đều sinh danh sách tham số từ đây |
| model.go | Trạng thái của một lần xếp hình: phần tử, cạnh, lưới; cảnh báo của engine |
| shape.go | Khai báo hình học của từng loại phần tử: hình nguyên thủy, luật nối dây, điểm trên đường viền |
| size.go | Đo và ngắt dòng từng phần tử, trước mọi bước xếp chỗ |
| topo.go | Thứ tự topo của các node |
| branch.go | Chọn nhánh chính, xếp cột cho nhánh phụ, tìm cột hợp nhánh |
| place.go | Xếp chỗ: gán lane, hàng, cột cho mọi phần tử |
| route.go | Chọn kiểu đi dây cho từng cạnh và ghi các đoạn dây vào máng, kênh |
| tracks.go | Đặt cổng trên mặt node, xếp đoạn dây vào track |
| seg.go | Kiểu dữ liệu của đoạn dây và tài nguyên chứa dây |
| geometry.go | Đổi lưới thành điểm ảnh, dựng đường gấp khúc cho từng dây |
| labels.go | Chừa chỗ và đặt nhãn cho dây, cùng hàm Run chạy đủ các pha |
| axis.go | Đổi trục cho các hướng LR, RL, BT |
| result.go | Result, kiểu dữ liệu thuần trả ra ngoài |
| check.go | Tự kiểm hình học trên Result |

## Package nào được phép làm gì?

Các ranh giới sau giữ cho cùng một đường code phục vụ được cả dòng lệnh, web và
thư viện:

- Core gồm package gốc cùng model, num, source, schema, validate, text, data,
  layout, writer, merge và internal. Core không đọc ghi file và không gọi tiến
  trình ngoài. Dữ liệu vào là bytes, dữ liệu ra là văn bản.
- render gọi tiến trình drawio nên nằm ngoài core. Chỉ dòng lệnh và test dùng
  nó.
- merge thuộc core và cũng nhận bytes, nhưng chỉ dòng lệnh dùng, vì chỉ dòng
  lệnh có file cũ nằm cạnh bảng. Dịch vụ web không nhận file cũ.
- model và num nằm riêng khỏi package gốc vì mọi package đọc đầu vào đều cần
  chúng, còn package gốc lại cần các package đó. Package gốc phơi lại các kiểu
  của model bằng bí danh.
- Tự kiểm, merge và writer chỉ đọc Result. Nhờ vậy tự kiểm kiểm được cả hình học
  do người dùng sửa tay hoặc do test cố tình làm hỏng.
- Chỉ axis.go biết về hướng vẽ. Các pha khác luôn tính như sơ đồ đi từ trên
  xuống.

Chiều phụ thuộc giữa các package:

```
package gốc --> source, validate, layout, merge, writer/drawio, text, data, model
source      --> model
validate    --> schema, model
layout      --> text, schema, model, num
writer      --> layout, num
merge       --> layout, num
render      --> layout
```

Tất cả đều có thể dùng internal. Không package nào trong core được import render
hay cmd.
