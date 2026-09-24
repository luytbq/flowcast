# Kiểm tra

Tài liệu này dành cho người chuẩn bị commit một thay đổi vào flowcast. Đọc xong,
bạn biết chạy kiểm tra nào, khi nào phải sinh lại golden, thêm case thế nào, và
làm sao biết bộ test có thật sự bắt được lỗi hay không.

Nên đọc [structure.md](structure.md) trước để biết các package và thư mục.

## Cái gì quyết định một thay đổi là đúng?

Hai thứ, cả hai chỉ nhìn vào đầu ra của flowcast (ADR-0008):

- **Golden.** Mỗi case trong bộ case có một file .drawio và một file báo cáo do
  chính flowcast sinh. File báo cáo liệt kê issue của bảng, cảnh báo của engine
  và kết quả tự kiểm, mỗi thứ kèm mã. Ngoài ra còn golden cho các hướng vẽ khác,
  và bản ghi phiên làm việc của dòng lệnh: lệnh chạy, từng dòng in ra, mã thoát,
  và dấu vân tay của từng file sinh ra.
- **Bất biến.** Các điều mọi sơ đồ phải thỏa, kiểm trên mọi case hợp lệ:
  - tự kiểm hình học không báo lỗi;
  - merge trên file vừa sinh, chưa sửa, không làm đổi file;
  - dây vào condition nối vào đỉnh, dây ra chỉ ở trái, phải, đáy;
  - các hướng khác là ảnh đổi trục hoặc lật của hướng từ trên xuống.

Test đơn vị trong từng package được phép kiểm chi tiết bên trong package đó.
Không test nào ngoài package được so trạng thái trung gian của engine, nên tái
cấu trúc engine mà giữ nguyên đầu ra thì không test nào đỏ.

## Chạy kiểm tra nào trước khi commit?

```
sh tools/check.sh
```

Script kiểm gofmt, chạy go vet, rồi chạy toàn bộ go test. Máy có drawio CLI thì
nó chạy thêm phép kiểm draw.io thật vẽ dây đúng toạ độ trên vài case.

Trước một thay đổi lớn về bố cục, chạy thêm nghiệm thu đầu cuối. Nó dựng mọi case
ở mọi hướng rồi bắt drawio thật vẽ và so từng đường dây, mất vài phút:

```
DIRS="TD BT LR RL" sh tools/accept.sh
```

## Khi nào phải sinh lại golden?

Khi thay đổi cố ý đổi đầu ra: bố cục, style, thông điệp, dòng in ra của dòng
lệnh. Sửa code trước, rồi sinh lại:

```
go test ./conformance -update      # golden .drawio, báo cáo, golden các hướng
go test ./cmd/flowcast -update     # bản ghi dòng lệnh
```

Sau đó đọc diff của golden, và mở ảnh của những sơ đồ bị đổi, ví dụ bằng
flowcast build với cờ --png. Chỉ commit khi mọi chỗ đổi đều là chỗ bạn định đổi.
Một golden đổi mà không ai xem là một hành vi đã thay đổi mà không ai biết.

Chạy go test ./conformance -update khi thay đổi không định đổi đầu ra mà test
vẫn đỏ là sai: thứ cần sửa là code, không phải golden.

## Thêm case thế nào?

1. Viết bảng vào conformance/cases, đặt tên theo nhánh thuật toán nó chốt, không
   theo nội dung nghiệp vụ. 25-tracks-parallel.md, không phải
   25-luong-dat-hang.md. Bảng cố ý sai mang số bắt đầu bằng 9 và chỉ có golden
   báo cáo, không có .drawio.
2. Giữ bảng nhỏ nhất có thể mà vẫn ép được nhánh cần chốt.
3. Sinh golden, xem ảnh, đọc báo cáo.
4. Chạy thử đột biến (mục dưới) để xác nhận case mới giết được đột biến nó nhắm
   tới. Một case được thêm vì nghĩ rằng nó chạm tới nhánh nào đó, mà không giết
   được đột biến nào, là một case vô ích.

Case do máy tìm (số 60 trở lên, nội dung kiểu "A-1 x", "d0") giữ nguyên nội
dung: đổi chữ là đổi bề rộng hộp, và có thể làm case thôi giết được đột biến.

Sơ đồ không có lane đặt ở conformance/flowchart, sơ đồ mermaid ở
conformance/mermaid. Kịch bản merge (bảng cùng file .drawio cũ đã sửa tay) đặt
ở conformance/merge và được chạy qua bản ghi dòng lệnh.

## Làm sao biết bộ test bắt được lỗi?

```
sh tools/mutate.sh                 # mọi đột biến
ONLY=layout/ sh tools/mutate.sh    # chỉ đột biến trong một thư mục
```

Script cố tình làm sai từng chỗ trong code, chạy go test, rồi khôi phục. Một đột
biến mà test vẫn xanh là một chỗ bộ test không canh. Cây làm việc phải sạch trước
khi chạy, vì script khôi phục bằng git checkout.

Với mỗi đột biến bỏ lọt, chỉ có hai cách xử lý:

- Thêm case làm lỗi đó lộ ra ở đầu ra.
- Chứng minh đột biến không bao giờ đổi đầu ra, rồi ghi lý do vào danh sách đột
  biến tương đương ở đầu script, và bỏ nó khỏi danh sách chạy.

Thêm một đột biến mới cho mỗi luật mới đưa vào engine. Chuỗi đích phải xuất hiện
đúng một lần trong file, và nên tránh khoảng trắng canh lề vì gofmt canh lại cột.

Vài kinh nghiệm khi thiết kế case cho pha xếp chỗ:

- Việc một mặt đã có mũi tên ngang chỉ ảnh hưởng tới cột của nhánh phụ khi mũi
  tên ngang được đặt trước nhánh phụ, tức đích khác lane phải đứng trước trong
  bảng.
- Việc chọn nguồn nào làm chuẩn cho node hợp nhánh chỉ lộ ra khi các nguồn nằm ở
  cột khác nhau và không có node rẽ chung.
- Muốn ép nhánh phụ dạt qua nhiều ô bị chặn, dùng mũi tên ngang sang lane khác:
  nó chặn mọi cột dương của lane nguồn trên cùng hàng.

## Các file vector là gì?

Thư mục conformance có vài file JSON chứa đầu vào và kết quả mong đợi cho các
phần cần kiểm trên rất nhiều giá trị: làm tròn số, đọc csv, giải mã bảng mã, đọc
ghi XML, đọc số và base64 trong file .drawio cũ, đo và ngắt dòng chữ, tự kiểm
trên hình học hỏng, và kiểm render trên SVG thật do drawio xuất. Test đơn vị của
package tương ứng đọc chúng.

Chúng là dữ liệu hồi quy cố định. Khi cố ý đổi một hành vi mà vector chốt, sửa
kết quả mong đợi trong file tương ứng và ghi lý do vào commit.

## Đo chất lượng bố cục

```
go run ./tools/metrics conformance/mermaid/*.mmd conformance/flowchart/*.md
```

In cho từng sơ đồ: số dây mỗi kiểu, số điểm gấp, số chỗ hai dây cắt nhau, tổng
chiều dài dây, kích thước, số phát hiện tự kiểm, và tỉ lệ cạnh của đường chính
được vẽ thẳng đứng. Dùng nó để đánh giá một heuristic bằng số liệu thay vì cảm
nhận: chạy trước và sau thay đổi rồi so.

## Sinh lại bảng số đo font

```
go run ./tools/extractmetrics /đường/dẫn/Verdana.ttf data/verdana.json
```

Chỉ cần khi đổi font. Đổi bảng là đổi kích thước mọi hộp, nên sau đó phải sinh
lại toàn bộ golden và xem ảnh.
