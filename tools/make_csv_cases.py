"""Sinh các case csv cho bộ đối chiếu.

    python3 tools/make_csv_cases.py conformance/cases

Sinh bằng script chứ không viết tay vì vài case cần điều khiển từng byte: file mã
cp1252, file có BOM, byte không giải mã được. Chạy lại luôn ra đúng các file đó.

Mỗi case chốt một điều bản tham chiếu hứa trong README về csv.
"""
import csv
import io
import os
import sys

HEAD = ['id', 'type', 'parent', 'content', 'metadata']
FLOW = [
    ['A', 'lane', '', 'Lane A', ''],
    ['B', 'lane', '', 'Lane B', ''],
    ['A-1', 'start', 'A', 'Bắt đầu', ''],
    ['E1', 'edge', '', 'gửi yêu cầu', 'from=A-1; to=B-1'],
    ['B-1', 'task', 'B', 'Xử lý', ''],
    ['E2', 'edge', '', '', 'from=B-1; to=B-2'],
    ['B-2', 'end', 'B', 'Xong', ''],
]


def rows_to_csv(rows, d=','):
    buf = io.StringIO(newline='')
    csv.writer(buf, delimiter=d, lineterminator='\r\n').writerows(rows)
    return buf.getvalue()


CASES = {
    # Dấu phân cách được đoán, không cần chỉ định.
    'csv-01-comma': rows_to_csv([HEAD] + FLOW).encode('utf-8'),
    'csv-02-semicolon': rows_to_csv([HEAD] + FLOW, ';').encode('utf-8'),
    'csv-03-tab': rows_to_csv([HEAD] + FLOW, '\t').encode('utf-8'),
    # utf-8 có BOM: nguồn báo utf-8-sig.
    'csv-04-bom': b'\xef\xbb\xbf' + rows_to_csv([HEAD] + FLOW).encode('utf-8'),
    # Không phải utf-8: lùi về cp1252, kèm cảnh báo.
    'csv-05-cp1252': rows_to_csv([HEAD] + [r[:3] + [r[3].replace('Bắt đầu', 'Démarrer').replace('Xử lý', 'Traité')
                                                     .replace('gửi yêu cầu', 'envoyé'), r[4]] for r in FLOW]
                                 ).encode('cp1252'),
    # Tiêu đề phía trên header, ghi chú phía dưới bảng, cột thừa bên phải.
    'csv-06-title-and-notes': rows_to_csv(
        [['Luồng có tiêu đề'], ['ghi chú không thuộc bảng'], []] +
        [r + ['cột thừa'] for r in [HEAD] + FLOW] + [[], ['ghi chú sau bảng'], ['A-9', 'task', 'A', 'không đọc', '']]
    ).encode('utf-8'),
    # Header lệch sang phải một cột.
    'csv-07-header-offset': rows_to_csv([[''] + r for r in [HEAD] + FLOW]).encode('utf-8'),
    # Ô có dấu nháy chứa dấu phân cách, xuống dòng thật và dấu nháy kép.
    'csv-08-quoted-cells': rows_to_csv([HEAD] + [
        ['A', 'lane', '', 'Lane, có phẩy', ''],
        ['A-1', 'start', 'A', 'Dòng một\nDòng hai', ''],
        ['E1', 'edge', '', 'nhãn "trích"', 'from=A-1; to=A-2'],
        ['A-2', 'task', 'A', 'có<br>thẻ br', ''],
        ['E2', 'edge', '', '', 'from=A-2; to=A-3'],
        ['A-3', 'end', 'A', 'Xong', ''],
    ]).encode('utf-8'),
    # Nội dung dán từ markdown sang mà chưa bỏ escape gạch đứng.
    'csv-09-markdown-escape': rows_to_csv([HEAD] + [
        ['A', 'lane', '', 'Lane A', ''],
        ['A-1', 'start', 'A', 'a \\| b', ''],
        ['E1', 'edge', '', '', 'from=A-1; to=A-2'],
        ['A-2', 'end', 'A', 'Xong', ''],
    ]).encode('utf-8'),
    # Cả dấu phẩy lẫn dấu chấm phẩy đều tìm được một hàng header. Dấu phẩy được
    # thử trước nên thắng, và bảng đọc theo dấu phẩy mới là bảng đúng.
    'csv-10-delimiter-priority': ('id;type;parent;content;metadata\r\n' +
                                  rows_to_csv([HEAD] + FLOW)).encode('utf-8'),
    # Không có hàng header với bất kỳ dấu phân cách nào.
    'csv-90-invalid-no-header': 'a,b,c\n1,2,3\n'.encode('utf-8'),
    # Không giải mã được theo bảng mã nào: 0x81 không có trong cp1252.
    'csv-91-invalid-undecodable': b'id,type\n\x81\xff\n',
    # Bảng đúng cấu trúc nhưng có lỗi nội dung.
    'csv-92-invalid-dangling-ref': rows_to_csv([HEAD] + FLOW[:4]).encode('utf-8'),
}


def main(argv):
    if len(argv) != 1:
        print(__doc__)
        return 2
    for name, data in CASES.items():
        with open(os.path.join(argv[0], name + '.csv'), 'wb') as f:
            f.write(data)
    print(f'{argv[0]}: {len(CASES)} case csv')
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
