# flowcast

flowcast biến một sơ đồ luồng viết bằng bảng hoặc bằng mermaid thành file draw.io
đã xếp hình sẵn. Mọi toạ độ do tool tính, cùng một đầu vào luôn cho ra cùng một
file, và không ai phải kéo node hay sửa XML bằng tay.

Công cụ vẽ tự động thường xếp hình không đẹp: dây cắt qua node, nhãn đè nhau, các
nhánh rẽ lộn xộn. flowcast có engine xếp hình và đi dây riêng, tự kiểm hình học
trước khi ghi file, và kiểm được cả việc draw.io có vẽ đúng thứ nó đã tính hay
không.

## Làm được gì

- Đọc bảng luồng viết bằng markdown, csv hoặc xlsx, và flowchart mermaid.
- Vẽ bốn hướng: từ trên xuống (TD), từ dưới lên (BT), trái sang phải (LR), phải
  sang trái (RL). Sơ đồ có lane thì lane là băng dọc hoặc băng ngang tùy hướng.
  Bảng không khai báo lane nào thì vẽ thành flowchart.
- Sinh lại sơ đồ mà giữ chỉnh sửa tay trong draw.io: vị trí node, bề rộng lane,
  điểm gấp của dây, và các hình người dùng tự vẽ thêm.
- Tự kiểm hình học trước khi ghi: dây cắt node, hai dây chồng nhau, nhãn đè.
- Xuất PNG, và so đường dây draw.io vẽ ra với toạ độ đã tính.
- Dùng được qua dòng lệnh, qua dịch vụ web, và như một thư viện Go.

## Build

Cần Go 1.25 trở lên.

```
go build -o flowcast ./cmd/flowcast      # dòng lệnh
go build -o flowcastd ./cmd/flowcastd    # dịch vụ web
```

Xuất ảnh và kiểm render cần thêm drawio CLI (bản desktop của draw.io) trên máy.
Thiếu nó thì mọi chức năng khác vẫn chạy.

## Chạy từ dòng lệnh

```
./flowcast check bang.md                     # chỉ kiểm bảng, không vẽ
./flowcast build bang.md                     # ghi bang.drawio cạnh file đầu vào
./flowcast build so-do.mmd                   # flowchart mermaid, hoặc khối mermaid trong file .md
./flowcast build bang.md --direction LR
./flowcast build bang.md -o ra.drawio --png --verify
```

Định dạng bảng đầu vào nằm ở [docs/flow-table-format.md](docs/flow-table-format.md).

### Tham số chung

| Tham số | Ý nghĩa |
|---|---|
| -o, --output FILE | File .drawio ghi ra. Mặc định cùng tên với file đầu vào, đổi đuôi. |
| --title T | Tiêu đề sơ đồ. Mặc định lấy từ tiêu đề trong file nguồn. |
| --direction D | TD, BT, LR hoặc RL. Mặc định theo nguồn; nguồn không nói thì TD. |
| --mode merge hoặc force | Dùng khi file đích đã có. merge giữ chỉnh sửa tay, force sinh lại toàn bộ. Không truyền thì tool hỏi, hoặc dừng nếu không chạy trong terminal. |
| --no-backup | Không ghi bản sao .bak của file cũ trước khi ghi đè. |
| --png [FILE] | Xuất ảnh PNG bằng drawio CLI. Không ghi FILE thì ảnh nằm cạnh file .drawio. |
| --verify | Xuất SVG bằng drawio CLI rồi so từng đường dây với toạ độ đã tính. |
| --layout-json FILE | Ghi toạ độ đã tính ra JSON, để công cụ khác đọc. |
| --sheet S | Với xlsx: tên sheet chứa bảng. |
| --delimiter D | Với csv: dấu phân cách, khi không muốn tool tự đoán. |
| --encoding E | Với csv: bảng mã, khi không muốn tool tự đoán. |

### Tham số xếp hình

Đơn vị là điểm ảnh. Chạy lệnh flowcast --help để xem miền giá trị của từng tham số.

| Tham số | Mặc định | Ý nghĩa |
|---|---|---|
| --task-min-w | 120 | Bề rộng tối thiểu của hộp task |
| --task-max-w | 240 | Bề rộng tối đa của hộp task |
| --cond-wrap | 150 | Bề rộng ngắt dòng trong hình thoi |
| --term-wrap | 170 | Bề rộng ngắt dòng của start, end và external |
| --db-wrap | 130 | Bề rộng ngắt dòng của db |
| --text-wrap | 260 | Bề rộng ngắt dòng của ghi chú |
| --label-wrap | 180 | Bề rộng ngắt dòng của nhãn trên dây |
| --track-gap | 12 | Khoảng cách giữa hai dây chạy song song |
| --gutter-margin | 15 | Lề từ mép khe dọc giữa hai cột tới dây đầu tiên |
| --channel-margin | 12 | Lề từ mép khe ngang giữa hai hàng tới dây đầu tiên |
| --min-gutter | 24 | Khoảng trống tối thiểu giữa hai cột |
| --min-channel | 30 | Khoảng trống tối thiểu giữa hai hàng |
| --attach-gap | 40 | Khoảng cách từ db hoặc ghi chú tới node nó bám |
| --lane-header | 30 | Bề dày thanh tên lane |
| --pool-header | 30 | Bề dày thanh tiêu đề sơ đồ |
| --min-lane-w | 120 | Bề dày tối thiểu của một lane |
| --label-pad | 4 | Lề quanh chữ của nhãn trên dây |

### Mã thoát

| Mã | Khi nào |
|---|---|
| 0 | Thành công |
| 1 | Không đọc được file, bảng có lỗi, hoặc không ghi được file |
| 2 | Đã ghi file nhưng tự kiểm hình học báo lỗi |
| 3 | Đã ghi file nhưng xuất ảnh hỏng, hoặc draw.io vẽ lệch toạ độ đã tính |
| 4 | Không ghi: file đích đã có mà chưa chọn chế độ, hoặc không đọc được file cũ để merge |

Bảng quá 50000 dòng hoặc file quá 64 MB bị từ chối.

## Chạy dịch vụ web

```
./flowcastd -addr :8080
curl -F file=@bang.md 'localhost:8080/api/build?download=1' -o bang.drawio
```

Mở địa chỉ gốc của dịch vụ trong trình duyệt sẽ ra trang upload.

| Đường dẫn | Việc làm |
|---|---|
| POST /api/build | Dựng sơ đồ, trả JSON gồm file .drawio và các phát hiện. Thêm download=1 để tải thẳng file .drawio. |
| POST /api/check | Chỉ kiểm bảng. |
| GET /api/fields | Danh sách tham số xếp hình kèm mặc định và miền giá trị. |
| GET /healthz | Kiểm dịch vụ còn sống. |

Hai đường dẫn POST nhận multipart với trường file, cùng các trường tùy chọn title,
direction, sheet, delimiter, encoding và mọi tham số xếp hình (bỏ hai gạch đầu).
Trang upload và dòng lệnh dựng danh sách tham số từ cùng một khai báo, nên hai
bên luôn khớp nhau.

Cờ -concurrent đặt số lần dựng chạy cùng lúc, mặc định bằng số CPU. Yêu cầu phải
chờ quá 5 giây để có chỗ thì nhận mã 503.

Dịch vụ web chặn chặt hơn dòng lệnh: file tới 1 MB, bảng tới 1000 dòng và 1000
cạnh, mỗi lần dựng tới 10 giây. Vượt giới hạn thì nhận mã 413. Bảng có lỗi nhận
422, tham số sai nhận 400. Dịch vụ không merge, không đọc ghi file trên máy chủ
và không gọi drawio.

## Tài liệu

| Muốn biết | Đọc |
|---|---|
| Viết bảng đầu vào thế nào | [docs/flow-table-format.md](docs/flow-table-format.md) |
| Nghĩa của các từ dùng trong code và tài liệu | [CONTEXT.md](CONTEXT.md) |
| Code nằm ở đâu, dữ liệu đi qua những package nào | [docs/structure.md](docs/structure.md) |
| Engine tính toạ độ và đi dây thế nào | [docs/algorithm.md](docs/algorithm.md) |
| Chạy kiểm tra, sinh lại golden, thêm case | [docs/testing.md](docs/testing.md) |
| Thiết kế tổng thể của core | [docs/core-design.md](docs/core-design.md) |
| Sinh lại mà giữ chỉnh sửa tay hoạt động ra sao | [docs/merge-design.md](docs/merge-design.md) |
| Đọc csv và xlsx | [docs/input-formats-design.md](docs/input-formats-design.md) |
| Các quyết định đã chốt và lý do | [docs/adr/](docs/adr/) |

## Giấy phép

MIT, xem [LICENSE](LICENSE).

Tool đo chữ bằng một bảng độ rộng ký tự sinh từ font Verdana, một font thương mại
của Microsoft, và nhúng bảng này vào binary. Bảng số đo không phải file font,
nhưng cần người hiểu luật xem qua trước khi phân phối công khai. Lý do phải nhúng
bảng, và phương án dự phòng là đổi sang một font tự do, nằm ở mục 11 của
[docs/core-design.md](docs/core-design.md).
