# Từ vựng của flowcast

Tài liệu này định nghĩa các từ dùng xuyên suốt code, test và tài liệu thiết kế.
Một khái niệm chỉ có một tên. Thấy tên khác trong code nghĩa là code cần sửa,
không phải tài liệu cần thêm từ đồng nghĩa.

## Miền: bảng và sơ đồ

**Flow Table** - một sơ đồ luồng viết thành bảng 5 cột id, type, parent, content,
metadata. Đặc tả đầy đủ ở flow-table-format.md. Số cột là một cam kết tương
thích, không phải một chi tiết cài đặt.

**Source** - đầu vào chưa phân tích: bytes, tên hiển thị, gợi ý định dạng. CLI
dựng Source từ đường dẫn, web dựng từ upload. Core chỉ nhận Source, không bao
giờ nhận đường dẫn.

**Table** - kết quả phân tích một Source: tiêu đề, danh sách Row, danh sách
Issue. Mọi định dạng đầu vào đều quy về đây, kể cả mermaid.

**Row** - một dòng bảng đã tách ô, chưa diễn giải theo ngữ nghĩa sơ đồ.

**kind** - kiểu sơ đồ. Kind sở hữu ba thứ: lược đồ metadata, luật kiểm tra, và
thuật toán xếp hình. Hiện có một kind duy nhất, activity-swimlane, và flowchart
là chính kind đó với lane trở thành tùy chọn.

**lược đồ metadata** - khai báo của một kind về các key hợp lệ trong cột
metadata: theo từng type, key nào bắt buộc, key nào là tham chiếu tới id khác,
miền giá trị của từng key. Vì cột metadata chở cả tham chiếu bắt buộc (from, to,
attach), lược đồ này là nơi duy nhất giữ tính tường minh đó.

**lane** - làn chứa phần tử. Một sơ đồ có thể không có lane nào, khi đó nó là
flowchart. Lane là tùy chọn, không phải khái niệm đặc quyền.

**nhánh chính** - trong các cạnh ra của một node, nhánh giữ nguyên cột của node
đó. Các nhánh còn lại là nhánh phụ và dạt sang hai bên. Flow Table để người viết
chỉ định nhánh chính bằng thứ tự dòng; đầu vào không có quy ước đó thì nhánh
chính được suy ra theo độ sâu đường đi tới node kết thúc.

## Trục

**flow** - trục luồng đi tới. Trong sơ đồ TD đây là trục dọc, trong LR là trục
ngang.

**cross** - trục rẽ nhánh, vuông góc với flow. Cột trong lane và thứ tự lane đều
đo trên trục này.

Thuật toán xếp chỗ và đi dây chỉ nói bằng flow và cross. Chỉ một module duy nhất
được phép ánh xạ cặp đó sang x và y trên màn hình. Nhờ vậy LR, BT, RL là các
biến thể của cùng một ánh xạ chứ không phải các nhánh code riêng.

**máng (gutter)** - khoảng trống giữa hai vị trí cross liền nhau, nơi các đoạn
dây chạy dọc theo flow.

**kênh (channel)** - khoảng trống giữa hai bước flow liền nhau, nơi các đoạn dây
chạy ngang theo cross.

**track** - một làn dây bên trong một máng hoặc một kênh. Nhiều cạnh chia nhau
một máng bằng cách gán track khác nhau; các cạnh cùng đích dùng chung track để
gộp thành một đường.

## Kết quả và các lớp bọc ngoài

**LayoutResult** - dữ liệu thuần mô tả sơ đồ đã xếp xong: kích thước pool, các
lane, các phần tử kèm toạ độ và hình nguyên thủy, các cạnh kèm điểm gấp và vị
trí nhãn. Nó nói bằng hình nguyên thủy (chữ nhật, hình thoi, elip, elip đôi,
elip nét đứt, trụ, ghi chú) chứ không bằng loại ngữ nghĩa, để writer không phải
biết về kind.

**khai báo hình học** - với mỗi loại phần tử, hình nguyên thủy và luật nối dây
của nó, như chỉ nhận dây vào ở đỉnh. Engine và writer chỉ đọc khai báo này,
không so tên loại.

**writer** - hàm biến LayoutResult thành văn bản một định dạng đích. Writer chỉ
đọc LayoutResult, không gọi thuật toán.

**renderer** - thứ biến văn bản đích thành ảnh. Renderer gọi tiến trình ngoài
nên nằm ngoài core.

**Overrides** - các chỉnh sửa tay trích từ một file đích đã có, khoá theo id: vị
trí, kích thước, điểm gấp, cùng các cell người dùng tự vẽ giữ nguyên dạng đóng.
Đọc file đích là việc của adapter định dạng đó; áp Overrides lên LayoutResult là
việc chung của core.

**merge** - sinh lại sơ đồ mà giữ Overrides. Sau khi áp Overrides, LayoutResult
vẫn phải đi qua check().

**check** - tự kiểm hình học trên LayoutResult: dây cắt node, hai dây chồng
nhau, node chồng nhau. Vì nó chạy trên LayoutResult nên mọi kind và mọi đường
sinh, kể cả merge, đều được kiểm.

**verify** - đối chiếu ảnh do renderer vẽ ra với toạ độ đã tính. Khác check ở
chỗ nó cần tiến trình ngoài, nên là công cụ kiểm thử chứ không phải một bước của
build.

## Cấu hình và lỗi

**CoreConfig** - cấu hình không phụ thuộc kind: font, giới hạn tài nguyên, mức
nghiêm ngặt.

**cấu hình kind** - tham số xếp hình do kind khai báo, kèm mặc định và miền giá
trị. CLI sinh cờ và web sinh form từ cùng một khai báo đó, nên hai bên không thể
lệch nhau.

**Issue** - một phát hiện trả về cho caller, mang mã máy ổn định, mức độ, vị trí
có cấu trúc và thông điệp cho người đọc. Core không in Issue ra, caller tự
trình bày.

**BuildResult** - thứ build() trả về: văn bản đích, danh sách Issue, báo cáo
merge nếu có, và số liệu. Không chứa mã thoát và không chứa chuỗi đã định dạng
sẵn để in.
