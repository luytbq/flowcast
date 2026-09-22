# Thiết kế core

Tài liệu này mô tả hình dạng core sau khi bóc tách từ script hiện tại. Từ vựng
dùng ở đây định nghĩa trong CONTEXT.md. Các quyết định có lý do chịu lực nằm
trong docs/adr.

Mục tiêu: một core phục vụ được hai caller rất khác nhau, một CLI trên máy có
filesystem và một request HTTP chỉ có bytes, mà không bên nào phải viết lại
chính sách của bên kia.

## 0. Ngôn ngữ và ràng buộc kỹ thuật

Go, module `github.com/luytbq/flowcast`, mốc tương thích là phiên bản Go ổn định
hiện hành.

Core **chỉ dùng stdlib, trừ một ngoại lệ**. `archive/zip` và `encoding/xml` đủ
để đọc xlsx; số đo font đọc từ `data/verdana.json` nên không cần thư viện font.
Ngoại lệ duy nhất là `golang.org/x/text/unicode/norm` cho chuẩn hóa NFC, vì
stdlib không có và viết lại chuẩn hóa Unicode cho đúng là bãi mìn; lý do đầy đủ
trong ADR-0005. Các package cli, web và render được dùng thư viện ngoài.

Đây là một bản port, không phải một bản viết mới. Bản Python trong `reference/`
ở lại repo và là máy sinh đáp án: mọi module Go phải tái tạo đúng từng byte đầu
ra trong `conformance/golden/` trước khi được coi là xong. Lý do và cách dùng bộ
đó nằm trong `conformance/README.md`.

WASM chưa nằm trong phạm vi. Thiết kế không đóng cửa đó: core không chạm I/O nên
build cho `js/wasm` về sau là việc thêm một target, không phải thiết kế lại.

## 1. Luật định hình mọi thứ

Core không chạm filesystem, không gọi tiến trình ngoài, không đọc biến môi
trường, không in ra stdout, không bắt tín hiệu. Vào là bytes, ra là văn bản đích
cộng dữ liệu chẩn đoán.

Luật này loại ba thứ đang nằm trong ruột hôm nay: load mở file, export gọi
drawio CLI, cmd_build in báo cáo rồi trả mã thoát. Cả ba đi ra ngoài core.

Hệ quả kiểm thử: mọi test của core chạy được không cần file tạm, và mọi lỗi
hoặc là một Issue trả về, hoặc là một FlowTableError có kiểu. Không có đường
thứ ba.

## 2. Bố cục package

```
flowcast/
  reference/               bản Python, máy sinh đáp án, không build vào binary
  conformance/             bộ đối chiếu: cases/ và golden/
  data/verdana.json        bảng độ rộng glyph, dùng chung hai bản
  go.mod
  build.go                 hàm Build, điểm vào duy nhất của core
  config.go                CoreConfig, Field, hồ sơ cli và web
  model/                   Row, Table, Issue, Location, Error
  num/                     Fmt và Rnd, hai quy tắc chuẩn hóa số
  source/                  Source, đoán định dạng, các adapter
    markdown.go  csv.go  xlsx.go  mermaid.go
  schema/                  lược đồ metadata
  validate/
  text/                    đọc bảng số đo, đo và ngắt dòng
  axis/                    ánh xạ flow, cross sang x, y
  layout/
    place.go  route.go  tracks.go  geometry.go  labels.go
    result.go              LayoutResult và các kiểu con
    check.go
  merge/                   Overrides, ApplyOverrides, MergeReport
  writer/
    drawio/                sinh xml, và đọc Overrides từ xml
  cmd/flowcast/            CLI
  cmd/flowcastd/           HTTP service
  render/                  adapter drawio CLI: png, svg, verify
```

model và num nằm riêng khỏi gói gốc vì mọi adapter đều cần chúng, còn gói gốc
lại cần các adapter; gói gốc phơi lại bằng bí danh kiểu. Chia file theo pha đã
có sẵn trong Layout.run của bản tham chiếu, không phải theo số dòng. Giữ đúng ranh giới đó làm cho việc đối chiếu từng module với bản
Python trở nên khả thi.

## 3. Interface công khai

Core phơi ra đúng một hàm. Mọi thứ khác là kiểu dữ liệu.

```go
func Build(src Source, opt Options) (Result, error)
```

```go
type Source struct {
    Data    []byte
    Name    string            // tiêu đề dự phòng, không dùng để mở file
    Format  string            // rỗng nghĩa là tự đoán
    Options map[string]string // tham số riêng của từng định dạng
}

type Options struct {
    Core      CoreConfig
    Kind      string         // rỗng nghĩa là tự chọn theo bảng
    KindConf  map[string]int
    Writer    string         // rỗng nghĩa là "drawio"
    Overrides *Overrides     // nil nghĩa là sinh mới
}
```

options là chỗ chứa tham số riêng của từng định dạng: sheet cho xlsx, delimiter
và encoding cho csv, direction cho mermaid. Mỗi source adapter khai báo các
option nó nhận, cùng miền giá trị, theo đúng cơ chế mà cấu hình kind dùng ở mục
6. Nhờ vậy CLI và web sinh giao diện từ cùng một khai báo.

```go
type Result struct {
    Text        string          // rỗng khi có lỗi chặn
    Title       string
    Issues      []Issue
    Layout      *LayoutResult
    MergeReport *MergeReport
    Stats       Stats           // số lane, phần tử, cạnh, kích thước pool
}
```

BuildResult không mang mã thoát và không mang chuỗi đã định dạng sẵn để in. CLI
tự ánh xạ sang mã thoát, web tự ánh xạ sang HTTP status. Đây là chỗ duy nhất
hai front-end được phép khác nhau về chính sách.

Nói rõ về tham số kind: hôm nay chỉ có một engine, và flowchart là engine đó với
lane trở thành tùy chọn. Tham số này chọn một bộ mặc định, chưa chọn một thuật
toán. Nó tồn tại để khi có engine thứ hai thật sự thì chữ ký không phải đổi.

## 4. Error model

```go
type Location struct {
    Kind   string // "table" hoặc "text"
    Row    int    // table: số dòng trong bảng, 0 nghĩa là không xác định
    Column string // table: tên cột
    Sheet  string
    Line   int    // text: dòng trong nguồn, dùng cho mermaid
    ID     string // id phần tử liên quan, nếu xác định được
}

type Issue struct {
    Code   string // mã ổn định, ví dụ "ref.dangling"
    Level  string // "error" hoặc "warning"
    Loc    Location
    Msg    string // tiếng Việt, dành cho người đọc
    Params map[string]string
}
```

Mã lỗi đặt theo miền, chấm, rồi triệu chứng. Chúng là một phần của interface
công khai: đổi một mã là một thay đổi phá vỡ tương thích.

| tiền tố | miền |
|---|---|
| source. | đọc và giải mã đầu vào |
| table. | cấu trúc bảng, header, ô |
| ref. | tham chiếu id: from, to, attach, parent |
| schema. | key metadata thiếu, lạ, hoặc sai miền giá trị |
| order. | thứ tự dòng theo đặc tả |
| mermaid. | cú pháp mermaid không có chỗ chứa |
| layout. | tự kiểm hình học |
| limit. | vượt giới hạn tài nguyên |

Thông điệp tiếng Việt nằm trong core, cạnh chỗ phát hiện lỗi. Muốn đa ngôn ngữ
về sau thì đã có mã để tra, không phải sửa lại chỗ phát hiện.

FlowTableError chỉ dùng cho thứ khiến build không thể tiếp tục và không quy được
về một dòng cụ thể: không đoán được định dạng, file hỏng, vượt giới hạn. Mọi thứ
khác là Issue.

## 5. Table và lược đồ metadata

Bảng giữ đúng 5 cột id, type, parent, content, metadata (ADR-0001). Vì cột
metadata chở cả tham chiếu bắt buộc là from, to, attach, tính tường minh phải
nằm ở nơi khác: một lược đồ do kind khai báo.

```go
type KeySpec struct {
    Name     string
    Required bool
    IsRef    bool     // giá trị là id của một dòng khác
    Values   []string // nil nghĩa là chuỗi tự do
    Multi    bool     // nhiều giá trị, ngăn bằng dấu phẩy
}

// mỗi type khai báo các key của nó
var SwimlaneSchema = map[string][]KeySpec{
    "edge": {
        {Name: "from", Required: true, IsRef: true},
        {Name: "to", Required: true, IsRef: true},
        {Name: "back", Values: []string{"true"}},
        {Name: "style", Values: []string{"highlight", "dashed", "bold", "noarrow"}, Multi: true},
    },
    "db": {
        {Name: "attach", Required: true, IsRef: true},
        {Name: "style", Values: []string{"highlight"}, Multi: true},
    },
}
```

validate trở thành hai phần: một vòng chung chạy trên lược đồ, bắt key thiếu,
key lạ, giá trị ngoài miền, và tham chiếu treo; cộng một nhúm luật riêng của
kind mà lược đồ không diễn đạt được, như condition phải có từ hai cạnh ra và
cạnh ra phải nằm liền sau node.

parent được định nghĩa lại là **cây chứa đựng đơn cha**, không còn là "id của
một lane". Định nghĩa này phủ được lane, subgraph của mermaid, và về sau là
trạng thái tổ hợp hay subprocess, mà không tốn gì thêm hôm nay.

## 6. Config hai tầng

Cả 18 trường Config hiện tại đều là tham số xếp hình của swimlane. Không trường
nào thuộc về core. Vì vậy tách đôi.

```go
type CoreConfig struct {
    Strict         bool          // web bật, CLI tắt
    MaxSourceBytes int
    MaxRows        int
    MaxEdges       int
    MaxCellChars   int
    Deadline       time.Duration // 0 nghĩa là không chặn
}
```

Cấu hình kind không phải một dataclass cố định mà là một khai báo:

```go
type Field struct {
    Name    string
    Default int
    Lo, Hi  int
    Help    string
}
```

CLI sinh cờ từ danh sách Field, web sinh form và validate từ cùng danh sách đó.
Hai bên không thể lệch nhau, vì chỉ có một nguồn sự thật. Giá trị ngoài miền
lo..hi là một Issue mã schema.out_of_range, không phải một exception.

Tên trường phải trung lập theo trục. min_lane_w trong sơ đồ LR là độ dày của
một băng ngang, nên tên hiện tại sẽ sai nghĩa và cần đổi khi tách.

Hồ sơ mặc định: hồ sơ cli nới giới hạn, hồ sơ web siết. Hồ sơ là dữ liệu, không
phải nhánh if trong core.

## 7. Trục flow và cross

place và route hôm nay đã trung lập về trục: chúng chỉ làm việc trên lưới lane,
col, row, và hàm face chỉ trả về L hoặc R, tức dấu trên trục rẽ nhánh. Pixel chỉ
xuất hiện ở compute_geometry, track_x, track_y, resolve_paths và to_drawio.

Vì vậy không sửa rải rác. Đặt tên hai trục rồi dồn toàn bộ ánh xạ vào một module:

```go
type Axis struct {
    Flow string // "down", "up", "right", "left"
}

func (a Axis) ToXY(flow, cross float64) (x, y float64)

// Extent trả về bề dài theo flow và theo cross của một hộp w x h.
func (a Axis) Extent(w, h float64) (alongFlow, alongCross float64)
```

Sau đó TD, LR, BT, RL là bốn giá trị của cùng một ánh xạ, không phải bốn nhánh
code.

Một điều quan trọng dễ hiểu sai: **hộp node không lật theo trục**. Chữ vẫn đọc
từ trái sang phải, bề rộng hộp vẫn bị chặn bởi task_max_w, chiều cao vẫn mọc
theo số dòng. Chỉ có lưới lật. Đó là lý do LR rẻ. Hàm extent ở trên là nơi duy
nhất biết điều này: với flow đi xuống, bề dài theo flow là h; với flow đi sang
phải, là w.

Máng và kênh đổi vai cho nhau theo trục, nên chúng nên được gọi theo trục chứ
không theo hướng màn hình: máng chạy dọc theo flow, kênh chạy dọc theo cross.

LayoutResult mang theo Axis, vì writer cần biết lane nằm dọc hay nằm ngang để
đặt cờ horizontal của swimlane trong draw.io.

## 8. LayoutResult

Tách Layout-thuật-toán khỏi Layout-kết-quả. Bằng chứng là việc này đã gần xong:
to_drawio chỉ đọc dữ liệu trên đối tượng Layout, không gọi một method nào của
nó. Kiểu kết quả đã tồn tại ngầm, chỉ chưa được đặt tên và đóng băng.

```go
var Shapes = []string{"rect", "round", "diamond", "ellipse", "cylinder", "note"}

type PlacedItem struct {
    ID         string
    Shape      string   // một trong Shapes
    Roles      []string // "highlight", ...
    Lane       int      // -1 nghĩa là không thuộc lane nào
    Lines      []string
    X, Y, W, H float64
    Order      int
    Semantic   string // "task", "condition", ... writer không được đọc trường này
}

type PlacedEdge struct {
    ID        string
    Src, Dst  string
    Lines     []string
    Points    [][2]float64
    ExitFrac  *[2]float64
    EntryFrac *[2]float64
    LabelT    float64
    LabelOff  [2]float64
    Roles     []string // "dashed", "bold", "noarrow", "highlight"
    Order     int
    Pinned    bool // đến từ Overrides, đích tự đi dây
}

type LayoutResult struct {
    Axis     Axis
    Pool     [2]float64
    Origin   [2]float64
    Lanes    []PlacedLane
    Items    []PlacedItem
    Edges    []PlacedEdge
    Foreign  []any // cell người dùng tự vẽ, giữ nguyên dạng đóng
    Warnings []Issue
}
```

Điểm mấu chốt: LayoutResult nói bằng **hình nguyên thủy**, không bằng loại ngữ
nghĩa. Kind quyết định task là round và condition là diamond. Writer chỉ cần
biết sáu hình. Nếu writer phải biết loại ngữ nghĩa thì mỗi writer phải biết mọi
kind, và số việc thành tích của hai con số thay vì tổng của chúng.

Trường semantic vẫn được mang theo cho công cụ và cho gỡ lỗi, nhưng writer không
được đọc nó. Đây là một quy ước, và một test nên canh nó.

check chạy trên LayoutResult. Nhờ vậy mọi kind và mọi đường sinh, kể cả merge,
đều được tự kiểm mà không viết lại gì.

## 9. Overrides và merge

Đọc mxCell là việc của adapter drawio. Áp vị trí cũ theo id là việc chung. Tách
theo đúng ranh giới đó.

```go
// trong writer/drawio
func ReadOverrides(data []byte) (Overrides, []Issue, error)

// trong merge
func ApplyOverrides(lay LayoutResult, ov Overrides) (LayoutResult, MergeReport)
```

```go
type Overrides struct {
    Items   map[string]ItemOverride // tâm, và kích thước nếu người dùng đã chỉnh
    Edges   map[string]EdgeOverride // waypoint, điểm neo
    Lanes   map[string]LaneOverride // vị trí và bề rộng
    Foreign []any                   // cell tự vẽ, dạng đóng
}
```

apply_overrides trả về LayoutResult mới thay vì sửa tại chỗ. Vì vậy build chạy
được check sau merge, vá đúng lỗ hổng hôm nay là merge đi vòng qua lớp tự kiểm.
Phát hiện sau merge hạ xuống mức warning, vì đường dây lúc đó một phần do người
dùng quyết định, nhưng chúng phải được báo chứ không bị vứt.

MergeReport giữ nguyên nội dung hôm nay: phần tử mới, phần tử phải dịch, cạnh
nối tới phần tử mới, cạnh đã sửa tay, cell tự vẽ mất đầu nối.

## 10. Giới hạn tài nguyên

Chặn trên số dòng và số cạnh là biện pháp chính, vì chúng chặn luôn thời gian
chạy: topo là O(V+E), place duyệt lưới, gán track là tô màu khoảng trên số cạnh.
deadline_ms là biện pháp phụ, kiểm ở ranh giới giữa các pha, không cắt ngang một
thuật toán.

Vượt giới hạn là FlowTableError mã limit.*, không phải Issue, vì không có kết
quả bộ phận nào đáng trả về.

Hồ sơ web siết chặt hơn hồ sơ cli. Con số cụ thể là dữ liệu trong config.py, để
đổi được mà không sửa code.

## 11. Font và tính tất định

Cam kết: cùng một Source, cùng một CoreConfig, cùng một cấu hình kind, cùng một
phiên bản tool thì ra cùng một chuỗi byte. Bộ đối chiếu trong `conformance/`
canh điều này.

Đã làm, ở bước 0: **đóng gói bảng độ rộng glyph, không đóng gói file font.**
Tool chỉ cần advance width theo từng codepoint để đo và ngắt dòng, nên bảng
`data/verdana.json` với 738 codepoint, khoảng 10KB, là đủ. Bảng sinh bằng
`tools/extract_metrics.py` và đã commit. Đã xác nhận nó cho ra đầu ra không lệch
một byte so với đọc thẳng file font.

Bản Go đọc đúng bảng này. Nhờ vậy binary phân phối đi không mang theo font và
không phụ thuộc máy đích có cài Verdana hay không, và bộ đối chiếu tái tạo được
trên mọi máy.

Ghi rõ một rủi ro: Verdana là font thương mại của Microsoft, giấy phép phân phối
lại file font rất chặt. Phân phối lại một bảng số đo là chuyện khác hẳn với phân
phối lại font, nhưng nếu dịch vụ này công khai thì đây là điểm cần người hiểu
luật xem qua. Phương án dự phòng nếu không ổn: đổi fontFamily trong đầu ra sang
một font tự do và sinh bảng đo từ font đó.

Độc lập với chuyện giấy phép: ở chế độ strict, thiếu bảng đo là lỗi cứng chứ
không phải cảnh báo.

## 12. Đầu vào mermaid

mermaid là source adapter thứ tư, quy về cùng một Table như ba cái kia. Ánh xạ:

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
có tương đương trung thực trong style string của draw.io, và không tham chiếu
tới dòng khác. Theo biên đó, bốn kiểu mũi tên chỉ là ba giá trị mới của key
style đã có, không phải key mới. Màu đặt thẳng trên node không vào, vì key style
mang vai trò ngữ nghĩa chứ không mang màu; màu khớp bảng màu nhấn thì quy về
highlight, còn lại cảnh báo. icon, click, href không đổi bố cục nên không vào.
subgraph lồng nhau thì mô hình chịu được vì parent đã là cây, nhưng engine chưa
xếp được lane lồng lane, nên phiên bản đầu dẹp về subgraph ngoài cùng kèm cảnh
báo mã mermaid.nested_subgraph.

Mọi thứ bỏ qua đều là một Issue có mã và có số dòng trong nguồn, để web hiện
được danh sách đã bỏ qua những gì.

## 13. Flowchart và heuristic nhánh chính

flowchart không phải một kind mới. Nó là activity-swimlane với lane trở thành
tùy chọn. Cụ thể: cho phép bảng không có dòng lane nào, pool_header và
lane_header bằng 0, bỏ ràng buộc parent phải là id của một lane. Khóa lưới
gkey trả về cặp lane và col; với đúng một lane nó rút về col và toàn bộ thuật
toán chạy nguyên vẹn.

Nhưng có một chỗ hỏng thật. branch_drift đi dọc một nhánh tìm cạnh đầu tiên rời
khỏi lane, không thấy thì trả về 0. Trong sơ đồ một lane nó **luôn** trả về 0,
nên heuristic mạnh nhất của engine chết hoàn toàn và mọi nhánh rơi về luật dự
phòng là lần lượt sang phải rồi sang trái, tức đặt theo thứ tự chứ không theo
nội dung.

Lý do sâu hơn: đặc tả Flow Table đẩy quyết định này sang người viết bảng, ở luật
"đặt trước những nhánh kết thúc sớm để nhánh đi tiếp dài nhất nằm cuối", và
engine tin vào đó qua quy ước cạnh ra cuối cùng là nhánh chính. Mermaid không có
quy ước nào như vậy.

Vì vậy với sơ đồ không có lane, nhánh chính được suy ra theo **độ sâu đường đi**:
trong các cạnh ra của một node, nhánh có đường dài nhất tới một node kết thúc giữ
nguyên cột, các nhánh còn lại dạt sang hai bên. Đo độ sâu trên đồ thị đã bỏ cạnh
back, và dừng ở node hợp nhánh để không tính phần luồng chung vào độ sâu của một
nhánh riêng.

Với sơ đồ có lane, branch_drift vẫn được ưu tiên, vì hướng lane là tín hiệu mạnh
hơn. Độ sâu chỉ thay vào chỗ branch_drift trả về 0.

Đây là phần quyết định sản phẩm có hơn draw.io hay không, nên nó phải được đo
chứ không được cảm nhận: dựng một bộ sơ đồ mermaid thật trước khi viết heuristic,
và giữ số liệu check trên bộ đó qua từng thay đổi.

## 14. Lộ trình port

Lộ trình bóc tách tại chỗ trước đây không còn dùng được: đây là một bản port,
nên không có trạng thái trung gian nào mà cả hai bản cùng chạy trên cùng một
cây code.

### Cổng chặn

Mỗi module Go xong khi nó tái tạo đúng đầu ra của bản tham chiếu trên toàn bộ
`conformance/cases/`. So từng byte, không so bằng mắt.

Vấn đề: so từng byte chỉ làm được ở cuối chuỗi, khi đã có xml. Các module phía
trước cần điểm so của riêng chúng. Vì vậy việc đầu tiên là **thêm chế độ dump
trung gian vào bản tham chiếu**, và bản Go dump đúng cùng định dạng:

| dump | chốt module |
|---|---|
| dòng đã ngắt kèm bề rộng từng phần tử | text |
| Table sau khi parse, dạng JSON | source, model |
| danh sách Issue, dạng JSON | schema, validate |
| lưới lane, row, col của từng phần tử | layout/place |
| danh sách Seg kèm track | layout/route, layout/tracks |
| toạ độ cuối cùng, tức `to_json` đã có sẵn | layout/geometry, labels |

### Thứ tự

| bước | module | ghi chú |
|---|---|---|
| 1 | dump trung gian trong bản tham chiếu | xong |
| 2 | text | xong, cùng `num` và `layout.SizeItem` |
| 3 | model, source/markdown | xong |
| 4 | schema, validate | xong |
| 5 | layout/place | xong |
| 6 | layout/route, layout/tracks | 4 kiểu đi dây, tô màu khoảng |
| 7 | layout/geometry, layout/labels | |
| 8 | layout/check | |
| 9 | writer/drawio | **cổng thật**: 29 file .drawio khớp từng byte |
| 10 | build, cmd/flowcast | từ đây bản Go dùng được |
| 11 | source/csv, source/xlsx | cần bổ sung case đối chiếu trước |
| 12 | merge, Overrides | cần bổ sung case đối chiếu trước, xem lỗ hổng đã biết |
| 13 | render, giới hạn tài nguyên, cmd/flowcastd | đủ điều kiện chạy dịch vụ |

### Sau khi khớp

Chỉ khi bước 9 và 12 đã khớp thì mới làm phần mới, vì từ đây bản Python không
còn là đáp án nữa:

| bước | việc |
|---|---|
| 14 | lane thành tùy chọn |
| 15 | bộ sơ đồ đo, rồi heuristic độ sâu đường đi |
| 16 | source/mermaid, hướng khác TD quy về TD kèm cảnh báo |
| 17 | axis, LR và BT và RL |

Bước 16 ra trước bước 17 có chủ ý: mermaid quy về TD đã dùng được ngay, còn lật
trục là khối lớn nhất. Quyết định hỗ trợ LR thật không đổi, chỉ xếp sau bản dùng
được đầu tiên.

Từ bước 14 trở đi, `conformance/golden/` thôi là đáp án và thành bộ chống hồi
quy: nó phải đổi khi và chỉ khi một trong các bước đó cố ý đổi bố cục, và mỗi
lần sinh lại đều phải đọc diff.
