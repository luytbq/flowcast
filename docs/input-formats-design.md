# Thiết kế: nhận thêm input csv và xlsx

Trạng thái: đã implement (read_markdown, read_csv, read_xlsx, table_from_grid trong flowtable2drawio.py) và test ở tests/test_inputs.py. Cách dùng: README.md, mục Định dạng đầu vào.

## Bài toán

Flow Table hiện chỉ viết được trong file markdown. Người viết bảng có thể quen Excel hơn, hoặc nhận bảng từ người khác dưới dạng .csv, .xlsx. Tool cần dựng sơ đồ thẳng từ các file đó.

## Quyết định đã chốt

### 1. csv và xlsx là nguồn ngang hàng với md

- check và build nhận thẳng .md, .csv, .xlsx; định dạng suy từ đuôi file.
- Không sinh file trung gian, không có lệnh convert, để không bao giờ có hai bản lệch nhau.
- File đích vẫn chỉ đổi đuôi: flow.xlsx --> flow.drawio.
- Chỉ làm chiều đọc. Không ghi ra .csv hay .xlsx.

### 2. Tìm bảng và lấy tiêu đề

- Hàng header là hàng đầu tiên, quét từ trên xuống, có đủ 5 ô id, type, parent, content, metadata. Không phân biệt hoa thường, bỏ khoảng trắng thừa.
- Mọi hàng phía trên header bị bỏ qua, nên đầu file được phép có tiêu đề, ghi chú, tên người viết.
- Bảng kết thúc ở hàng trống hoàn toàn đầu tiên sau header; phía dưới vẫn ghi chú được.
- xlsx: lấy sheet đầu tiên có hàng header. Cờ --sheet chọn sheet theo tên.
- Tiêu đề sơ đồ: ưu tiên --title, rồi tới ô không rỗng đầu tiên nằm phía trên hàng header, cuối cùng là tên file bỏ đuôi.

### 3. Quy ước nội dung ô trong csv/xlsx

- Xuống dòng: nhận cả xuống dòng thật trong ô lẫn thẻ br, để bảng dán từ md sang vẫn chạy.
- Gạch đứng là ký tự bình thường.
- Không xử lý escape kiểu markdown: gạch chéo ngược là ký tự thật. Thấy chuỗi gạch chéo ngược kèm gạch đứng thì cảnh báo, vì gần như chắc chắn là dán từ md sang mà chưa dọn.
- Ô kiểu số: đọc theo giá trị hiển thị, cắt phần .0 thừa. Riêng ô id, from, to, attach mà là kiểu số thì cảnh báo, vì id dạng 4.10 bị Excel biến thành 4.1 và làm hỏng tham chiếu. Cách tránh: định dạng cột đó là Text.
- Cắt khoảng trắng đầu và cuối mọi ô.

### 4. csv: tự đoán dấu phân cách và bảng mã, có cờ ghi đè

- Dấu phân cách: thử dấu phẩy, chấm phẩy, tab; chọn cái nào cho ra hàng header đủ 5 ô. Không cái nào ra thì báo lỗi kèm gợi ý --delimiter.
- Bảng mã: thử utf-8 có BOM, utf-8, rồi cp1252. Phải lùi về cp1252 thì in cảnh báo, vì thường là file lưu sai và chữ tiếng Việt có thể đã hỏng từ trước.
- Cờ --delimiter và --encoding để ghi đè.
- Dấu nháy theo chuẩn csv (nháy kép bọc ô, nháy đôi là một nháy), đúng như Excel xuất, nên xuống dòng trong ô đọc được.

### 5. Cách đọc file xlsx

- Ô công thức: lấy giá trị đã tính lưu trong file, không tự tính. Không có giá trị lưu sẵn thì coi như ô rỗng và cảnh báo kèm địa chỉ ô.
- Cột thừa bên phải bảng: bỏ qua, để giữ được các cột riêng như ghi chú, người phụ trách.
- Ô gộp: giá trị thuộc ô trên cùng bên trái, các ô còn lại rỗng. Ô gộp nằm trong vùng bảng thì cảnh báo, vì dễ làm bảng kết thúc sớm hoặc mất nội dung.
- Hàng và cột ẩn: vẫn đọc bình thường; có hàng ẩn trong vùng bảng thì cảnh báo. Bảng là nguồn duy nhất của sơ đồ, cái gì còn trong bảng thì phải lên hình.
- In tên sheet đã dùng trong báo cáo.

## Ghi chú cho lúc implement

- Đọc xlsx bằng zipfile và ElementTree của thư viện chuẩn: xl/workbook.xml cho danh sách sheet, xl/sharedStrings.xml cho chuỗi dùng chung, xl/worksheets/sheetN.xml cho ô. Cần xử lý ô kiểu s (shared string), inlineStr, n, b và ô công thức có thẻ v.
- Thông báo lỗi và cảnh báo phải chỉ đúng chỗ trong file nguồn: md dùng số dòng, csv dùng số dòng, xlsx dùng tên sheet kèm địa chỉ ô (vd Sheet1!C12).
