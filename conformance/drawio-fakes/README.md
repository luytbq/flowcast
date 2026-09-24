drawio CLI giả cho các bản ghi CLI có --png và --verify. Bản ghi mang dòng
"# drawio: <tên>" thì test chạy CLI với PATH chỉ gồm thư mục <tên> ở đây, nên
drawio thật trên máy không lọt vào.

| tên | hành vi |
|---|---|
| ok | ghi dòng lệnh nhận được vào file PNG, hoặc SVG không có data-cell-id |
| stdout-fail | in lỗi ra stdout, mã thoát 1 |
| fail | in lỗi ra stderr, mã thoát 1 |
| silent | mã thoát 0 nhưng không ghi file nào |
| none | không có drawio |

Phần so SVG của kiểm render được chốt riêng, trên SVG thật do drawio xuất, bằng
verify-vectors.json.
