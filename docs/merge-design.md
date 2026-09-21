# Thiết kế: generate lại có giữ chỉnh sửa tay

Trạng thái: đã implement trong flowtable2drawio.py (lớp Merger) và test ở tests/test_merge.py. Cách dùng: README.md, mục Generate lại khi file .drawio đã có.

## Bài toán

Workflow: từ file Flow Table xxx.md generate ra xxx.drawio, người dùng sửa tay file drawio, rồi sửa bảng (thêm, bớt, đổi node và edge) và generate lại. Lần generate lại phải giữ tối đa những chỉnh sửa tay đáng giữ.

Chỉnh sửa tay cần giữ thường là:
- di chuyển vị trí element;
- chỉnh edge: thêm, bớt, di chuyển các điểm gấp.

## Quyết định đã chốt

### 1. Hai chế độ generate: force và merge

- force: sinh lại toàn bộ, bỏ file cũ.
- merge: sinh lại, nhưng giữ một số thuộc tính hình học lấy từ file hiện có, đối chiếu theo id của phần tử (id trong bảng cũng là id của cell trong drawio).
- Chọn bằng cờ. File đích đã tồn tại mà không có cờ thì hỏi lại khi chạy trong terminal. Nếu không chạy trong terminal (ví dụ agent gọi), dừng với lỗi yêu cầu chọn rõ.

### 2. Phạm vi của merge

Giữ lại từ file hiện có, theo id:
- vị trí của element;
- waypoint và điểm neo (exit/entry) của edge.

Mọi thứ khác sinh lại từ bảng: chữ, style, loại hình, lane chứa phần tử, thêm hoặc xoá element và edge.

Không lưu bản gốc của lần generate trước. Merge là 2 chiều: phần tử có trong file hiện có thì lấy hình học của nó, không phân biệt người dùng đã di chuyển hay chưa.

### 3. Đặt element mới theo láng giềng đã ghim

Element cũ giữ nguyên chỗ, nên toạ độ từ layout mới tính không còn khớp với xung quanh.

- Node mới lấy node nguồn làm mốc (nguồn cũng mới thì lấy node đích; cả hai đều mới thì dùng mốc của node nối tiếp gần nhất). Khoảng lệch so với mốc giữ như trong layout mới tính.
- Nếu chồng lên element khác, node mới dịch xuống tới chỗ trống. Không bao giờ di chuyển element cũ. Các node đã phải dịch được liệt kê trong báo cáo.
- db/text mới đứng cạnh node được attach theo cùng quy tắc.
- Edge:
  - hai đầu đều là node cũ và from/to không đổi: giữ waypoint;
  - nối tới node mới, hoặc from/to đã đổi: không ghi waypoint, để draw.io tự đi dây vuông góc, và liệt kê trong báo cáo để người dùng chỉnh tay.

### 4. Lane cũng giữ hình học

Toạ độ node trong drawio tương đối theo lane. Lane mà sinh lại bề rộng thì node cũ trôi lệch, waypoint không khớp, node có thể tràn khỏi lane. Vì vậy lane được xử lý như một element:

- Lane cũ: giữ vị trí và bề rộng từ file. Chỉ nới khi có element (cũ hoặc mới) không vừa. Khi nới thì các lane bên phải, và waypoint của edge nằm bên phải, dịch sang cùng một khoảng.
- Lane mới: chèn theo thứ tự trong bảng, bề rộng theo layout mới tính, các lane bên phải dịch sang.
- Lane bị xoá khỏi bảng: bỏ đi, các lane bên phải dồn sang trái (waypoint dịch theo).
- Thứ tự lane luôn theo bảng.
- Pool tự co giãn để bao hết nội dung; tiêu đề lấy từ file .md.

### 5. Kích thước sinh lại, neo theo tâm

- Kích thước node luôn tính lại theo chữ; kích thước người dùng tự kéo không được giữ.
- Vị trí được giữ là tâm node. Nhờ vậy dây vào đỉnh/đáy vẫn đi qua trung điểm cạnh, dây vào mặt bên giữ nguyên độ cao, waypoint cũ vẫn vuông góc với node.
- Node to ra đè lên element bên cạnh thì không đẩy gì cả, chỉ liệt kê cặp chồng nhau trong báo cáo.

### 6. Nhãn edge nằm ngoài phạm vi merge, trừ edge chưa bị sửa

- Chữ của nhãn luôn sinh lại từ bảng.
- Merge không giữ vị trí nhãn từ file. Vị trí nhãn do tool tính dựa trên đường dây của chính tool, nên không áp được lên đường dây giữ từ file.
- Nếu đường dây của edge trong file trùng với đường tool vừa tính (sai lệch trong ngưỡng nhỏ), edge đó coi như chưa bị sửa và dùng vị trí nhãn tool tính.
- Các edge còn lại không ghi vị trí nhãn; draw.io đặt nhãn giữa đường. Tool không báo nhãn đè ở các edge này.
- Chế độ force không đổi: nhãn đặt vào chỗ trống như hiện tại.

### 7. Giữ cell người dùng tự vẽ

- Mỗi cell do tool sinh mang một dấu nhận biết trong style (vd flowtable=1). Cell có dấu mà id không còn trong bảng thì bị xoá.
- Cell không có dấu và không phải pool/lane là cell tự vẽ: chép nguyên sang file mới (style, chữ, hình học). Lane cha còn thì giữ trong lane đó; lane đã bị xoá thì chuyển vào pool, giữ vị trí tuyệt đối.
- Edge tự vẽ nối tới node đã bị xoá: bỏ đầu nối đó, edge thành dây trơ đầu.
- Báo cáo liệt kê các cell tự vẽ đã giữ.
- Chế độ force bỏ mọi cell tự vẽ.
- File sinh từ phiên bản tool chưa gắn dấu (không cell nào có dấu): cell có id đúng dạng id của bảng (LANE-số, E+số, có thể kèm hậu tố thập phân) được coi là do tool sinh.

### 8. Tên file đích chỉ đổi đuôi

- File đích mặc định là file .md đổi đuôi thành .drawio, cùng thư mục (xxx.md --> xxx.drawio). Không có quy tắc tên nào khác.
- Cờ -o ghi đè đường dẫn đích.
- Báo cáo luôn in file đích và chế độ đang chạy (tạo mới, force, merge).

### 9. Sao lưu file cũ trước khi ghi đè

- Trước khi ghi đè (merge hoặc force), chép file đích hiện có thành <file>.drawio.bak cùng thư mục. Chỉ giữ một bản, lần sau ghi đè lên.
- File đích chưa tồn tại thì không tạo .bak.
- Cờ --no-backup tắt việc sao lưu.
- Báo cáo in đường dẫn file .bak.

## Ghi chú khi implement

- Tool đọc được cả file drawio dạng nén (base64 + deflate) và cell bọc trong object/UserObject. Các trang khác của file được chép nguyên sang file mới.
- Chiều cao pool sau merge tính bằng đáy thấp nhất cộng min_channel, để bảng không đổi thì file không đổi.
- So đường dây để biết edge đã bị sửa hay chưa: so waypoint và điểm neo với đường tool vừa tính, ngưỡng 2px.
