# Thuật toán xếp hình và đi dây

Tài liệu này dành cho người sửa engine xếp hình. Đọc xong, bạn lần ra được pha
nào gây ra một lỗi bố cục, và sửa mà không phá tính tất định: cùng một đầu vào
luôn cho ra cùng một file.

Cần đọc [CONTEXT.md](../CONTEXT.md) trước, nhất là các từ lane, nhánh chính,
flow, cross, máng, kênh và track. Vị trí từng file của engine nằm ở
[structure.md](structure.md).

## Engine chạy qua những pha nào?

Engine không tính điểm ảnh ngay. Nó xếp mọi thứ lên một lưới trừu tượng trước:
mỗi phần tử có một lane, một hàng và một cột trong lane. Mỗi đoạn dây nằm trong
một máng hoặc một kênh, ở một track nào đó. Chỉ khi lưới đã xong thì engine mới
đổi lưới ra điểm ảnh.

Engine chỉ biết một hướng là từ trên xuống. Các hướng khác được dựng trong một
không gian ảo từ trên xuống rồi đổi trục ở cuối.

Các pha theo thứ tự:

1. Đo kích thước từng phần tử.
2. Xếp chỗ: gán lane, hàng, cột.
3. Đi dây: chọn kiểu dây cho từng cạnh, đặt cổng, xếp đoạn dây vào track.
4. Hình học: đổi lưới ra điểm ảnh, dựng đường gấp khúc.
5. Lặp lại pha 4 một lần, sau khi biết thêm chỗ cần chừa cho nhãn.
6. Đặt nhãn.
7. Tự kiểm hình học.
8. Đổi trục theo hướng vẽ.

Pha 1 chạy khi dựng trạng thái ban đầu, pha 7 và 8 chạy trong Build. Các pha
còn lại nằm trong hàm Run.

## 1. Đo kích thước

Mỗi phần tử được ngắt dòng theo bề rộng tối đa của loại nó (task, condition,
start, end, external, db, ghi chú), rồi đo bằng bảng độ rộng ký tự nhúng sẵn
trong binary. Kích thước chỉ phụ thuộc loại, nội dung và cấu hình, không phụ
thuộc chỗ đứng. Vì vậy pha này chạy trước mọi pha khác, và mọi pha sau coi kích
thước là hằng số.

Chữ được ngắt dòng sẵn và ghi vào file kèm thẻ xuống dòng. File đầu ra không bật
chế độ tự ngắt dòng của draw.io, vì draw.io ngắt lại có thể ra số dòng khác với
kích thước đã tính.

## 2. Xếp chỗ

### Thứ tự duyệt

Engine duyệt node theo thứ tự topo, bỏ qua các cạnh đánh dấu back (cạnh vòng
lặp). Hai node cùng mức thì node viết trước trong bảng ra trước. Node nằm trong
một vòng lặp chưa đánh dấu back thì không có thứ tự topo. Các node đó được nối
vào cuối theo thứ tự bảng, kèm một cảnh báo.

db và ghi chú có attach không tham gia thứ tự này. Chúng được đặt ngay sau node
mà chúng bám.

### Hàng

- Node có nguồn đã đặt: hàng bằng hàng sâu nhất trong các nguồn, cộng một.
- Node start không có nguồn: hàng 0.
- Node khác không có nguồn: hàng ngay dưới hàng sâu nhất đã dùng.

Node chỉ có đúng một nguồn thì được thử đặt cùng hàng với nguồn, để mũi tên đi
thẳng ngang. Điều kiện:

- nguồn nằm ở lane khác, hoặc node là nhánh phụ của nguồn;
- các ô nằm giữa nguồn và đích trên hàng đó còn trống, và không có mũi tên ngang
  nào đã đặt chạy qua;
- node không phải condition. Condition chỉ nhận dây vào từ đỉnh, nên luôn đứng
  dưới nguồn;
- node không phải hình không chữ nhật có từ hai nhánh phụ trở lên, vì nó cần cả
  hai mặt bên cho các nhánh.

### Cột và nhánh

Các cạnh ra cùng lane của một node được chia thành nhánh chính và nhánh phụ.

- Sơ đồ có lane: nhánh chính là cạnh viết sau cùng trong bảng. Người viết bảng
  được dặn đặt nhánh đi tiếp dài nhất ở cuối.
- Sơ đồ không có lane (thường đến từ mermaid): nhánh chính là nhánh sâu nhất,
  tính bằng số node trên đường dài nhất đi từ đầu nhánh. Hòa thì lấy cạnh viết
  sau.

Nhánh chính giữ cột của node rẽ. Mỗi nhánh phụ lệch sang một bên:

- Nhánh rốt cuộc rời lane thì lệch về phía lane nó dẫn tới. Engine đi dọc nhánh
  tới cạnh đầu tiên rời lane, và dừng ở node hợp nhánh vì từ đó là luồng chung.
  Nhờ vậy một nhánh trả lỗi về lane bên trái nằm bên trái, dây của nó không phải
  vòng qua node khác.
- Nhánh không rời lane thì không rõ hướng. Các nhánh như vậy lần lượt sang phải
  rồi sang trái.
- Mặt nào của node đã có mũi tên ngang thì mọi nhánh phụ dồn sang mặt kia.

Node có nhiều nhánh cùng lane đi vào (node hợp nhánh) quay về cột của node rẽ
gần nhất mà mọi nhánh đều đi qua. Luồng chung sau khi hợp nhánh vì vậy thẳng cột
với chỗ đã rẽ. Sơ đồ không có lane chỉ áp dụng điều này cho node nằm trên xương
sống, tức đường đi theo nhánh chính từ mỗi điểm đầu.

Ô định đặt đã có người thì nhánh phụ thử dạt thêm sang bên tối đa ba cột, rồi
mới chịu xuống hàng dưới.

### db và ghi chú

db và ghi chú đứng cùng hàng với node mà chúng bám, về phía node không có cạnh
nối. Cả hai phía đều có cạnh thì chọn bên phải. Engine thử ô sát bên phía đó, rồi
ô sát bên phía kia, rồi ô cách hai cột, và cuối cùng lùi xa hơn kèm một cảnh báo.

Cuối pha, các cột và máng của mọi lane được đánh số liên tục trên trục ngang, để
các pha sau so vị trí ngang giữa các lane.

## 3. Đi dây

### Kiểu dây

Luật riêng của từng loại phần tử, như condition chỉ nhận dây ở đỉnh hay hình
nào chỉ có một điểm nối mỗi mặt, đọc từ bảng khai báo hình học của loại đó, không
viết thành điều kiện rải trong code.

Engine quét mọi cạnh bốn lượt, mỗi lượt một kiểu dây, từ đơn giản tới tổng quát.
Một cạnh nhận kiểu đầu tiên mà đường đi của nó còn trống.

| Kiểu | Hình dạng | Điều kiện |
|---|---|---|
| A | Thẳng đứng từ đáy nguồn xuống đỉnh đích | Cùng cột, đích ở hàng dưới, các ô giữa còn trống hoặc chỉ bị dây khác cùng đích chiếm, và đáy nguồn chưa có dây ra |
| B | Thẳng ngang từ mặt bên nguồn sang mặt bên đích | Cùng hàng, khác cột, hai mặt đó chưa có dây và không có db hay ghi chú đứng cạnh, các ô giữa còn trống, đích không phải condition |
| C | Chữ L: ra mặt bên nguồn, rẽ xuống đỉnh đích | Đích ở hàng dưới và khác cột, mặt ra còn trống, cả đoạn ngang lẫn đoạn dọc còn trống |
| D | Tổng quát: đi qua máng và kênh | Mọi cạnh còn lại, kể cả cạnh vòng lặp |

Thứ tự lượt là cố ý: dây thẳng được giữ chỗ trước, nên dây phức tạp phải tránh
chúng chứ không ngược lại.

Dây kiểu A, C, D luôn vào đỉnh đích. Chỉ dây kiểu B vào mặt bên.

### Đường đi của dây kiểu D

Trước hết engine chọn mặt ra. Thứ tự thử là mặt hướng về đích, rồi đáy, rồi mặt
đối diện. Đích nằm ngang hàng hoặc phía trên thì không thử đáy. Engine lấy mặt
đầu tiên còn trống. Một mặt không còn trống khi:

- có db hoặc ghi chú đứng cạnh mặt đó;
- mặt đó đã có dây vào;
- node không phải hình chữ nhật và mặt đó đã có dây ra, vì hình thoi và elip chỉ
  có một điểm nối giữa mỗi mặt;
- node là hình chữ nhật và mặt đó đã có dây kiểu B hoặc C, vì hai kiểu này ra
  đúng giữa mặt.

Không mặt nào trống thì engine ra ở mặt hướng về đích, và bước đặt cổng sẽ tách
cổng ra khỏi các dây vào cùng mặt.

Sau đó đường đi phụ thuộc mặt ra:

- Ra mặt bên: chạy dọc trong máng sát mặt đó, tới kênh ngay trên hàng đích, rồi
  chạy ngang trong kênh tới cột đích và đi xuống đỉnh đích.
- Ra đáy, đích ở hàng ngay dưới: chạy ngang trong kênh ngay dưới nguồn tới cột
  đích.
- Ra đáy, cột đích còn trống suốt từ hàng nguồn xuống: chạy ngang trong kênh ngay
  dưới nguồn, rồi đi thẳng xuống theo cột đích.
- Ra đáy, còn lại: chạy ngang trong kênh ngay dưới nguồn tới máng cạnh cột đích,
  dọc máng xuống kênh trên hàng đích, rồi ngang vào cột đích.

Mỗi đoạn ngang hay dọc được ghi thành một khoảng nằm trên một máng hoặc một kênh.
Toạ độ của đoạn chưa có. Engine chỉ ghi lại đoạn nào nối với đoạn nào và với
node nào, để pha hình học điền điểm ảnh vào sau.

### Đặt cổng

Cổng là điểm dây chạm vào node, ghi bằng tỉ lệ trên bề rộng và bề cao của node.

- Dây vào luôn nối giữa mặt.
- Dây ra cùng một mặt có chung nguồn, nên được phép gộp. Hình thoi và elip cho
  chúng ra chung đúng giữa mặt. Hình chữ nhật giữ dây A, B, C ở giữa mặt và chia
  phần còn lại của mặt cho các dây D. Dây rẽ về phía nào thì lấy cổng ở phía đó,
  để các đoạn đầu không cắt nhau.
- Mặt vừa có dây ra vừa có dây vào thì dây ra phải tách khỏi giữa mặt, vì dây vào
  khác cả nguồn lẫn đích nên chồng lên nó là sai. Có một dây ra thì cổng lệch
  một phần tư mặt về phía dây sẽ rẽ. Với hình thoi và elip, cổng lệch nằm trên
  đường viền của hình chứ không trên khung bao, và được làm tròn tới hai chữ số
  như lúc ghi file.

### Xếp track

Mỗi máng và mỗi kênh là một tài nguyên chứa nhiều đoạn dây. Engine xếp các đoạn
vào track (làn dây) bằng tô màu khoảng, tham lam theo thứ tự đoạn được tạo:

- Hai đoạn chồng nhau trên cùng tài nguyên phải nằm ở hai track khác nhau.
- Hai đoạn cùng đích được dùng chung track, và vẽ ra thành một đường chung.
- Hai đoạn có chân nối vào cùng một vị trí từ hai phía thì đoạn đến từ phía thấp
  phải nằm ở track nhỏ hơn. Nếu không, hai chân sẽ đè lên nhau.

Không track nào nhận được đoạn thì engine chèn một track mới vào vị trí thỏa ràng
buộc thứ tự. Không có vị trí nào thỏa thì track mới nằm ở cuối, kèm một cảnh báo.

Vì thuật toán tham lam theo thứ tự tạo đoạn, thứ tự đó là một phần của kết quả.
Đổi thứ tự quét cạnh hay thứ tự tạo đoạn là đổi bố cục.

## 4. Hình học

Pha này đổi lưới ra điểm ảnh.

Bề rộng mỗi cột bằng phần tử rộng nhất trong cột. Bề rộng mỗi máng là tổng của:

- số track nhân khoảng cách track, cộng lề hai bên, không nhỏ hơn khoảng trống
  tối thiểu giữa hai cột;
- chỗ chừa cho nhãn của các dây ra mặt bên;
- chỗ chừa cho db hoặc ghi chú đứng sát node.

Lane có tên dài hơn tổng bề rộng các cột và máng thì phần thiếu được chia đều
cho hai máng ngoài cùng.

Trục dọc làm tương tự. Hàng cao bằng phần tử cao nhất trong hàng. Kênh cao theo
số track cộng lề, cộng chỗ chừa cho nhãn của các dây ra đáy, và không nhỏ hơn
khoảng trống tối thiểu giữa hai hàng.

Phần tử được căn giữa trong ô. db và ghi chú đứng sát node thì đặt cách mép node
đúng khoảng cách bám. Chúng chỉ đứng sát khi mặt đó của node có đúng một phần tử
như vậy và không có dây nào ra vào mặt đó. Còn lại chúng căn giữa ô bên cạnh như
mọi phần tử khác.

### Dựng đường gấp khúc

Mỗi dây bắt đầu ở cổng ra và kết thúc ở cổng vào. Các điểm giữa lấy toạ độ ngang
từ track của đoạn dọc, toạ độ dọc từ track của đoạn ngang, hoặc từ tâm cột đích.
Sau đó engine bỏ các điểm trùng nhau và các điểm nằm giữa hai đoạn thẳng hàng,
rồi làm tròn mọi toạ độ tới hai chữ số thập phân.

## 5. Lượt hình học thứ hai

Lượt đầu chưa biết đoạn ngang đầu tiên của các dây B và C dài bao nhiêu, nên chưa
biết nhãn của chúng có vừa không. Sau lượt đầu, engine đo các đoạn đó, chừa thêm
chỗ trong máng cho những nhãn không vừa, rồi chạy lại toàn bộ pha 4.

## 6. Đặt nhãn

Với mỗi dây có nhãn, engine liệt kê các chỗ có thể đặt, theo thứ tự ưu tiên:

- đoạn gần nguồn trước;
- trên mỗi đoạn đủ dài, thử sát đầu đoạn rồi giữa đoạn;
- mỗi chỗ thử cả hai bên đường.

Mỗi chỗ có một chi phí: diện tích đè lên node, lên nhãn đã đặt và lên thanh tiêu
đề, cộng một khoản cố định cho mỗi dây khác hay đường phân cách lane cắt qua.
Chỗ đầu tiên có chi phí bằng không được nhận ngay. Không có thì lấy chỗ rẻ nhất
và ghi cảnh báo.

Nhãn đặt trước chiếm chỗ của nhãn đặt sau, nên thứ tự dây trong bảng ảnh hưởng
tới vị trí nhãn.

## 7. Tự kiểm

Tự kiểm chạy trên Result, không đọc trạng thái của engine. Nó báo lỗi khi:

- hai phần tử chồng lên nhau;
- một phần tử tràn ra ngoài lane;
- một đoạn dây không nằm ngang hay dọc;
- một dây cắt qua phần tử, trừ đoạn đầu và đoạn cuối chạm vào chính nguồn và
  đích của nó;
- hai dây nằm chồng lên nhau trên cùng một đường, trừ khi cùng nguồn hoặc cùng
  đích, vì khi đó chúng được phép gộp.

Nó báo cảnh báo khi nhãn đè lên phần tử, lên nhãn khác, hoặc lên dây khác.

Một engine đúng không bao giờ sinh ra lỗi ở đây. Tự kiểm báo lỗi nghĩa là engine
có lỗi, không phải bảng đầu vào có lỗi.

## 8. Đổi trục

Với hướng trái sang phải và phải sang trái, bề rộng và bề cao của mọi phần tử và
nhãn được hoán đổi trước khi xếp. Nhờ vậy hàng của không gian ảo có đúng độ dày
của cột thật. Chữ vẫn được ngắt dòng theo chiều thật.

Các pha 2 tới 7 chạy y như sơ đồ từ trên xuống. Sau tự kiểm, mọi toạ độ, tỉ lệ
cổng và hộp nhãn được đổi sang hướng thật:

- trái sang phải: đổi chỗ hai trục;
- dưới lên trên: lật chiều dọc của vùng nội dung;
- phải sang trái: đổi chỗ hai trục rồi lật.

Khi lật, thanh tiêu đề sơ đồ và thanh tên lane vẫn ở đầu băng, chỉ nội dung đổi
chiều. Đổi trục và lật hình giữ nguyên mọi
quan hệ chồng lấn và khoảng cách, nên tự kiểm trên không gian ảo vẫn đúng cho
hình thật.

## Làm sao lần ra pha gây lỗi?

Chạy build với cờ --layout-json để ghi toạ độ ra JSON. File này có hàng và cột
của từng phần tử, kiểu dây, mặt ra và các điểm của từng dây. Sau đó dò theo
triệu chứng:

| Triệu chứng | Nhìn vào |
|---|---|
| Node đứng sai hàng, sai cột, sai lane | Xếp chỗ: hàng, nhánh chính, hướng dạt của nhánh phụ |
| Dây đi đường vòng dù có đường thẳng | Đi dây: kiểu dây đã chọn và lý do các kiểu đơn giản hơn bị loại |
| Dây ra nhầm mặt | Đi dây: thứ tự thử mặt ra và điều kiện mặt còn trống |
| Hai dây chồng nhau | Đặt cổng (cùng điểm nối) hoặc xếp track (cùng làn) |
| Khoảng trống quá rộng hoặc quá hẹp | Hình học: chỗ chừa cho nhãn và cho db, ghi chú đứng sát |
| Nhãn đặt ở chỗ lạ | Đặt nhãn: danh sách chỗ thử và chi phí |
| Chỉ sai ở hướng khác từ trên xuống | Đổi trục |

Sửa xong, kiểm bằng ảnh: build với cờ --png để xem, và cờ --verify để draw.io
xác nhận nó vẽ đúng toạ độ đã tính.

## Giữ tính tất định thế nào?

Cùng một đầu vào phải cho ra cùng một file trên mọi máy. Các quy tắc sau giữ
điều đó. Vi phạm một quy tắc thường không làm test đỏ ngay trên máy của mình, mà
chỉ lộ ra trên máy khác hoặc ở một phiên bản Go khác.

- Không để thứ tự duyệt map quyết định kết quả. Duyệt theo thứ tự bảng, hoặc sắp
  xếp khóa trước khi duyệt.
- Không đổi thứ tự của một phép cộng dồn số thực. Cộng số thực không có tính kết
  hợp, và toạ độ được cộng dồn theo một thứ tự cố định.
- Bọc mọi tích số thực trong float64() trước khi cộng hay trừ tiếp. Nếu không,
  trình biên dịch được phép gộp phép nhân và phép cộng thành một lệnh FMA trên
  một số kiến trúc, cho kết quả khác ở bit cuối. Một test tĩnh trong repo bắt lỗi
  này.
- Khi hai số bằng nhau, max và min trong engine trả số đứng trước, không dùng
  math.Max và math.Min. Hai hàm này đổi dấu của số không, và dấu đó lọt được vào
  file đầu ra.
- Làm tròn toạ độ tới hai chữ số ở cuối, trước khi ghi. Giá trị nào so với số
  đọc lại từ file, như tỉ lệ cổng mà merge so, thì phải làm tròn giống hệt lúc
  ghi.
- Đo chữ bằng bảng độ rộng nhúng sẵn, không bao giờ bằng font cài trên máy.
