drawio CLI giả cho các bản ghi CLI có --png và --verify. Bản ghi mang dòng
"# drawio: <tên>" thì cả bản tham chiếu lẫn bản Go chạy với PATH chỉ gồm thư
mục <tên> ở đây cùng /usr/bin và /bin, nên không ai thấy drawio thật.

| tên | hành vi |
|---|---|
| ok | ghi ảnh PNG giả, hoặc SVG không có data-cell-id |
| fail | in lỗi ra stderr, mã thoát 1 |
| silent | mã thoát 0 nhưng không ghi file nào |
| none | không có drawio |

Phần so SVG của kiểm render được chốt riêng, trên SVG thật, bằng
verify-vectors.json.
