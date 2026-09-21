# flowtable2drawio

Chuyển một Flow Table (định dạng mô tả ở flow-table-format.md cạnh file này) thành file draw.io kiểu activity-swimlane: mỗi lane là một cột dọc, luồng đi từ trên xuống. Bảng viết trong .md, .csv hoặc .xlsx đều được.

Tool chỉ dùng thư viện chuẩn của Python 3.10+. Mọi toạ độ đều do code tính và kết quả tất định, nên cùng một bảng luôn ra cùng một file. AI hay người dùng không cần, và không nên, sửa toạ độ trong XML bằng tay. Subagent flowtable-drawio (~/.claude/agents/flowtable-drawio.md) gọi tool này.

## Cách chạy

```
python3 reference/flowtable2drawio.py check  <file> [--sheet TÊN] [--delimiter ,] [--encoding utf-8]
python3 reference/flowtable2drawio.py build  <file> [-o out.drawio] [--mode merge|force] [--png [out.png]] [--verify] [--layout-json out.json] [--title "..."] [--no-backup] [--sheet TÊN] [--delimiter ,] [--encoding utf-8]
```

- check: kiểm tra bảng theo mục "Kiểm tra một bảng có hợp lệ không" của định dạng.
- build: chạy check trước, bảng có lỗi thì dừng. Nếu không có lỗi, tool dựng layout, ghi file .drawio rồi tự kiểm hình học. Đường ra mặc định nằm cạnh file nguồn, cùng tên, đuôi .drawio.
- --png: xuất ảnh bằng drawio CLI (scale 2), mặc định cạnh file .drawio.
- --verify: xuất SVG bằng drawio CLI rồi so từng đường dây draw.io vẽ ra với toạ độ đã tính. Bước này bắt trường hợp draw.io tự đi lại đường khác waypoint.
- --mode: bắt buộc khi file đích đã tồn tại, xem mục Generate lại bên dưới.
- --no-backup: không tạo file .drawio.bak trước khi ghi đè.
- --sheet, --delimiter, --encoding: chỉ dùng cho xlsx và csv, xem mục Định dạng đầu vào bên dưới.
- Tool đọc bảng đầu tiên có header id | type | parent | content | metadata.

Exit code:

| code | ý nghĩa |
|---|---|
| 0 | ổn (có thể vẫn có cảnh báo) |
| 1 | bảng có lỗi, không vẽ |
| 2 | layout tự kiểm có lỗi (dây cắt node, hai dây chồng nhau, node chồng nhau) |
| 3 | ảnh render lệch toạ độ, hoặc drawio CLI lỗi |
| 4 | file đích đã tồn tại mà chưa chọn chế độ, hoặc không đọc được file đích |

Các dòng in ra có dạng ERROR/WARNING kèm số dòng và id trong bảng, hoặc kèm tiền tố layout: / render:.

## Định dạng đầu vào

Thiết kế đầy đủ: docs/input-formats-design.md.

.md, .csv và .xlsx là ba nguồn ngang hàng: build đọc thẳng, không có bước chuyển đổi trung gian, và cùng một nội dung bảng thì ba định dạng cho ra file .drawio giống hệt nhau.

Chung cho cả ba:
- Hàng header là hàng đầu tiên có đủ 5 tên cột id, type, parent, content, metadata.
- Tiêu đề sơ đồ lấy theo thứ tự: cờ --title, heading # đầu tiên (md) hoặc ô không rỗng đầu tiên phía trên header (csv, xlsx), cuối cùng là tên file bỏ đuôi.

Riêng csv và xlsx:
- Mọi hàng phía trên header bị bỏ qua, nên đầu file được phép có tiêu đề và ghi chú. Bảng kết thúc ở hàng đầu tiên có cả 5 ô đều rỗng, nên phía dưới cũng ghi chú được. Cột thừa bên phải bị bỏ qua.
- Xuống dòng trong ô: nhận cả xuống dòng thật lẫn thẻ br. Gạch đứng là ký tự thường. Không xử lý escape kiểu markdown, và tool cảnh báo khi thấy gạch chéo ngược kèm gạch đứng vì đó thường là dán từ md sang mà chưa dọn.

csv:
- Dấu phân cách tự đoán trong dấu phẩy, chấm phẩy, tab; bảng mã thử utf-8-sig, utf-8, rồi cp1252. Phải lùi về cp1252 thì tool cảnh báo.
- Đoán sai thì ghi đè bằng --delimiter và --encoding.

xlsx:
- Lấy sheet đầu tiên có hàng header, hoặc sheet chỉ định bằng --sheet. Tên sheet đã dùng được in ra và mọi lỗi đều ghi kèm tên sheet.
- Ô công thức lấy giá trị đã lưu trong file; chưa có giá trị thì đọc thành rỗng và cảnh báo.
- Cảnh báo với hàng ẩn trong vùng bảng, ô gộp trong vùng bảng, và ô kiểu số ở cột id hoặc parent (id dạng 4.10 bị Excel đổi thành 4.1, nên để cột đó là Text).

## Cách tool xếp hình

1. Xếp hàng: tool duyệt theo thứ tự topo, cùng mức thì ưu tiên thứ tự dòng trong bảng, và bỏ qua cạnh back=true. Đích chỉ có một nguồn thì được đặt cùng hàng với nguồn để mũi tên đi ngang, miễn là các ô trên đường đi còn trống; điều này áp dụng cho cả đích khác lane lẫn nhánh phụ cùng lane, nên một nhánh phụ không bắt buộc phải tụt xuống dưới node đã rẽ ra nó. Riêng condition có từ 2 nhánh phụ trở lên thì không nhận mũi tên ngang.
   Node có nhiều nhánh cùng lane đi vào thì quay về cột của node rẽ gần nhất mà mọi nhánh đều đi qua, thay vì bám theo cột của nhánh được xếp sau cùng. Nhờ vậy luồng chung sau khi hợp nhánh nằm thẳng cột với chỗ đã rẽ ra. Không có node rẽ chung thì giữ cách cũ.
2. Cột con: cạnh ra cuối cùng của một node được coi là nhánh chính và giữ nguyên cột. Mỗi nhánh phụ được đặt về phía lane mà nhánh đó rốt cuộc dẫn tới: tool đi dọc nhánh tới cạnh đầu tiên rời khỏi lane, và dừng ở node hợp nhánh vì từ đó là luồng chung. Nhờ vậy một nhánh chỉ để trả lỗi về lane bên trái sẽ nằm bên trái, mũi tên ra khỏi nó không phải vòng ngược qua các node khác. Nhánh không rời lane thì không rõ hướng, và các nhánh như vậy lần lượt sang phải rồi sang trái. Nếu một mặt đã có mũi tên ngang thì mọi nhánh phụ dồn sang mặt còn lại. Condition có từ 2 nhánh phụ trở lên thì không nhận mũi tên ngang, để giữ cả hai mặt bên cho các nhánh.
3. db và text đứng cùng hàng với node được attach, về phía không có cạnh nối, và bám sát node đó đúng --attach-gap. Ô sát bên được ưu tiên hơn phía thuận: mặt thuận đã kín thì tool lấy ô sát bên mặt kia trước khi lùi ra xa. Mặt đang đỡ một db hoặc text không được dùng làm cổng ra, nên cạnh đi ra sẽ chọn mặt khác hoặc đi xuống đáy; nhờ vậy không có đoạn dây nào cắt qua chỗ đã chừa. Máng bên cạnh được nới thêm đúng phần chừa đó, và dây chạy ở phía ngoài. Khi mặt đó đã có dây ngang không tránh được, hoặc node có nhiều hơn một db/text cùng phía, thì phần tử không bám sát được và quay về cách cũ là căn giữa ô bên cạnh.
4. Đi dây: có 4 kiểu.
   - A: thẳng đứng.
   - B: thẳng ngang.
   - C: hình chữ L, ra từ mặt bên rồi đi xuống đỉnh node đích.
   - D: tổng quát, đi qua kênh ngang giữa các hàng và máng dọc giữa các cột.
   Mỗi kênh hoặc máng được chia thành nhiều track, gán theo thuật toán tô màu khoảng. Các cạnh cùng đích được dùng chung track, nên chúng gộp thành một đường chung.
5. Nhãn cạnh: tool thử các vị trí dọc theo đường dây, gần nguồn trước, và chọn chỗ đầu tiên không đè node, nhãn khác, dây khác, header hay biên lane. Kênh và máng được chừa thêm chỗ cho nhãn.
6. Chữ trong node được đo bằng độ rộng glyph thật của Verdana.ttf và ngắt dòng sẵn sao cho các dòng dài xấp xỉ nhau. Vì vậy style của node không bật whiteSpace=wrap: nếu bật, draw.io có thể ngắt ra số dòng khác với kích thước node đã tính. Không có font thì tool dùng độ rộng ước lượng và in cảnh báo.

## Generate lại khi file .drawio đã có

Thiết kế đầy đủ: docs/merge-design.md.

File đích đã tồn tại thì tool không tự ghi đè. Chạy trong terminal, tool hỏi chọn merge hay force. Chạy không có terminal (agent gọi), tool dừng với exit code 4 và yêu cầu truyền --mode. Trước khi ghi đè, tool chép file cũ thành <file>.drawio.bak.

force sinh lại toàn bộ, bỏ mọi chỉnh sửa tay.

merge giữ lại từ file hiện có, đối chiếu theo id:
- vị trí element (neo theo tâm; kích thước vẫn tính lại theo chữ);
- vị trí và bề rộng lane; lane chỉ được nới khi có element không vừa, khi đó các lane bên phải dịch sang;
- waypoint và điểm neo của edge, với điều kiện hai đầu edge đều là element cũ và from/to không đổi;
- cell người dùng tự vẽ (không mang dấu flowtable=1 trong style), kể cả chữ và style của chúng.

Sinh lại từ bảng: chữ, style, loại hình, lane chứa phần tử, thêm hoặc xoá element và edge.

Những chỗ cần chỉnh tay sau khi merge đều được liệt kê trong báo cáo:
- element mới, và element mới phải dịch xuống vì chồng chỗ;
- edge nối tới element mới hoặc đã đổi from/to: draw.io tự đi dây nên có thể cắt qua node;
- edge đã sửa tay: nhãn về giữa đường;
- cell tự vẽ bị mất đầu nối vì node kia đã bị xoá khỏi bảng.

Ở chế độ merge, tool không tự kiểm dây và nhãn, cũng không kiểm render, vì đường dây lúc này đến từ file cũ hoặc do draw.io tự đi.

## Tham số layout

Mọi trường của Config đều có cờ CLI tương ứng, ví dụ --track-gap 14 hay --task-max-w 280. Tất cả nhận số nguyên (px).

| cờ | mặc định | tác dụng |
|---|---|---|
| --task-min-w / --task-max-w | 120 / 240 | bề rộng hộp task |
| --cond-wrap | 150 | bề rộng ngắt dòng trong hình thoi |
| --term-wrap | 170 | ngắt dòng start/end/external |
| --db-wrap / --text-wrap | 130 / 260 | ngắt dòng db / ghi chú |
| --label-wrap | 180 | ngắt dòng nhãn cạnh |
| --track-gap | 12 | khoảng cách giữa hai track |
| --gutter-margin / --channel-margin | 15 / 12 | lề từ mép máng/kênh tới track đầu |
| --min-gutter / --min-channel | 24 / 30 | khoảng trống tối thiểu giữa hai cột / hai hàng |
| --attach-gap | 40 | khoảng cách từ db/text tới node nó attach |

## Giới hạn đã biết

- Mới hỗ trợ kiểu activity-swimlane dọc.
- Hình thoi và elip chỉ nối dây ở 4 điểm giữa cạnh. Condition có nhiều nhánh ra hơn số mặt còn trống thì các nhánh dùng chung một điểm ra rồi mới tách.
- Nếu cả hai mặt của node đều có cạnh sang lane khác, db/text được đặt bên phải. Khi đó cạnh ra bên phải phải đi vòng qua kênh.
- Nhánh kết thúc sớm có thể tạo một đường chung dài chạy dọc lane. Tool coi đó là chấp nhận được, không tự đổi cách xếp.

## Test

```
python3 -m unittest discover -s tests
```

Fixtures nằm trong tests/fixtures:
- order.md: ví dụ trong tài liệu định dạng.
- retry.md: có cạnh back, condition 3 nhánh và mục "Phần còn lại".

test_flowtable2drawio.py kiểm parse, validate, layout và đi dây. test_merge.py dựng lại các tình huống generate lại: kéo node, thêm điểm gấp, thêm ghi chú tay, chèn và xoá dòng trong bảng, thêm lane, đổi lane của node, file drawio nén, file nhiều trang. test_inputs.py kiểm đọc csv và xlsx, gồm cả bảng mã, dấu phân cách, ô gộp, hàng ẩn, ô công thức, và khẳng định md, csv, xlsx cho ra cùng một file. Helper xlsx_fixture.py sinh file .xlsx ngay trong test, không cần thư viện ngoài.
