"""Sinh các case xlsx cho bộ đối chiếu.

    python3 tools/make_xlsx_cases.py conformance/cases

Dựng XML bằng reference/tests/xlsx_fixture.py, nhưng ghi zip với thời điểm cố định:
bộ tạo của bản tham chiếu dùng thời điểm hiện tại, nên mỗi lần chạy lại ra một file
khác vài byte dù nội dung như nhau.

Mỗi case chốt một điều bản tham chiếu hứa trong README về xlsx.
"""
import html
import os
import sys
import zipfile

sys.path.insert(0, os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), 'reference', 'tests'))

from xlsx_fixture import MAIN, PKG, REL, sheet_xml  # noqa: E402

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
FIXED = (2026, 1, 1, 0, 0, 0)


def at(start, rows):
    return [(start + i, cells) for i, cells in enumerate(rows)]


def shared(rows):
    return [[(v, 'shared') if v != '' else '' for v in r] for r in rows]


def build(sheets, absolute=False):
    """sheets: list của (tên, hàng, hàng ẩn, ô gộp). Trả về bytes của file xlsx.

    absolute ghi đường dẫn sheet trong workbook.xml.rels dạng /xl/worksheets/...,
    như vài công cụ ngoài Excel làm, thay vì dạng tương đối.
    """
    import io
    strings = []
    parts = [(name, i, sheet_xml(rows, strings, hidden, merges))
             for i, (name, rows, hidden, merges) in enumerate(sheets, start=1)]
    wb_sheets = ''.join(f'<sheet name="{html.escape(n, quote=True)}" sheetId="{i}" r:id="rId{i}"/>'
                        for n, i, _ in parts)
    files = [
        ('[Content_Types].xml',
         '<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>'),
        ('xl/workbook.xml', f'<?xml version="1.0"?><workbook xmlns="{MAIN}" xmlns:r="{REL}"><sheets>{wb_sheets}</sheets></workbook>'),
        ('xl/_rels/workbook.xml.rels', f'<?xml version="1.0"?><Relationships xmlns="{PKG}">' +
         ''.join(f'<Relationship Id="rId{i}" Type="x/worksheet" '
                 f'Target="{"/xl/" if absolute else ""}worksheets/sheet{i}.xml"/>' for _, i, _ in parts) +
         '</Relationships>'),
    ] + [(f'xl/worksheets/sheet{i}.xml', xml) for _, i, xml in parts]
    if strings:
        items = ''.join(f'<si><t xml:space="preserve">{html.escape(s, quote=True)}</t></si>' for s in strings)
        files.append(('xl/sharedStrings.xml',
                      f'<?xml version="1.0"?><sst xmlns="{MAIN}" count="{len(strings)}" '
                      f'uniqueCount="{len(strings)}">{items}</sst>'))
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, 'w') as z:
        for name, text in files:
            z.writestr(zipfile.ZipInfo(name, date_time=FIXED), text)
    return buf.getvalue()


CASES = {
    # Chuỗi dùng chung, cách Excel thật lưu chữ.
    'xlsx-01-shared-strings': [('Flow', at(1, shared([HEAD] + FLOW)), (), ())],
    # Chuỗi nội tuyến.
    'xlsx-02-inline-strings': [('Flow', at(1, [HEAD] + FLOW), (), ())],
    # Tiêu đề phía trên, header lệch sang cột B, ghi chú phía dưới bảng.
    'xlsx-03-title-offset': [('Flow', [(1, ['Luồng trong Excel']), (2, [])] +
                              at(3, [[''] + r for r in [HEAD] + FLOW]) +
                              [(11, []), (12, ['', 'ghi chú dưới bảng'])], (), ())],
    # id và parent kiểu số: cảnh báo ở cả hai cột, và 7.0 đọc thành 7. Tránh 0 và
    # 1, hai id dành riêng của draw.io.
    'xlsx-04-numeric-ids': [('Flow', at(1, [HEAD] + [
        [('3', 'num'), 'lane', '', 'Lane số', ''],
        [('5', 'num'), 'start', ('3', 'num'), 'Vào', ''],
        ['E1', 'edge', '', '', 'from=5; to=7'],
        [('7.0', 'num'), 'end', ('3', 'num'), 'Xong', ''],
    ]), (), ())],
    # Ô công thức có giá trị lưu sẵn và ô công thức chưa có giá trị.
    'xlsx-05-formulas': [('Flow', at(1, [HEAD] + [
        ['A', 'lane', '', ('Lane từ công thức', 'formula'), ''],
        ['A-1', 'start', 'A', 'Vào', ''],
        ['E1', 'edge', '', ('', 'formula_empty'), 'from=A-1; to=A-2'],
        ['A-2', 'end', 'A', 'Xong', ''],
    ]), (), ())],
    # Hàng ẩn và ô gộp trong vùng bảng thì cảnh báo; ở ngoài vùng bảng thì không.
    'xlsx-06-hidden-and-merged': [('Flow', [(1, ['Tiêu đề'])] + at(2, [HEAD] + FLOW),
                                   (1, 5), ('A1:C1', 'D6:E6'))],
    # Sheet đầu không có header nên bỏ qua; lấy sheet thứ hai.
    'xlsx-07-second-sheet': [('Ghi chú', [(1, ['không phải bảng'])], (), ()),
                             ('Luồng', at(1, [HEAD] + FLOW), (), ())],
    # Hai sheet đều có header: sheet đầu thắng, trừ khi chỉ định --sheet.
    'xlsx-08-two-flows': [('Một', at(1, [HEAD] + FLOW[:4] + [['B-1', 'end', 'B', 'Xong sớm', '']]), (), ()),
                          ('Hai', at(1, [HEAD] + FLOW), (), ())],
    # Không sheet nào có hàng header.
    'xlsx-90-invalid-no-header': [('Flow', [(1, ['a', 'b']), (2, ['1', '2'])], (), ())],
}


def main(argv):
    if len(argv) != 1:
        print(__doc__)
        return 2
    for name, sheets in CASES.items():
        with open(os.path.join(argv[0], name + '.xlsx'), 'wb') as f:
            f.write(build(sheets))
    # Đường dẫn sheet tuyệt đối.
    with open(os.path.join(argv[0], 'xlsx-09-absolute-target.xlsx'), 'wb') as f:
        f.write(build([('Flow', at(1, [HEAD] + FLOW), (), ())], absolute=True))
    # Không phải file zip.
    with open(os.path.join(argv[0], 'xlsx-91-invalid-not-zip.xlsx'), 'wb') as f:
        f.write(b'day khong phai file xlsx\n')
    print(f'{argv[0]}: {len(CASES) + 2} case xlsx')
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
