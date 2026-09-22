# Bộ đối chiếu

Đây là bộ đáp án cho bản port sang Go. Bản Python trong reference/ sinh ra nó;
bản Go phải tái tạo đúng từng byte.

Lý do tồn tại: khoảng 1800 trên 2646 dòng của bản tham chiếu là heuristic xếp
hình tích lũy cộng ngữ nghĩa merge. Không luật nào trong số đó suy ra được bằng
cách ngồi thiết kế lại; chúng được tìm ra qua nhiều vòng nhìn sơ đồ xấu rồi sửa.
Cách duy nhất mang chúng sang Go mà không phải tái phát hiện là biến bản Python
thành máy sinh đáp án chứ không phải tài liệu tham khảo.

## Cấu trúc

```
cases/              bảng đầu vào, viết tay, mỗi file chốt một nhánh thuật toán
golden/             đầu ra cuối cùng do bản tham chiếu sinh
dumps/              trạng thái trung gian, cổng chặn cho từng module Go
text-vectors.json   vector riêng cho việc đo chữ và ngắt dòng
csv-vectors.json    vector cho bộ đọc csv, so với csv.reader của Python
decode-vectors.json vector cho việc giải mã bảng mã của file csv
round-vectors.json  vector cho num.Round, so từng bit với round() của Python
check-vectors.json  hình học hỏng, cho tự kiểm
merge/              kịch bản merge: bảng đã sửa cùng file .drawio cũ đã sửa tay
cli/                bản ghi phiên làm việc của CLI, kể cả merge
etree-vectors.json  XML ngẫu nhiên, so việc đọc ghi với ElementTree
merge-vectors.json  vector cho float(), b64decode và unquote kiểu Python
verify-vectors.json SVG thật do drawio xuất, cho kiểm render
drawio-fakes/       drawio CLI giả cho bản ghi có --png và --verify
flowchart/          bảng không có lane; golden của chúng nằm ở golden/flowchart
mermaid/            flowchart mermaid kiểu thực tế, golden ở golden/mermaid; cũng là bộ đo cho heuristic nhánh chính
diverge.txt         case mà bản Go cố ý khác bản tham chiếu, kèm lý do
```

Từ bước 14, bản Go có hành vi mà bản tham chiếu không có. Case của hành vi cũ
được ghi vào diverge.txt và test Go bỏ qua chúng; hành vi mới được chốt bằng
case riêng, có golden do bản Go sinh. Golden đó là bộ chống hồi quy, không phải
đáp án: sinh lại bằng `go test ./conformance -run Flowchart -update`, và phải
xem ảnh cùng diff trước khi commit.

Mỗi case cho ra hai file trong golden/:

| file | nội dung |
|---|---|
| `<case>.drawio` | xml đầy đủ, so từng byte |
| `<case>.report.txt` | issue, cảnh báo layout, phát hiện tự kiểm, số liệu |

Case mang tiền tố 9x là bảng có lỗi. Chúng không có file .drawio, chỉ có report.
Đó là cố ý: chúng chốt hành vi kiểm tra đầu vào, thứ bản port cũng phải tái tạo
đúng, kể cả nguyên văn thông điệp và số dòng.

## Dump trung gian

So từng byte file .drawio chỉ làm được ở cuối chuỗi. Port xong layout/place mà
chưa có writer thì không có cách nào biết nó đúng hay sai. Vì vậy mỗi case còn
có một file `dumps/<case>.json` cắt trạng thái thành sáu chặng:

| chặng | chốt module | nội dung |
|---|---|---|
| `table` | source, model | bảng sau parse: idx, loc, id, type, parent, lines, meta |
| `issues` | schema, validate | level, loc, id, msg |
| `text` | text | dòng đã ngắt và bề rộng, bề cao từng phần tử |
| `place` | layout/place | lane, row, col của từng phần tử, cộng thứ tự topo |
| `route` | layout/route, tracks | kiểu đi dây, cổng ra vào, các đoạn dây kèm track |
| `geometry` | layout/geometry, labels | toạ độ cuối cùng, điểm gấp, vị trí nhãn |

Bảng có lỗi thì chỉ có hai chặng `table` và `issues`, vì không có layout nào
được dựng.

Sinh ra bằng `reference/dump.py`, dùng riêng được:

```
python3 reference/dump.py <file> [--stage place]
```

### Ba quy ước để hai ngôn ngữ so được với nhau

**Số.** Mọi số thực đi qua `ft.fmt`, đúng hàm sinh số trong file .drawio, nên
dump và đầu ra cuối cùng chuẩn hóa số giống hệt nhau. Bản Go phải tái tạo cả
những góc kỳ quặc: `fmt(-0.001)` ra `"-0"`, không phải `"0"` hay `"-0.00"`.

**Thứ tự.** Phần tử và cạnh xếp theo thứ tự dòng trong bảng. Đoạn dây giữ đúng
thứ tự được thêm vào: gán track là tham lam theo thứ tự đó, nên cùng tập đoạn
dây thêm theo thứ tự khác sẽ ra track khác. Khóa JSON luôn sắp xếp.

**Mã lỗi không nằm trong dump.** Chặng `issues` chốt level, loc, id và msg. Mã
máy là thứ mới do `docs/core-design.md` quy định, không phải hành vi được port,
nên bản tham chiếu không có gì để đối chiếu ở đó.

### Chặng phải thật sự phản ứng

Một chặng luôn ra cùng một giá trị thì không chốt được gì, và điều đó im lặng
cho tới lúc bản port sai đúng ở module ấy mà vẫn qua cổng. `DumpTest` trong
`reference/tests/test_conformance.py` canh ba điều:

- mỗi chặng phải khác nhau giữa các case;
- `text` và `geometry` phải đổi khi bề rộng hộp đổi;
- `place` và `route` phải **không** đổi khi tham số pixel đổi.

Điều thứ ba là một bất biến của thiết kế, không chỉ của dump: các chặng lưới
sống trên lane, row, col và chưa biết tới pixel. Mục 7 của `docs/core-design.md`
dựa vào đúng tính chất này để LR và BT và RL chỉ là bốn giá trị của một ánh xạ.

## Vector đo chữ

Bộ case sơ đồ quá lỏng để chốt module text, và điều đó chỉ lộ ra khi đem thử
đột biến. Nguyên nhân nằm trong chính thuật toán: `Wrap` thu hẹp về bề rộng nhỏ
nhất vẫn giữ nguyên số dòng, nên đổi ngân sách vài pixel mà không lật số dòng
thì đầu ra y hệt. Đổi `CondWrap` thêm 10 vẫn không làm case nào đỏ.

`text-vectors.json` vá chỗ đó bằng ba lớp:

| lớp | số lượng | chốt |
|---|---|---|
| quét bề rộng từng pixel, 20 tới 300 | 2529 | `Wrap`, `Box`, cờ ngắt cứng |
| thang bậc nội dung dài dần theo từng loại | 770 | `SizeItem` |
| ngân sách ngắt dòng theo từng loại | 9 | `WrapBudget` |

Lớp thứ ba chốt thẳng con số thay vì chốt qua hành vi, vì một hằng số lệch 2px
gần như không lộ ra: ranh giới ký tự phải rơi đúng vào khoảng lệch mới đổi số
dòng, mà chữ rộng khoảng 7px nên cửa sổ 2px thường rỗng.

Sinh lại bằng:

```
python3 tools/text_vectors.py conformance/text-vectors.json
```

## Thử đột biến

```
sh tools/mutate.sh
```

Cố tình làm sai từng chỗ rồi xem test có đỏ không. Một bộ đối chiếu trông đồ sộ
mà không bắt được đột biến nào thì không chốt gì, và điều đó im lặng cho tới lúc
bản port sai thật. Chạy lại sau mỗi lần thêm module Go.

Công cụ khôi phục bằng `git checkout` chứ không bằng sed ngược, và bắt buộc
chuỗi đích xuất hiện đúng một lần. Cả hai luật đó đến từ việc làm sai: sed ngược
từng ghi đè nhầm một mệnh đề canh và để lại code hỏng vẫn qua được test, còn
thay chỗ đầu khi có nhiều chỗ thì để lại bản sao nguyên vẹn và khiến đột biến
vô hại trông như cổng bỏ lọt.

### Thiết kế case cho một đột biến bỏ lọt

```
python3 tools/py_mutant.py --from 'chuỗi gốc' --to 'chuỗi thay' --stage place cases/*.md
```

Áp cùng đột biến lên chính bản Python rồi xem case nào đổi kết quả ở chặng đó.
Case nào làm bản Python đổi thì chắc chắn làm bản Go đỏ, vì bản Go phải khớp bản
Python. Nhanh hơn hẳn đoán một bảng rồi chạy vòng qua Go, và cho biết ngay một
ứng viên có trúng hay không.

Có ba heuristic của place mà nguyên nhân bỏ lọt không hiển nhiên, đáng nhớ khi
thiết kế case cho các pha sau:

- hside chỉ ảnh hưởng tới cột của nhánh phụ khi mũi tên ngang được đặt **trước**
  nhánh phụ, tức đích khác lane phải đứng trước trong bảng.
- Việc chọn nguồn nào làm chuẩn cho node hợp nhánh chỉ lộ ra khi các nguồn nằm
  ở cột khác nhau và không có node rẽ chung.
- Muốn ép nhánh phụ dạt qua nhiều ô bị chặn, dùng mũi tên ngang sang lane khác:
  nó chặn mọi cột dương của lane nguồn trên cùng hàng.

### Khi thiết kế tay quá chậm: sinh bảng ngẫu nhiên

```
python3 tools/fuzz_cases.py tools/mutants/route.py --stage route --seconds 150 --out /tmp/found
```

Ở pha đi dây, điều kiện bị hình học ràng buộc chặt tới mức thiết kế case bằng
tay rất chậm và suy luận dễ sai. Công cụ này sinh hàng nghìn bảng hợp lệ, chạy
từng đột biến trên bản Python, giữ lại bảng nhỏ nhất giết được mỗi đột biến rồi
thu nhỏ tiếp. Lần chạy đầu ở pha đi dây: 107.848 bảng trong 150 giây, giết 16
trên 18, và bản Go khớp cả 16 case ngay lần đầu.

Bài học đáng nhớ: hai nhánh tôi từng suy luận là thừa đều bị bảng ngẫu nhiên
giết. Phép kiểm db trong kiểu B chịu lực vì `attach_sides` xét dấu của cột chứ
không xét có sát hay không, nên một db rơi vào ứng viên `-2 lần phía` có thể
nhảy qua nguồn rồi nằm hẳn bên kia, mà không ô nào giữa hai node bị chiếm. Chỉ
chứng minh tương đương cho những đột biến sống sót qua một lần chạy đủ dài;
danh sách đã chứng minh nằm ở đầu `tools/mutate.sh`.

Các case sinh ra từ đây mang số 60 trở lên và có nội dung kiểu `A-1 x`, `d0`.
Chúng là case hồi quy do máy tìm, không phải ví dụ để đọc: tên file và tiêu đề
nói chúng chốt nhánh nào, còn nội dung giữ nguyên vì đổi chữ là đổi bề rộng
hộp, và có thể làm case thôi giết được đột biến.

## Case csv và xlsx

Sinh bằng `tools/make_csv_cases.py` và `tools/make_xlsx_cases.py`, không viết tay:
vài case cần điều khiển từng byte, như file mã cp1252 hay byte không giải mã
được, và file xlsx cần thời điểm zip cố định để sinh lại ra đúng từng byte.

Bộ đọc csv của Go không dùng `encoding/csv`, vì hai bên hiểu dấu nháy khác nhau:
`"a"b` ra `ab` trong Python nhưng ra `a"b` trong Go. Nó port máy trạng thái của
`csv.reader`, và `csv-vectors.json` chốt nó trên hàng nghìn chuỗi ngẫu nhiên.

## Vector cho tự kiểm

Engine đúng không sinh ra lỗi hình học: trên cả bộ case chỉ 2 bảng có phát
hiện, và không bảng nào chạm tới một lỗi nào. Nên không kiểm được tự kiểm bằng
đầu ra của engine.

Tự kiểm nhận vào `layout.Result`, dữ liệu thuần, nên nó chạy được trên hình học
bất kỳ. `tools/check_vectors.py` lấy hình học hỏng từ hai nguồn: các bản tham
chiếu đã bị đột biến, gồm đột biến ở `tools/mutants/` và năm đột biến phá hỏng
có chủ đích; và bốn hình học dựng tay nằm đúng trên các ngưỡng của tự kiểm,
như hai dây cách nhau 0.3 điểm ảnh, vì engine hỏng không tình cờ rơi đúng ngưỡng.
Phát hiện mong đợi luôn do bản Python tính.

## Lệnh

```
python3 conformance/generate.py            # sinh lại golden và dumps
python3 conformance/generate.py --check    # so với bản đã commit, lệch thì mã thoát 1
python3 tools/coverage.py                  # đo bộ case chạm tới nhánh nào
python3 reference/dump.py <file>           # xem trạng thái trung gian của một bảng
```

Sửa bản tham chiếu mà golden đổi thì phải xem từng thay đổi trong diff rồi mới
sinh lại. Một golden đổi im lặng là một hành vi đã thay đổi mà không ai đọc.

## Tính tất định

Bộ này chỉ có giá trị nếu nó tái tạo được trên máy khác. Hai điều kiện:

1. **Số đo chữ lấy từ data/verdana.json**, không dò font trên máy. Bảng đó sinh
   bằng tools/extract_metrics.py và commit vào repo. generate.py dừng nếu không
   đọc được bảng, thay vì lặng lẽ lùi về độ rộng ước lượng.
2. **Report không chứa đường dẫn, thời gian, hay thứ tự phụ thuộc bảng băm.**

Bản Go cũng đọc data/verdana.json. Nó không đọc file font, nên binary phân phối
đi không mang theo font và không phụ thuộc máy đích có Verdana hay không.

## Độ phủ

tools/coverage.py đo từng nhánh thuật toán và thoát với mã 1 nếu có nhánh chưa
được chạm. Con số hiện tại:

| nhánh | số lần chạm |
|---|---|
| đi dây A (thẳng đứng) | 127 |
| đi dây B (thẳng ngang) | 59 |
| đi dây C (chữ L) | 8 |
| đi dây D (qua kênh và máng) | 62 |
| nhánh dạt trái (drift -1) | 14 |
| nhánh dạt phải (drift 1) | 17 |
| nhánh không rõ hướng (drift 0) | 159 |
| track thứ hai trở lên | 17 |
| db hoặc text bám node | 34 |
| cạnh back | 7 |
| cạnh có nhãn | 112 |
| phần tử tô nhấn | 4 |
| ngắt dòng cứng giữa từ | 1 |
| đầu vào ở dạng NFD | 1 |
| db hoặc text tràn ra ngoài bốn ô cạnh node | 3 |
| validate: id trống | 1 |
| validate: id dành riêng của draw.io | 1 |
| validate: id trùng | 2 |
| validate: type lạ | 1 |
| validate: bảng không có lane | 1 |
| validate: lane hoặc edge có parent | 2 |
| validate: thiếu parent | 3 |
| validate: parent không phải lane | 1 |
| validate: key metadata lạ | 4 |
| validate: style lạ | 3 |
| validate: back sai giá trị | 1 |
| validate: edge thiếu from hoặc to | 1 |
| validate: from hoặc to treo | 4 |
| validate: from hoặc to sai loại | 1 |
| validate: db thiếu attach | 1 |
| validate: attach treo | 1 |
| validate: attach sai loại | 1 |
| validate: attach khác lane | 1 |
| validate: condition thiếu nhánh | 1 |
| validate: node không có cạnh ra | 16 |
| validate: start có cạnh vào | 5 |
| validate: end có cạnh ra | 2 |
| validate: nhánh condition không nhãn | 1 |
| validate: cạnh ra lệch chỗ | 4 |
| validate: quay ngược mà thiếu back=true | 3 |

## Merge

Merge không có dump trung gian: đầu vào là cả một file .drawio đã sửa tay, và
đầu ra so được trọn vẹn qua CLI. Mỗi kịch bản trong merge/ là một thư mục gồm
flow.md và flow.drawio, và cli/ có một bản ghi chạy build --mode merge trên nó.

- Kịch bản đặt tên theo hành vi do `tools/make_merge_cases.py` sinh: file do bản
  tham chiếu dựng lần đầu, rồi sửa như người dùng sửa trong draw.io.
- Kịch bản `fuzz-<seed>` do `tools/fuzz_merge.py --emit` sinh. Chúng là những
  seed đã giết một đột biến mà kịch bản viết tay bỏ lọt.

`tools/fuzz_merge.py --go <binary>` so hai bản trên hàng nghìn seed. Mỗi seed
lấy một case, sửa ngẫu nhiên cả file .drawio lẫn bảng, rồi so từng byte những
gì in ra và mọi file trong thư mục.

Cell tự vẽ được chép nguyên sang file mới rồi ghi lại qua ElementTree, nên bản
Go có `internal/etree` đọc ghi XML đúng quy tắc của ElementTree, chốt bằng
`etree-vectors.json`.


## Hành vi kỳ quặc được port nguyên

Việc của bản port là khớp, không phải sửa. Những chỗ dưới đây là hành vi của
bản tham chiếu mà golden cố tình mang theo. Sửa chúng là một thay đổi hành vi có
chủ đích, làm sau khi bản Go khớp 100%, và mỗi cái sẽ làm golden đổi.

**Id trùng sinh ra lỗi "cạnh ra không nằm liền sau" giả.** Tra cứu theo id lấy
dòng đầu tiên mang id đó, nhưng vị trí của id lại lấy dòng cuối cùng. Khi một id
xuất hiện hai lần, luật kiểm cạnh liền sau đi tìm cạnh ra phía sau dòng cuối,
không thấy, rồi báo lỗi cho dòng đầu dù cạnh ra của nó nằm đúng chỗ. Người dùng
vẫn thấy lỗi id trùng thật, nhưng kèm một lỗi thừa gây nhiễu. Chốt bởi case
`992-invalid-refs-order`.

**db tràn ô có thể đè lên đường của mũi tên ngang.** Khi bốn ô sát node đã kín,
đường lùi ra xa chỉ tránh ô đã có phần tử, không tránh khoảng mà một mũi tên
ngang chạy qua. db rơi vào đúng khoảng đó, và pha đi dây phải chuyển mũi tên từ
kiểu B thẳng ngang sang kiểu D đi vòng. Không cắt node nào nên tự kiểm vẫn báo
0 lỗi; cái mất là một mũi tên lẽ ra thẳng. Chốt bởi case
`48-place-attach-overflow-span`.

**Tên lane nhiều dòng làm lane rộng quá mức.** Bề rộng header được đo trên tên
lane đã nối các dòng bằng ký tự xuống dòng, như thể đó là một dòng. Ký tự xuống
dòng không có trong bảng số đo nên còn được tính thêm bề rộng một glyph
`.notdef`, 12 điểm ảnh. Khi vẽ ra thì tên đó nằm trên nhiều dòng ngắn, nên lane
rộng hơn mức cần. "Lane A" xuống dòng "dòng hai" cho lane 135 điểm ảnh thay vì
120. Chốt bởi case `51-geom-lane-header-multiline`.

## Thêm case

1. Viết bảng vào cases/, đặt tên theo nhánh nó chốt, không theo nội dung nghiệp
   vụ. `25-tracks-parallel.md`, không phải `25-luong-dat-hang.md`.
2. Giữ bảng nhỏ nhất có thể mà vẫn ép được nhánh cần chốt.
3. Chạy tools/coverage.py để xác nhận nhánh đó thật sự tăng. Một case được thêm
   vì nghĩ rằng nó chạm tới nhánh nào đó, mà số không đổi, là một case vô ích.
4. Sinh lại golden rồi đọc diff.
