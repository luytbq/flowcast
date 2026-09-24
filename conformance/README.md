# Bộ case

Bộ case đầu vào, golden đầu ra và các test chạy trên toàn bộ bộ case. Cách dùng,
khi nào sinh lại golden và cách thêm case nằm ở
[docs/testing.md](../docs/testing.md).

| Đường dẫn | Nội dung |
|---|---|
| cases/ | Bảng markdown, csv, xlsx; mỗi file chốt một nhánh thuật toán. Số 9x là bảng cố ý sai |
| flowchart/ | Bảng không có lane |
| mermaid/ | Flowchart mermaid kiểu thực tế, cũng là bộ đo cho heuristic nhánh chính |
| golden/cases, golden/flowchart, golden/mermaid | File .drawio và báo cáo phát hiện của từng case |
| golden/direction/ | Sơ đồ ở các hướng BT, LR, RL |
| merge/ | Kịch bản merge: bảng đã sửa cùng file .drawio cũ đã sửa tay |
| cli/ | Bản ghi phiên làm việc của dòng lệnh, kể cả merge |
| drawio-fakes/ | drawio CLI giả cho bản ghi có --png và --verify |
| *-vectors.json | Dữ liệu hồi quy cố định cho test đơn vị của từng package |
| golden_test.go | So golden, và các bất biến chạy trên mọi case |
| direction_test.go | Golden và bất biến của các hướng vẽ |
