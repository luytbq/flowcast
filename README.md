# flowcast

Biến một sơ đồ luồng viết bằng bảng hoặc bằng mermaid thành file draw.io có bố
cục tất định: mọi toạ độ do code tính, cùng một đầu vào luôn cho ra cùng một
file, và không ai phải sửa toạ độ trong XML bằng tay.

Lý do tồn tại: mermaid và chức năng import của draw.io đều tự xếp hình, và xếp
không đẹp. Thứ dự án này có mà chúng không có là một engine xếp hình và đi dây
tất định, đã qua nhiều vòng sửa theo sơ đồ thật.

## Làm được gì

- Đọc bảng luồng viết bằng markdown, csv hoặc xlsx, và flowchart mermaid.
- Vẽ bốn hướng: TD, BT, LR và RL. Sơ đồ có lane thì lane là băng dọc hoặc băng
  ngang tùy hướng; bảng không có lane nào thì vẽ thành flowchart.
- Sinh lại mà giữ chỉnh sửa tay trong draw.io: vị trí node, bề rộng lane, điểm
  gấp của dây, và các cell tự vẽ thêm.
- Tự kiểm hình học trước khi ghi, và kiểm render: xuất SVG bằng drawio CLI rồi
  so đường dây draw.io vẽ với toạ độ đã tính.
- Dùng được từ CLI, từ thư viện Go, và từ dịch vụ web flowcastd.

Bản Python trong `reference/` là bản gốc, nay giữ vai trò máy sinh đáp án: bản
Go cho ra đúng từng byte file .drawio, đúng từng dòng in ra và đúng mã thoát
như nó, trên toàn bộ bộ đối chiếu. Những chỗ cố ý khác nằm ở cuối mục Dùng.

| thư mục | nội dung |
|---|---|
| `cmd/flowcast/` | CLI |
| gốc, `model/`, `source/`, `schema/`, `validate/`, `text/`, `layout/`, `writer/` | core |
| `merge/` | sinh lại mà giữ chỉnh sửa tay; chỉ CLI dùng |
| `render/` | gọi drawio CLI: xuất PNG, SVG, kiểm render; ngoài core |
| `cmd/flowcastd/` | dịch vụ web |
| `internal/` | đọc ghi XML kiểu ElementTree, chuỗi kiểu Python, kiểm tĩnh |
| `reference/` | bản Python đầy đủ, đồng thời là máy sinh đáp án cho bản port |
| `conformance/` | bộ đối chiếu: bảng đầu vào và đầu ra chuẩn |
| `data/` | bảng độ rộng glyph, nhúng vào binary |
| `tools/` | công cụ sinh đáp án, đo độ phủ, thử đột biến |
| `docs/` | đặc tả định dạng, thiết kế core, ADR |
| `CONTEXT.md` | từ vựng dùng xuyên suốt code và tài liệu |

Đọc `CONTEXT.md` trước, rồi `docs/core-design.md`. Các quyết định đã chốt kèm lý
do nằm trong `docs/adr/`; đừng mở lại chúng mà chưa đọc.

## Dùng

```
go build -o flowcast ./cmd/flowcast
./flowcast check bang.md
./flowcast build so-do.mmd                   # flowchart mermaid, hoặc khối ```mermaid trong file .md
./flowcast build bang.md --direction LR      # TD, BT, LR hoặc RL
./flowcast build bang.md [-o ra.drawio] [--title "..."] [--mode merge|force] [--task-max-w 280 ...]
```

Dịch vụ web:

```
go build -o flowcastd ./cmd/flowcastd
./flowcastd -addr :8080
curl -F file=@bang.md 'localhost:8080/api/build?download=1' -o bang.drawio
```

POST /api/build và /api/check nhận multipart với trường file, cùng các trường
tùy chọn title, direction, sheet, delimiter, encoding và mọi tham số xếp hình,
rồi trả JSON gồm issue có mã máy và vị trí. GET /api/fields khai báo các tham
số đó kèm miền giá trị; trang upload dựng form từ chính khai báo này, nên thêm
một tham số trong core là nó có mặt ở cả CLI lẫn web.

Bảng có lỗi trả 422, cấu hình sai trả 400, vượt giới hạn trả 413, máy chủ bận
trả 503. Dịch vụ dùng hồ sơ giới hạn WebLimits trong `limits.go`, không merge
và không gọi drawio, nên nó không đọc ghi file và không chạy tiến trình ngoài.

Tên cờ, các dòng in ra và mã thoát giống hệt bản Python, xem `reference/README.md`.
Những chỗ khác có chủ đích:

- `--font` bị bỏ qua, vì flowcast đo chữ bằng bảng số đo nhúng sẵn.
- `--help` in hướng dẫn của bản Go, không phải chữ của argparse.
- `--layout-json` cùng cấu trúc khóa nhưng viết số theo cách của Go.
- `--encoding` nhận utf-8, utf-8-sig, cp1252 và latin-1 cùng các tên gọi khác của
  chúng. Python nhận hàng trăm bảng mã; tên lạ được xử lý như Python xử lý một
  tên nó không biết.
- Bản Go làm được ba thứ bản Python không có, nên không so được: đọc flowchart
  mermaid, vẽ hướng khác TD qua `--direction`, và dựng bảng không có lane nào
  thành flowchart. Riêng cái cuối đổi hành vi cũ: bản Python báo lỗi "bảng
  không có lane nào". Merge chỉ hỗ trợ hướng TD.
- Case mà bản Go cố ý khác bản tham chiếu được liệt kê trong
  `conformance/diverge.txt`, kèm lý do.
- CLI chặn bảng quá 50000 dòng hoặc file quá 64 MB, theo hồ sơ CLILimits.
  Bản Python không chặn.
- File .drawio cũ không đọc được khi merge: mã thoát và câu hướng dẫn giống,
  nhưng phần mô tả lỗi của bộ đọc XML và của zlib là của Go. File khai báo một
  bảng mã khác UTF-8 không đọc được; draw.io luôn ghi UTF-8.
- Python 3.14 dùng Unicode 16, còn Go 1.25 dùng Unicode 15. Ký tự mới có ở
  Unicode 16 có thể được đổi chữ thường hoặc chuẩn hóa khác nhau.

## Kiểm tra

```
sh tools/check.sh
DIRS="TD BT LR RL" sh tools/accept.sh   # cần drawio: dựng mọi case rồi so dây draw.io vẽ
```

Chạy test của bản tham chiếu, kiểm golden và bản ghi CLI có còn khớp bản tham
chiếu, đo độ phủ của bộ đối chiếu, rồi toàn bộ test Go. Có drawio trên máy thì
chạy luôn phép kiểm draw.io thật vẽ dây đúng toạ độ. Thêm:

```
python3 conformance/generate.py            # sinh lại golden sau khi cố ý đổi hành vi
go test ./conformance -run Flowchart -update   # sinh lại golden của flowchart và mermaid
sh tools/mutate.sh                         # cố tình làm sai từng chỗ, xem test có đỏ không
ONLY=layout/ sh tools/mutate.sh            # chỉ đột biến trên một module
go run ./tools/metrics conformance/mermaid/*.mmd   # đo chất lượng bố cục
python3 tools/fuzz_merge.py --go ./flowcast --n 500 # so merge với bản tham chiếu trên seed ngẫu nhiên
```

Sinh lại golden thì phải đọc diff trước khi commit. Một golden đổi im lặng là
một hành vi đã thay đổi mà không ai xem.

## Giấy phép

MIT, xem `LICENSE`. Áp cho mã nguồn trong repo, gồm cả bản tham chiếu Python.

Một chỗ cần người hiểu luật xem qua trước khi phân phối công khai:
`data/verdana.json` là bảng độ rộng glyph sinh từ font Verdana, một font thương
mại của Microsoft. Bảng số đo không phải file font, nhưng ranh giới đó không do
giấy phép này quyết định. Lý do phải nhúng bảng, và phương án dự phòng là đổi
sang một font tự do, nằm ở mục 11 của `docs/core-design.md`.

## Dùng bản tham chiếu

Xem `reference/README.md`. Định dạng bảng đầu vào ở `docs/flow-table-format.md`.
