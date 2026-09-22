# Định dạng Flow Table

Flow Table là cách viết một activity diagram có swimlane thành một bảng markdown duy nhất. Người đọc đọc bảng từ trên xuống là đi theo luồng. Công cụ hoặc một phiên AI khác dựng lại được sơ đồ từ bảng mà không cần xem ảnh gốc.

## Bảng có những cột nào?

| Cột | Ý nghĩa |
|---|---|
| id | Định danh của dòng, không đổi sau khi đã gán |
| type | Loại phần tử, xem mục tiếp theo |
| parent | Id của lane chứa phần tử. Để trống với lane và edge |
| content | Chữ hiển thị: tên lane, chữ trong node, nhãn trên cạnh, nội dung ghi chú |
| metadata | Thông tin phụ, dạng key=value, các cặp cách nhau bằng dấu chấm phẩy, không cần nháy |

Trong content, xuống dòng viết là <br>, còn ký tự gạch đứng viết là \|, để không vỡ bảng markdown. Chữ trong content giữ nguyên văn như sơ đồ, kể cả lỗi chính tả.

## Có những loại phần tử nào?

Loại phần tử quyết định luôn hình vẽ, nên không có metadata riêng cho hình dạng.

| type | Hình vẽ | parent | Metadata dùng được |
|---|---|---|---|
| lane | Làn dọc | trống | không có |
| start | Điểm bắt đầu luồng | lane | style |
| end | Điểm kết thúc luồng | lane | style |
| task | Hộp bước xử lý | lane | style |
| condition | Hình thoi rẽ nhánh | lane | style |
| db | Hình trụ bảng dữ liệu, đứng cạnh node, không nối cạnh | lane | attach (bắt buộc), style |
| external | Hình elip, trỏ sang một luồng khác nằm ngoài sơ đồ | lane | style |
| text | Ghi chú, không có khung | lane | attach, style |
| edge | Mũi tên nối hai phần tử | trống | from, to (bắt buộc), style, back |

Ý nghĩa của từng key:

- **from, to:** id phần tử ở đầu và cuối mũi tên. Một cạnh nối được hai lane khác nhau, nên edge không thuộc lane nào.
- **attach:** id của node mà bảng dữ liệu hoặc ghi chú đứng cạnh.
- **style:** highlight nếu phần tử được tô màu nhấn trong sơ đồ. Riêng edge còn nhận thêm dashed là nét đứt, bold là nét đậm, và noarrow là không có đầu mũi tên. Nhiều giá trị cách nhau bằng dấu phẩy, ví dụ style=dashed,noarrow.
- **back:** ghi back=true cho cạnh quay ngược về một phần tử đã đi qua, tức cạnh tạo vòng lặp.

## Đặt id thế nào?

- **Lane:** mã ngắn viết hoa, ví dụ USR, API, SVC.
- **Phần tử thuộc lane** (mọi type trừ lane và edge): mã lane, dấu gạch ngang, rồi số, ví dụ API-1, SVC-3.
- **Edge:** chữ E rồi số, ví dụ E1, E12.

Số trong id tăng dần theo thứ tự xuất hiện lúc viết bảng lần đầu. Sau đó id không đánh lại. Cần chèn một phần tử vào giữa hai phần tử có sẵn thì thêm hậu tố thập phân vào id đứng trước:

- Chèn một cạnh sau E4, trước E5: E4.1. Chèn tiếp sau E4.1: E4.2.
- Chèn giữa E4 và E4.1: E4.0.1.
- Chèn một task sau API-3 trong lane API: API-3.1.

Id được so sánh theo từng đoạn số, nên E4 đứng trước E4.1, E4.1 đứng trước E4.2, và E4.9 đứng trước E4.10. Nhờ vậy số trong id vẫn gần với vị trí trong bảng, dễ tìm khi cuộn file, mà các tham chiếu from, to, attach không bị hỏng.

## Các dòng được xếp theo thứ tự nào?

Mọi lane đứng đầu bảng, theo thứ tự từ trái sang phải của sơ đồ.

Sau đó bảng đi theo luồng, bắt đầu từ các phần tử start. Mỗi khi viết một phần tử, lần lượt viết:

1. Chính phần tử đó.
2. Các db và text gắn vào nó (attach trỏ tới nó).
3. Tất cả các cạnh đi ra từ nó. Với condition, đây là toàn bộ các nhánh, viết liền nhau ngay sau dòng condition.
4. Các phần tử đích của những cạnh vừa viết, theo đúng thứ tự cạnh. Mỗi phần tử đích lại lặp từ bước 1.

Thứ tự các cạnh ra do người viết chọn. Nên đặt trước những nhánh kết thúc sớm như trả lỗi hay rẽ sang luồng khác, để nhánh đi tiếp dài nhất nằm cuối và luồng đọc liền mạch.

Phần tử đích bị bỏ qua ở bước 4, chưa viết tại chỗ đó, trong các trường hợp sau:

- **Đã có trong bảng:** cạnh chỉ trỏ tới nó bằng id, không viết lại.
- **Là node hợp nhánh, tức có nhiều cạnh đi vào, mà còn phần tử nguồn chưa viết:** node này chỉ được viết ngay sau khi phần tử nguồn cuối cùng của nó đã xuất hiện. Cạnh có back=true không tính khi xét điều kiện này.

Vì quy tắc hợp nhánh, một node đích chung như "trả kết quả cho người dùng" tự rơi xuống sau mọi nhánh dẫn tới nó. Các nhánh phía trên chỉ trỏ tới nó qua to=, người đọc biết node đó nằm ở phía dưới.

Phần tử nào không đi tới được từ start thì gom ở cuối bảng, sau một dòng text có nội dung "Phần còn lại".

## Ví dụ

Luồng đặt hàng dưới đây có đủ điều kiện rẽ nhánh, node hợp nhánh, bảng dữ liệu, ghi chú chèn thêm và cạnh nét đứt.

| id | type | parent | content | metadata |
|---|---|---|---|---|
| USR | lane | | Người dùng | |
| API | lane | | API | |
| SVC | lane | | Order Service | |
| USR-1 | start | USR | Gửi yêu cầu đặt hàng | |
| E1 | edge | | POST /orders | from=USR-1; to=API-1 |
| API-1 | task | API | Kiểm tra dữ liệu | |
| API-1.1 | text | API | Chỉ kiểm tra định dạng, chưa kiểm tồn kho | attach=API-1 |
| E2 | edge | | | from=API-1; to=API-2 |
| API-2 | condition | API | Dữ liệu hợp lệ? | style=highlight |
| E3 | edge | | No | from=API-2; to=API-3 |
| E4 | edge | | Yes | from=API-2; to=SVC-1 |
| API-3 | task | API | Trả lỗi 400 | |
| E5 | edge | | | from=API-3; to=USR-2 |
| SVC-1 | task | SVC | Tạo đơn hàng | |
| SVC-2 | db | SVC | DB.ORDER | attach=SVC-1 |
| E6 | edge | | | from=SVC-1; to=SVC-3 |
| SVC-3 | task | SVC | Trả kết quả | |
| E7 | edge | | | from=SVC-3; to=USR-2 |
| E7.1 | edge | | | from=SVC-3; to=SVC-4; style=dashed |
| USR-2 | end | USR | Nhận kết quả | |
| SVC-4 | end | SVC | Gửi email xác nhận | |

Đọc bảng này:

- **E3 và E4 đứng liền sau API-2**, rồi mới tới nội dung từng nhánh theo thứ tự đó: nhánh No (API-3) trước, nhánh Yes (SVC-1) sau.
- **E5 trỏ tới USR-2 trước khi USR-2 xuất hiện.** USR-2 có hai nguồn là API-3 và SVC-3, nên chỉ được viết sau SVC-3.
- **API-1.1 và E7.1 là phần tử chèn thêm sau lần viết đầu.** Id của chúng nằm giữa id của hai phần tử liền kề.

## Kiểm tra một bảng có hợp lệ không

Lỗi, bắt buộc phải sửa:

- Id bị trùng.
- from, to hoặc attach trỏ tới id không tồn tại.
- Edge thiếu from hoặc to.
- db thiếu attach.
- Phần tử thuộc lane mà parent để trống hoặc không phải id của một lane.
- Condition có ít hơn 2 cạnh ra.
- Cạnh ra của một phần tử không nằm liền sau phần tử đó (và sau các db, text gắn vào nó).

Cảnh báo, cần xem lại:

- task hoặc condition không có cạnh ra nào.
- start có cạnh đi vào, hoặc end có cạnh đi ra.
- Một node hợp nhánh xuất hiện trước một trong các phần tử nguồn của nó mà cạnh đó không ghi back=true.
- Cạnh ra của condition không có nhãn. Được phép khi sơ đồ gốc không ghi nhãn, nhưng nên bổ sung.
