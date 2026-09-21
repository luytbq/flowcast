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
cases/    bảng đầu vào, viết tay, mỗi file chốt một nhánh thuật toán
golden/   đầu ra do bản tham chiếu sinh, commit vào repo
```

Mỗi case cho ra hai file trong golden/:

| file | nội dung |
|---|---|
| `<case>.drawio` | xml đầy đủ, so từng byte |
| `<case>.report.txt` | issue, cảnh báo layout, phát hiện tự kiểm, số liệu |

Case mang tiền tố 9x là bảng có lỗi. Chúng không có file .drawio, chỉ có report.
Đó là cố ý: chúng chốt hành vi kiểm tra đầu vào, thứ bản port cũng phải tái tạo
đúng, kể cả nguyên văn thông điệp và số dòng.

## Lệnh

```
python3 conformance/generate.py            # sinh lại golden
python3 conformance/generate.py --check    # so golden hiện có, lệch thì mã thoát 1
python3 tools/coverage.py                  # đo bộ case chạm tới nhánh nào
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
| đi dây A, thẳng đứng | 73 |
| đi dây B, thẳng ngang | 38 |
| đi dây C, chữ L | 1 |
| đi dây D, qua kênh và máng | 16 |
| nhánh dạt trái | 11 |
| nhánh dạt phải | 14 |
| nhánh không rõ hướng | 69 |
| đoạn nằm ở track thứ hai trở lên | 4 |
| db hoặc text bám node | 7 |
| cạnh back | 2 |
| cạnh có nhãn | 66 |
| phần tử tô nhấn | 3 |
| ngắt dòng cứng giữa từ | 1 |

### Lỗ hổng đã biết

**Đi dây kiểu C chỉ được chạm đúng 1 lần.** Một case là đủ để nói nhánh đó có
chạy, không đủ để nói nó chạy đúng trong nhiều tình huống. Cần thêm case trước
khi tin vào bộ này cho phần C.

**Chưa có case nào cho merge.** Ngữ nghĩa merge là 490 dòng cộng 255 dòng test
trong reference/tests/test_merge.py, và nó chưa nằm trong bộ đối chiếu. Phải bổ
sung trước khi port phần merge sang Go.

**Chưa có case csv và xlsx.** reference/tests/test_inputs.py đã khẳng định ba
định dạng cho ra cùng một file, nên rủi ro thấp, nhưng bản port cần bộ riêng cho
việc đoán dấu phân cách, bảng mã, ô gộp và hàng ẩn.

## Thêm case

1. Viết bảng vào cases/, đặt tên theo nhánh nó chốt, không theo nội dung nghiệp
   vụ. `25-tracks-parallel.md`, không phải `25-luong-dat-hang.md`.
2. Giữ bảng nhỏ nhất có thể mà vẫn ép được nhánh cần chốt.
3. Chạy tools/coverage.py để xác nhận nhánh đó thật sự tăng. Một case được thêm
   vì nghĩ rằng nó chạm tới nhánh nào đó, mà số không đổi, là một case vô ích.
4. Sinh lại golden rồi đọc diff.
