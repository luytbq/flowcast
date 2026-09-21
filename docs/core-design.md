# Thiết kế core

Tài liệu này mô tả hình dạng core sau khi bóc tách từ script hiện tại. Từ vựng
dùng ở đây định nghĩa trong CONTEXT.md. Các quyết định có lý do chịu lực nằm
trong docs/adr.

Mục tiêu: một core phục vụ được hai caller rất khác nhau, một CLI trên máy có
filesystem và một request HTTP chỉ có bytes, mà không bên nào phải viết lại
chính sách của bên kia.

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
flowtable/                 core thuần, không I/O
  __init__.py              build, Source, CoreConfig, Issue, BuildResult
  issues.py                Issue, Location, danh mục mã lỗi
  model.py                 Row, Table
  config.py                CoreConfig, Field, hồ sơ cli và web
  sources/
    __init__.py            registry, đoán định dạng
    markdown.py  csv.py  xlsx.py  mermaid.py
  schema.py                lược đồ metadata
  validate.py              kiểm theo lược đồ, cộng luật riêng của kind
  text.py                  bảng độ rộng glyph, đo và ngắt dòng
  axis.py                  ánh xạ flow, cross sang x, y
  layout/
    place.py  route.py  tracks.py  geometry.py  labels.py
    result.py              LayoutResult và các kiểu con
    check.py               tự kiểm hình học
  merge.py                 Overrides, apply_overrides, MergeReport
  writers/
    __init__.py            registry
    drawio.py              sinh xml, và đọc Overrides từ xml
flowtable_cli/             cờ vào, in ra, mã thoát
flowtable_render/          adapter drawio CLI: png, svg, verify
```

Chia file theo pha đã có sẵn trong Layout.run, không phải theo số dòng.

## 3. Interface công khai

Core phơi ra đúng một hàm. Mọi thứ khác là kiểu dữ liệu.

```python
def build(
    source: Source,
    *,
    core: CoreConfig = CoreConfig(),
    kind: str = "auto",
    kind_config: Mapping[str, int] | None = None,
    writer: str = "drawio",
    overrides: Overrides | None = None,
) -> BuildResult
```

```python
@dataclass(frozen=True)
class Source:
    data: bytes
    name: str = ""                       # tiêu đề dự phòng, không dùng để mở file
    format: str | None = None            # None nghĩa là tự đoán
    options: Mapping[str, str] = MappingProxyType({})
```

options là chỗ chứa tham số riêng của từng định dạng: sheet cho xlsx, delimiter
và encoding cho csv, direction cho mermaid. Mỗi source adapter khai báo các
option nó nhận, cùng miền giá trị, theo đúng cơ chế mà cấu hình kind dùng ở mục
6. Nhờ vậy CLI và web sinh giao diện từ cùng một khai báo.

```python
@dataclass(frozen=True)
class BuildResult:
    text: str | None                     # None khi có lỗi chặn
    title: str
    issues: tuple[Issue, ...]
    layout: LayoutResult | None
    merge_report: MergeReport | None
    stats: Stats                         # số lane, phần tử, cạnh, kích thước pool
```

BuildResult không mang mã thoát và không mang chuỗi đã định dạng sẵn để in. CLI
tự ánh xạ sang mã thoát, web tự ánh xạ sang HTTP status. Đây là chỗ duy nhất
hai front-end được phép khác nhau về chính sách.

Nói rõ về tham số kind: hôm nay chỉ có một engine, và flowchart là engine đó với
lane trở thành tùy chọn. Tham số này chọn một bộ mặc định, chưa chọn một thuật
toán. Nó tồn tại để khi có engine thứ hai thật sự thì chữ ký không phải đổi.

## 4. Error model

```python
@dataclass(frozen=True)
class Location:
    kind: str                  # "table" hoặc "text"
    row: int | None = None     # table: số dòng trong bảng
    column: str | None = None  # table: tên cột
    sheet: str | None = None
    line: int | None = None    # text: dòng trong nguồn, dùng cho mermaid
    id: str | None = None      # id phần tử liên quan, nếu xác định được

@dataclass(frozen=True)
class Issue:
    code: str                  # mã ổn định, ví dụ "ref.dangling"
    level: str                 # "error" hoặc "warning"
    loc: Location
    msg: str                   # tiếng Việt, dành cho người đọc
    params: Mapping[str, str] = MappingProxyType({})
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

```python
@dataclass(frozen=True)
class KeySpec:
    name: str
    required: bool = False
    is_ref: bool = False          # giá trị là id của một dòng khác
    values: tuple[str, ...] | None = None   # None nghĩa là chuỗi tự do
    multi: bool = False           # nhiều giá trị, ngăn bằng dấu phẩy

# mỗi type khai báo các key của nó
SWIMLANE_SCHEMA = {
    "edge": (KeySpec("from", required=True, is_ref=True),
             KeySpec("to", required=True, is_ref=True),
             KeySpec("back", values=("true",)),
             KeySpec("style", values=("highlight", "dashed", "bold", "noarrow"), multi=True)),
    "db":   (KeySpec("attach", required=True, is_ref=True),
             KeySpec("style", values=("highlight",), multi=True)),
    ...
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

```python
@dataclass(frozen=True)
class CoreConfig:
    strict: bool = False              # web bật, CLI tắt
    max_source_bytes: int = 4 << 20
    max_rows: int = 2000
    max_edges: int = 3000
    max_cell_chars: int = 2000
    deadline_ms: int | None = None
```

Cấu hình kind không phải một dataclass cố định mà là một khai báo:

```python
@dataclass(frozen=True)
class Field:
    name: str
    default: int
    lo: int
    hi: int
    help: str
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

```python
@dataclass(frozen=True)
class Axis:
    flow: str      # "down", "up", "right", "left"

    def to_xy(self, flow_pos: float, cross_pos: float) -> tuple[float, float]: ...
    def extent(self, w: float, h: float) -> tuple[float, float]:
        """Trả về (bề dài theo flow, bề dài theo cross) của một hộp w x h."""
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

```python
SHAPES = ("rect", "round", "diamond", "ellipse", "cylinder", "note")

@dataclass(frozen=True)
class PlacedItem:
    id: str
    shape: str                    # một trong SHAPES
    roles: tuple[str, ...]        # "highlight", ...
    lane: int | None
    lines: tuple[str, ...]
    x: float; y: float; w: float; h: float
    order: int
    semantic: str                 # "task", "condition", ... dành cho công cụ

@dataclass(frozen=True)
class PlacedEdge:
    id: str
    src: str; dst: str
    lines: tuple[str, ...]
    points: tuple[tuple[float, float], ...]
    exit_frac: tuple[float, float] | None
    entry_frac: tuple[float, float] | None
    label_t: float; label_off: tuple[float, float]
    roles: tuple[str, ...]        # "dashed", "bold", "noarrow", "highlight"
    order: int
    pinned: bool                  # đến từ Overrides, đích tự đi dây

@dataclass(frozen=True)
class LayoutResult:
    axis: Axis
    pool: tuple[float, float]
    origin: tuple[float, float]
    lanes: tuple[PlacedLane, ...]
    items: tuple[PlacedItem, ...]
    edges: tuple[PlacedEdge, ...]
    foreign: tuple[object, ...]   # cell người dùng tự vẽ, giữ nguyên dạng đóng
    warnings: tuple[Issue, ...]
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

```python
# trong writers/drawio.py
def read_overrides(data: bytes) -> tuple[Overrides, tuple[Issue, ...]]: ...

# trong merge.py
def apply_overrides(lay: LayoutResult, ov: Overrides) -> tuple[LayoutResult, MergeReport]: ...
```

```python
@dataclass(frozen=True)
class Overrides:
    items: Mapping[str, ItemOverride]    # tâm, và kích thước nếu người dùng đã chỉnh
    edges: Mapping[str, EdgeOverride]    # waypoint, điểm neo
    lanes: Mapping[str, LaneOverride]    # vị trí và bề rộng
    foreign: tuple[object, ...]          # cell tự vẽ, dạng đóng
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
phiên bản tool thì ra cùng một chuỗi byte. Một golden test cho mỗi fixture canh
điều này.

Hôm nay tool dò Verdana.ttf ngoài hệ thống, thiếu thì ước lượng độ rộng và chỉ
cảnh báo. Với một service, đó là hai máy cho hai kết quả khác nhau từ cùng một
đầu vào, tức là phá cam kết.

Cách xử lý đề nghị: **đóng gói bảng độ rộng glyph, không đóng gói file font.**
Tool chỉ cần advance width theo từng codepoint để đo và ngắt dòng, nên một bảng
vài KB cho Latin và tiếng Việt là đủ. Bảng này sinh một lần từ Verdana bằng một
script trong tools, và được commit.

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

## 14. Lộ trình

Mỗi bước giữ toàn bộ test xanh và có thể dừng lại ở đó mà vẫn có ích.

| bước | việc | vì sao ở vị trí này |
|---|---|---|
| 0 | golden test: mỗi fixture chốt đúng chuỗi byte đầu ra | lưới an toàn cho mọi bước sau, làm trước tiên |
| 1 | tách package, thuần di chuyển, không đổi logic | test chỉ đổi dòng import |
| 2 | Source và đọc từ bytes; load theo đường dẫn lùi về CLI | mở đường cho mọi thứ còn lại |
| 3 | seam build; cmd_build teo lại còn cờ và in | sau bước này core đã cắm được vào web |
| 4 | Issue có mã và Location; CLI tự định dạng | web phân loại được lỗi |
| 5 | tách Config, khai báo Field, sinh cờ và form từ một nguồn | chặn được input không tin cậy |
| 6 | LayoutResult thành kiểu đóng băng; writer chỉ đọc nó | tách thuật toán khỏi kết quả |
| 7 | Overrides; merge trả kết quả mới và chạy lại check | vá lỗ hổng merge bỏ qua tự kiểm |
| 8 | bảng độ rộng glyph, giới hạn tài nguyên, renderer ra ngoài | đủ điều kiện chạy dịch vụ |
| 9 | lane thành tùy chọn | mở đường cho flowchart |
| 10 | bộ sơ đồ đo, rồi heuristic độ sâu đường đi | phần tạo ra giá trị thật |
| 11 | source adapter mermaid, hướng khác TD quy về TD kèm cảnh báo | bản dùng được đầu tiên cho mermaid |
| 12 | module axis, LR và BT và RL | khối lớn nhất, tách riêng làm mốc |

Bước 11 ra trước bước 12 có chủ ý: mermaid quy về TD đã dùng được ngay, còn lật
trục là khối lớn nhất trong cả lộ trình. Quyết định hỗ trợ LR thật không đổi,
chỉ xếp sau bản dùng được đầu tiên.
