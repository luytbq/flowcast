# flowcast

Biến một sơ đồ luồng viết bằng bảng hoặc bằng mermaid thành file draw.io có bố
cục tất định: mọi toạ độ do code tính, cùng một đầu vào luôn cho ra cùng một
file, và không ai phải sửa toạ độ trong XML bằng tay.

Lý do tồn tại: mermaid và chức năng import của draw.io đều tự xếp hình, và xếp
không đẹp. Thứ dự án này có mà chúng không có là một engine xếp hình và đi dây
tất định, đã qua nhiều vòng sửa theo sơ đồ thật.

## Trạng thái

Đang port từ Python sang Go. Bản Go đọc được bảng markdown, csv và xlsx, cho ra
đúng từng byte file .drawio như bản Python, in ra đúng từng dòng, và trả đúng mã
thoát. Chưa có: merge khi sinh lại, xuất ảnh và kiểm render. Những lệnh đó được
báo rõ là chưa hỗ trợ khi gọi tới.

| thư mục | nội dung |
|---|---|
| `cmd/flowcast/` | CLI |
| gốc, `model/`, `source/`, `schema/`, `validate/`, `text/`, `layout/`, `writer/` | core |
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
./flowcast build bang.md [-o ra.drawio] [--title "..."] [--mode force] [--task-max-w 280 ...]
```

Tên cờ, các dòng in ra và mã thoát giống hệt bản Python, xem `reference/README.md`.
Những chỗ khác có chủ đích:

- `--font` bị bỏ qua, vì flowcast đo chữ bằng bảng số đo nhúng sẵn.
- `--layout-json` cùng cấu trúc khóa nhưng viết số theo cách của Go.
- `--encoding` nhận utf-8, utf-8-sig, cp1252 và latin-1 cùng các tên gọi khác của
  chúng. Python nhận hàng trăm bảng mã; tên lạ được xử lý như Python xử lý một
  tên nó không biết.
- Python 3.14 dùng Unicode 16, còn Go 1.25 dùng Unicode 15. Ký tự mới có ở
  Unicode 16 có thể được đổi chữ thường hoặc chuẩn hóa khác nhau.

## Kiểm tra

```
sh tools/check.sh
```

Chạy test của bản tham chiếu, đo độ phủ của bộ đối chiếu, rồi toàn bộ test Go.
Thêm:

```
python3 conformance/generate.py --check    # golden còn khớp bản tham chiếu không
python3 conformance/generate.py            # sinh lại golden sau khi cố ý đổi hành vi
sh tools/mutate.sh                         # cố tình làm sai từng chỗ, xem test có đỏ không
ONLY=layout/ sh tools/mutate.sh            # chỉ đột biến trên một module
```

Sinh lại golden thì phải đọc diff trước khi commit. Một golden đổi im lặng là
một hành vi đã thay đổi mà không ai xem.

## Dùng bản tham chiếu

Xem `reference/README.md`. Định dạng bảng đầu vào ở `docs/flow-table-format.md`.
