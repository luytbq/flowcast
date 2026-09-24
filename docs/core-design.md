# Thiết kế core

Tài liệu này mô tả core của flowcast: nó hứa gì với caller, dữ liệu đi qua nó ra
sao, và vì sao nó được cắt như vậy. Từ vựng định nghĩa trong
[CONTEXT.md](../CONTEXT.md). Vị trí code nằm ở [structure.md](structure.md),
thuật toán xếp hình ở [algorithm.md](algorithm.md). Các quyết định kèm lý do
nằm trong [adr/](adr/).

Mục tiêu: một core phục vụ được hai caller rất khác nhau, một CLI trên máy có
filesystem và một request HTTP chỉ có bytes, mà không bên nào phải viết lại
chính sách của bên kia.

## 1. Luật định hình mọi thứ

Core không chạm filesystem, không gọi tiến trình ngoài, không đọc biến môi
trường, không in ra stdout. Vào là bytes, ra là văn bản đích cộng dữ liệu chẩn
đoán.

Ba việc vì vậy nằm ngoài core: đọc ghi file (CLI), gọi drawio để xuất ảnh
(package render), và biến kết quả thành dòng in ra hay mã thoát (CLI và web).

Hệ quả kiểm thử: mọi test của core chạy được không cần file tạm, và mọi lỗi hoặc
là một Issue trả về, hoặc là một model.Error có mã. Không có đường thứ ba. Kể cả
khi engine tự vi phạm một bất biến và panic, Build bắt lại và trả lỗi mã
layout.internal.

## 2. Ràng buộc kỹ thuật

Go, module github.com/luytbq/flowcast, mốc tương thích là phiên bản Go ổn định
hiện hành.

Core chỉ dùng thư viện chuẩn, trừ một ngoại lệ. archive/zip và encoding/xml đủ để
đọc xlsx; số đo font nhúng sẵn nên không cần thư viện font. Ngoại lệ duy nhất là
golang.org/x/text/unicode/norm cho chuẩn hóa NFC, vì thư viện chuẩn không có và
viết lại chuẩn hóa Unicode cho đúng là việc dễ sai; lý do đầy đủ trong ADR-0005.

WASM chưa nằm trong phạm vi, nhưng thiết kế không đóng cửa: core không chạm I/O
nên build cho js/wasm là việc thêm một target, không phải thiết kế lại.

## 3. Interface công khai

Core phơi ra hai hàm. Mọi thứ khác là kiểu dữ liệu.

```go
func Build(src Source, opt Options) (Result, error) // đọc, kiểm, xếp hình, sinh file
func Check(src Source, opt Options) (Result, error) // chỉ đọc và kiểm bảng
```

```go
type Source struct {
    Data        []byte
    Name        string            // tiêu đề dự phòng, và để đoán định dạng; không dùng để mở file
    Format      string            // rỗng nghĩa là tự đoán từ Name
    Options     map[string]string // tham số riêng của định dạng: sheet, delimiter, encoding
    MaxUnzipped int64
}

type Options struct {
    Config    *layout.Config // tham số xếp hình; nil là mặc định
    Title     string         // thay tiêu đề lấy từ nguồn
    Previous  *merge.Old     // file .drawio cũ để giữ chỉnh sửa tay; nil là sinh mới
    Direction string         // TD, BT, LR, RL; rỗng là theo nguồn, rồi TD
    Limits    *Limits        // giới hạn tài nguyên; nil là không chặn
}

type Result struct {
    Title, Source string
    Issues   []Issue          // phát hiện về bảng đầu vào
    Text     string           // file .drawio; rỗng khi bảng có lỗi
    Warnings []Warning        // cảnh báo của engine, có mã
    Findings []Finding        // kết quả tự kiểm hình học, có mã
    Layout   *layout.Result   // toạ độ đã tính
    Merge    *merge.Report    // báo cáo merge khi có Previous
    Stats    Stats            // số lane, phần tử, cạnh, kích thước pool
}
```

Result không mang mã thoát và không mang chuỗi đã định dạng sẵn để in. CLI tự
ánh xạ sang mã thoát, web tự ánh xạ sang HTTP status. Đây là chỗ duy nhất hai
front-end được phép khác nhau về chính sách.

## 4. Mô hình lỗi

Mọi phát hiện đều có mã máy ổn định, đặt theo miền, dấu chấm, rồi triệu chứng.
Mã là một phần của interface công khai: đổi một mã là một thay đổi phá vỡ tương
thích.

| Tiền tố | Miền | Dạng |
|---|---|---|
| source. | đọc và giải mã đầu vào | Issue hoặc Error |
| table. | cấu trúc bảng, header, ô | Issue |
| ref. | tham chiếu id: from, to, attach, parent | Issue |
| schema. | key metadata thiếu, lạ, sai miền; tham số xếp hình ngoài miền | Issue hoặc Error |
| order. | thứ tự dòng theo đặc tả | Issue |
| mermaid. | cú pháp mermaid không có chỗ chứa | Issue hoặc Error |
| config. | tham số của lần dựng, như hướng | Error |
| merge. | merge không làm được | Error |
| limit. | vượt giới hạn tài nguyên | Error |
| layout. | cảnh báo của engine; layout.internal là lỗi nội bộ | Warning hoặc Error |
| check. | tự kiểm hình học | Finding |

Issue nói về nội dung bảng người dùng viết, và mang vị trí có cấu trúc (dòng, ô,
sheet, dòng trong mermaid) để web trỏ đúng chỗ. model.Error dành cho thứ khiến
việc dựng không thể tiếp tục và không quy được về một dòng: không đoán được định
dạng, file hỏng, vượt giới hạn, tham số sai. Warning là chỗ engine vẫn dựng được
nhưng phải chấp nhận một phương án kém hơn. Finding là kết quả tự kiểm.

Thông điệp tiếng Việt nằm trong core, cạnh chỗ phát hiện lỗi. Muốn đa ngôn ngữ
về sau thì đã có mã để tra, không phải sửa lại chỗ phát hiện.

## 5. Bảng và lược đồ metadata

Bảng giữ đúng 5 cột id, type, parent, content, metadata (ADR-0001). Vì cột
metadata chở cả tham chiếu bắt buộc là from, to, attach, tính tường minh phải
nằm ở nơi khác: lược đồ metadata trong package schema, khai báo từng type nhận
những key nào, key nào bắt buộc, key nào trỏ tới id khác, và miền giá trị.

validate có hai phần: một vòng chung chạy trên lược đồ, bắt key thiếu, key lạ,
giá trị ngoài miền và tham chiếu treo; cộng các luật riêng của sơ đồ mà lược đồ
không diễn đạt được, như condition phải có từ hai cạnh ra và cạnh ra phải nằm
liền sau node.

parent là cây chứa đựng đơn cha, không chỉ là "id của một lane". Định nghĩa này
phủ được lane, subgraph của mermaid, và về sau là subprocess, mà không tốn gì
thêm hôm nay.

## 6. Cấu hình

Cấu hình chia hai tầng. Giới hạn tài nguyên không phụ thuộc loại sơ đồ nên nằm ở
package gốc, dưới dạng hai hồ sơ dữ liệu: CLILimits nới, WebLimits siết. Hồ sơ là
dữ liệu, không phải nhánh if trong core.

Tham số xếp hình thuộc về engine và được khai báo thành danh sách Field: tên, mặc
định, miền giá trị, lời giải thích. CLI sinh cờ từ danh sách này, web sinh form
và kiểm giá trị từ cùng danh sách đó, nên hai bên không thể lệch nhau.

Giá trị ngoài miền là lỗi dừng việc dựng, mã schema.out_of_range, chứ không phải
Issue: Issue nói về nội dung bảng, còn đây là tham số người gọi truyền vào, và
không có kết quả bộ phận nào đáng trả về. Miền để rộng, vì cấu hình cho ra bố
cục xấu vẫn là quyền của người dùng và tự kiểm sẽ báo; chỉ giá trị làm engine
chạy sai mới bị chặn.

## 7. Hướng vẽ

Engine chỉ xếp theo một hướng, từ trên xuống. Các hướng khác được dựng bằng cách
đổi trục kết quả ở cuối (ADR-0006). Các pha xếp chỗ, đi dây, hình học và nhãn
không có nhánh code nào cho từng hướng.

Hộp phần tử không lật theo trục: chữ vẫn đọc từ trái sang phải, bề rộng hộp vẫn
bị chặn bởi bề rộng tối đa, chiều cao vẫn mọc theo số dòng. Chỉ có lưới lật. Với
hướng ngang, bề rộng và bề cao được hoán đổi trước khi xếp để hàng của không gian
ảo có đúng độ dày của cột thật.

## 8. Kết quả xếp hình

layout.Result là dữ liệu thuần: kích thước pool, lane, phần tử kèm toạ độ, cạnh
kèm điểm gấp, cổng và vị trí nhãn. Tự kiểm, merge và writer chỉ đọc Result,
không gọi gì vào engine.

Result nói bằng hình nguyên thủy (chữ nhật, thoi, elip, elip đôi, elip nét đứt,
trụ, ghi chú), không bằng loại ngữ nghĩa. Mỗi loại phần tử khai báo một lần hình
nguyên thủy và luật nối dây của nó trong layout.Kinds; engine và writer chỉ đọc
khai báo đó. Nếu writer phải biết loại ngữ nghĩa thì mỗi writer phải biết mọi
loại, và số việc thành tích của hai con số thay vì tổng. Loại ngữ nghĩa vẫn được
mang theo trong Result cho công cụ và cho gỡ lỗi, nhưng writer không đọc nó.

Tự kiểm chạy trên Result. Nhờ vậy nó kiểm được mọi đường sinh, và kiểm được cả
hình học hỏng dựng tay, là cách duy nhất để kiểm chính tự kiểm.

## 9. Merge

Merge sinh lại sơ đồ mà giữ những gì người dùng đã sửa tay trong file .drawio
cũ: vị trí node, bề rộng lane, điểm gấp của dây, và các cell tự vẽ. Thiết kế chi
tiết nằm ở [merge-design.md](merge-design.md).

Đọc file cũ là việc của package merge, nhận bytes chứ không nhận đường dẫn. Chỉ
CLI dùng merge, vì chỉ CLI có file cũ nằm cạnh bảng; dịch vụ web không nhận file
cũ (ADR-0004). Merge chỉ hỗ trợ hướng từ trên xuống và báo lỗi merge.direction
với các hướng khác.

Một bất biến canh merge: merge trên file vừa sinh, chưa ai sửa, không được làm
đổi một byte nào. File chỉ ghi số đã làm tròn hai chữ số, nên mọi phép so giữa
số đọc lại và số tính được phải làm tròn giống hệt lúc ghi.

## 10. Giới hạn tài nguyên

Chặn trên số dòng và số cạnh là biện pháp chính, vì chúng chặn luôn thời gian
chạy: thời gian dựng tăng gần theo bình phương số phần tử. Timeout là biện pháp
phụ, kiểm ở ranh giới giữa các pha, không cắt ngang một pha. Với xlsx còn chặn
tổng số byte giải nén, vì một file zip vài KB có thể giải ra hàng GB.

Vượt giới hạn là lỗi mã limit.*, không phải Issue, vì không có kết quả bộ phận
nào đáng trả về.

## 11. Font và tính tất định

Cam kết: cùng một đầu vào, cùng một cấu hình, cùng một phiên bản flowcast thì ra
cùng một chuỗi byte, trên mọi máy.

Tool không đọc file font lúc chạy. Nó đo chữ bằng bảng advance width theo từng
codepoint, sinh một lần từ file font bằng công cụ trích số đo trong tools và
nhúng vào binary. Nhờ vậy binary phân phối đi không mang theo font và không phụ
thuộc máy đích có cài Verdana hay không.

Một rủi ro cần ghi rõ: Verdana là font thương mại của Microsoft. Phân phối lại
một bảng số đo khác hẳn với phân phối lại font, nhưng nếu dịch vụ này công khai
thì cần người hiểu luật xem qua. Phương án dự phòng: đổi fontFamily trong đầu ra
sang một font tự do và sinh bảng đo từ font đó.

### Số thực phải tất định tới từng bit

Pha hình học so sánh số thực để ra quyết định: nhãn đặt ở ứng viên nào tùy tổng
chi phí nào nhỏ hơn, điểm gấp nào bị bỏ tùy hai toạ độ có cách nhau dưới 0.01
hay không. Lệch một đơn vị ở bit cuối là đủ lật một quyết định, nên kết quả phải
tất định tới từng bit, không chỉ tới hai chữ số thập phân.

Các chỗ dễ lệch, đều đã có phép kiểm canh:

- **Làm tròn.** num.Round làm tròn trên giá trị nhị phân thật, nửa chính xác về
  số chẵn, bằng cách đi qua chuỗi thập phân. math.Round sai ở những số như 2.675
  và 0.125.
- **Nhân rồi cộng.** Đặc tả Go cho phép gộp a*b + c thành một lệnh FMA, chỉ làm
  tròn một lần, và được gộp cả qua nhiều câu lệnh. Trên arm64 điều đó xảy ra ở
  khoảng một phần tư số phép a*1.42 + c. Chỉ phép chuyển kiểu tường minh chặn
  được việc gộp, nên mọi phép nhân số thực phải bọc trong float64(), trừ khi kết
  quả đi thẳng vào một phép nhân hay chia khác. Một test tĩnh trong
  internal/lint kiểm quy tắc này trên toàn module ở mỗi lần go test.
- **Thứ tự cộng.** Cộng số thực không có tính kết hợp, nên thứ tự cộng dồn toạ độ
  và chi phí là một phần của kết quả.
- **Dấu của số không.** max và min trong engine trả số đứng trước khi hai số bằng
  nhau, vì math.Max và math.Min đổi dấu của số không.

## 12. Đầu vào mermaid

mermaid quy về cùng một Table như ba định dạng bảng (ADR-0002). Ánh xạ:

| mermaid | Flow Table |
|---|---|
| node kèm cú pháp ngoặc | một dòng, type suy từ hình: [] task, {} condition, ([]) start hoặc end, [()] db, (()) external |
| nhãn node | content |
| subgraph | một dòng lane, và parent của các node bên trong |
| cạnh kèm nhãn | một dòng edge, content là nhãn, from và to vào metadata |
| nét đứt, nét đậm, không mũi tên | giá trị dashed, bold, noarrow của key style |
| classDef và class | giải ngay lúc đọc thành giá trị style cụ thể trên từng dòng |

id lấy thẳng từ mermaid nên ổn định, và merge theo id chạy được với đầu vào
mermaid mà không cần gì thêm.

Biên của từ vựng metadata: một key được nhận vào chỉ khi nó đổi thứ được vẽ ra,
có tương đương trung thực trong style của draw.io, và không tham chiếu tới dòng
khác. Theo biên đó, các kiểu mũi tên chỉ là giá trị mới của key style đã có.
Màu đặt thẳng trên node không vào, vì key style mang vai trò ngữ nghĩa chứ không
mang màu; màu khớp bảng màu nhấn thì quy về highlight, còn lại cảnh báo. icon,
click, href không đổi bố cục nên không vào. subgraph lồng nhau bị dẹp về
subgraph ngoài cùng kèm cảnh báo mermaid.nested_subgraph.

Mọi thứ bỏ qua đều là một Issue có mã và có số dòng trong nguồn, để web hiện được
danh sách đã bỏ qua những gì.

## 13. Flowchart và nhánh chính

flowchart không phải một loại sơ đồ mới. Nó là activity-swimlane với lane trở
thành tùy chọn: bảng không có dòng lane nào được dựng trên một lane ẩn, thanh
tiêu đề và thanh tên lane bằng 0, và không vẽ pool.

Có một chỗ phải xử lý riêng. Engine xếp nhánh phụ về phía lane mà nhánh đó rốt
cuộc dẫn tới. Trong sơ đồ một lane, không nhánh nào rời lane, nên tín hiệu đó
luôn trống và mọi nhánh rơi về luật dự phòng là lần lượt sang phải rồi sang trái.
Thêm nữa, đặc tả Flow Table để người viết bảng chọn nhánh chính bằng thứ tự dòng,
còn mermaid không có quy ước đó.

Vì vậy với sơ đồ không có lane:

- Nhánh chính là nhánh có đường dài nhất tới một node kết thúc, đo trên đồ thị
  đã bỏ cạnh back, dừng ở node hợp nhánh.
- Xương sống là chuỗi nhánh chính đi từ mỗi điểm đầu. Node hợp nhánh nằm ngoài
  xương sống giữ cột của nguồn sâu nhất thay vì quay về cột của node rẽ, để một
  bước chỉ nhận các nhánh lỗi không kéo luồng chính đi vòng qua nó.

Sơ đồ có lane giữ quy ước của Flow Table.

Số đo lúc đưa hai luật này vào, bằng go run ./tools/metrics trên 28 sơ đồ mà bộ
mermaid và flowchart có khi đó:

| Chỉ số | Trước | Sau |
|---|---|---|
| dây đi vòng qua kênh (kiểu D) | 38 | 33 |
| điểm gấp | 126 | 112 |
| tổng chiều dài dây | 31874 | 29362 |
| cạnh của đường dài nhất vẽ thẳng đứng | 117/152 | 135/152 |
| phát hiện tự kiểm | 0 | 0 |

## 14. Việc đã cân nhắc và để lại

Những thứ dưới đây đã cân nhắc, kèm lý do, để lần sau không phải nghĩ lại từ
đầu.

**Giảm chỗ hai dây cắt nhau.** Lúc đo có 33 chỗ cắt trên 28 sơ đồ của bộ mermaid
và flowchart, tệ nhất là sơ đồ CI với 7 chỗ. Nguyên nhân: dây đi vòng chạy dọc trong máng ngay
cạnh cột nguồn, mà máng đó là chỗ mọi mũi tên ngang xuất phát từ cột ấy đi qua.
Hai phép thử nhanh đã làm và đã bỏ:

| Thử | Chỗ cắt | Dài | Tự kiểm |
|---|---|---|---|
| lúc đo | 33 | 29362 | 0 |
| nhánh không rõ hướng luôn dạt phải | 33 | 29379 | 0 |
| dây đi vòng luôn dùng máng ngoài cùng | 27 | 32530 | 5 |
| như trên, nhưng lùi về khi hai đoạn ngang đâm vào node | 31 | 32066 | 2 |

Hai phép sau làm dây dài thêm gần 10% và sinh ra lỗi hình học thật. Làm tử tế
nghĩa là chọn máng và chọn track cùng lúc, với hàm chi phí gồm cả số chỗ cắt,
chứ không chọn máng trước rồi xếp track sau như hiện nay.

**Tự kiểm sau merge.** Giá trị thấp hơn tưởng: dây do người dùng giữ lại thì
đúng theo định nghĩa, dây để draw.io tự đi thì không có toạ độ để kiểm, và nhãn
không được tính lại sau khi node dịch chỗ. Phần duy nhất còn ý nghĩa là phần tử
chồng nhau, mà báo cáo merge đã liệt kê. Làm đúng việc này nghĩa là một pha đi
dây thứ hai chạy trên hình học do người dùng đặt.

**Merge cho hướng khác từ trên xuống.** Cách rẻ nhất là đổi trục file cũ lúc đọc
rồi đổi ngược lúc ghi, nhưng cell tự vẽ được chép nguyên văn nên cũng phải đổi
trục theo, và đó là chỗ dễ sai. Hiện tại merge báo rõ là chưa hỗ trợ.

**Lane lồng lane.** Mô hình chịu được vì parent đã là cây, nhưng engine chưa xếp
được.

**Bề dày lane trong sơ đồ đi ngang.** Tên lane được xoay dọc, nhưng engine vẫn
lấy bề rộng chữ làm bề dày tối thiểu của băng, nên tên dài làm băng dày quá mức.
Sửa đúng là tách hai số: bề dày theo chiều cao chữ, bề dài theo bề rộng chữ.

**WASM.** Core không chạm I/O, nên chạy thẳng trong trình duyệt là chuyện đóng
gói, không phải chuyện kiến trúc. Chưa làm vì dịch vụ web đã đủ dùng.
