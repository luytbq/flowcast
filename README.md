# flowcast

Biến một sơ đồ luồng viết bằng bảng hoặc bằng mermaid thành file draw.io có bố
cục tất định: mọi toạ độ do code tính, cùng một đầu vào luôn cho ra cùng một
file, và không ai phải sửa toạ độ trong XML bằng tay.

Lý do tồn tại: mermaid và chức năng import của draw.io đều tự xếp hình, và xếp
không đẹp. Thứ dự án này có mà chúng không có là một engine xếp hình và đi dây
tất định, đã qua nhiều vòng sửa theo sơ đồ thật.

## Trạng thái

Đang port từ Python sang Go. Bản Python trong `reference/` chạy được và là bản
đang dùng thật; bản Go chưa bắt đầu.

| thư mục | nội dung |
|---|---|
| `reference/` | bản Python đầy đủ, chạy được, đồng thời là máy sinh đáp án cho bản port |
| `conformance/` | bộ đối chiếu: bảng đầu vào và đầu ra chuẩn, bản Go phải khớp từng byte |
| `data/` | bảng độ rộng glyph, dùng chung cho cả hai bản |
| `tools/` | trích số đo font, đo độ phủ, chạy kiểm tra |
| `docs/` | đặc tả định dạng, thiết kế core, ADR |
| `CONTEXT.md` | từ vựng dùng xuyên suốt code và tài liệu |

Đọc `CONTEXT.md` trước, rồi `docs/core-design.md`. Các quyết định đã chốt kèm lý
do nằm trong `docs/adr/`; đừng mở lại chúng mà chưa đọc.

## Kiểm tra

```
sh tools/check.sh
```

Chạy test của bản tham chiếu và đo độ phủ của bộ đối chiếu. Thêm:

```
python3 conformance/generate.py --check    # golden còn khớp bản tham chiếu không
python3 conformance/generate.py            # sinh lại golden sau khi cố ý đổi hành vi
```

Sinh lại golden thì phải đọc diff trước khi commit. Một golden đổi im lặng là
một hành vi đã thay đổi mà không ai xem.

## Dùng bản tham chiếu

Xem `reference/README.md`. Định dạng bảng đầu vào ở `docs/flow-table-format.md`.
