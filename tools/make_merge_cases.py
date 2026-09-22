"""Sinh các kịch bản merge cho bộ đối chiếu.

    python3 tools/make_merge_cases.py conformance/merge

Mỗi kịch bản là một thư mục gồm flow.md và flow.drawio. flow.drawio là file mà
bản tham chiếu sinh lần đầu từ bảng gốc, rồi được sửa tay như người dùng thật làm
trong draw.io: kéo node, thêm điểm gấp, vẽ thêm ghi chú. flow.md là bảng sau khi đã
sửa. Chạy build --mode merge trên cặp đó là chạy đúng việc merge phải làm.

Sửa file .drawio bằng ElementTree rồi ghi lại không thụt lề, như draw.io không giữ
nguyên cách thụt lề của file mà nó mở ra.
"""
import base64
import io
import os
import shutil
import sys
import urllib.parse
import xml.etree.ElementTree as ET
import zlib

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.join(ROOT, 'reference'))

import flowtable2drawio as ft  # noqa: E402

FIX = os.path.join(ROOT, 'reference', 'tests', 'fixtures')


def cell(root, cid):
    return [c for c in root if c.get('id') == cid][0]


def move(cid, dx, dy):
    def fn(root):
        g = cell(root, cid).find('mxGeometry')
        g.set('x', str(float(g.get('x')) + dx))
        g.set('y', str(float(g.get('y')) + dy))
    return fn


def waypoints(eid, pts):
    def fn(root):
        g = cell(root, eid).find('mxGeometry')
        for old in g.findall("Array[@as='points']"):
            g.remove(old)
        arr = ET.SubElement(g, 'Array', {'as': 'points'})
        for x, y in pts:
            ET.SubElement(arr, 'mxPoint', x=str(x), y=str(y))
    return fn


def set_style(cid, key, value):
    def fn(root):
        c = cell(root, cid)
        parts = [p for p in c.get('style', '').split(';') if p and not p.startswith(key + '=')]
        c.set('style', ';'.join(parts + [f'{key}={value}']) + ';')
    return fn


def note(cid, parent, x, y, text='Ghi chú tay'):
    def fn(root):
        n = ET.SubElement(root, 'mxCell', id=cid, value=text, style='text;html=1;', vertex='1', parent=parent)
        ET.SubElement(n, 'mxGeometry', x=str(x), y=str(y), width='120', height='40', **{'as': 'geometry'})
    return fn


def arrow(cid, source, target):
    def fn(root):
        a = ET.SubElement(root, 'mxCell', id=cid, style='endArrow=classic;html=1;', edge='1',
                          parent='1', source=source, target=target)
        ET.SubElement(a, 'mxGeometry', relative='1', **{'as': 'geometry'})
    return fn


def group(gid, child, parent):
    """Một nhóm tự vẽ chứa một ô con: ô con có toạ độ tương đối theo nhóm."""
    def fn(root):
        g = ET.SubElement(root, 'mxCell', id=gid, value='', style='group;', vertex='1', parent=parent)
        ET.SubElement(g, 'mxGeometry', x='40', y='420', width='200', height='80', **{'as': 'geometry'})
        c = ET.SubElement(root, 'mxCell', id=child, value='trong nhóm', style='text;', vertex='1', parent=gid)
        ET.SubElement(c, 'mxGeometry', x='10', y='10', width='100', height='30', **{'as': 'geometry'})
    return fn


def object_note(cid, parent, x, y):
    """Ghi chú mang thuộc tính riêng: draw.io bọc mxCell trong một phần tử object."""
    def fn(root):
        o = ET.SubElement(root, 'object', id=cid, label='Ghi chú có thuộc tính', owner='QA')
        n = ET.SubElement(o, 'mxCell', style='text;html=1;', vertex='1', parent=parent)
        ET.SubElement(n, 'mxGeometry', x=str(x), y=str(y), width='120', height='40', **{'as': 'geometry'})
    return fn


def strip_marks(root):
    for c in root:
        if c.get('style'):
            c.set('style', c.get('style').replace('flowtable=1;', ''))


def compress_first_page(path):
    tree = ET.parse(path)
    diagram = tree.getroot().find('diagram')
    model = diagram.find('mxGraphModel')
    raw = urllib.parse.quote(ET.tostring(model, encoding='unicode'), safe='')
    comp = zlib.compressobj(9, zlib.DEFLATED, -15)
    diagram.remove(model)
    diagram.text = base64.b64encode(comp.compress(raw.encode()) + comp.flush()).decode()
    tree.write(path)


def bare_model(path):
    """File chỉ còn mxGraphModel, không có mxfile và diagram bọc ngoài."""
    tree = ET.parse(path)
    ET.ElementTree(tree.getroot().find('diagram').find('mxGraphModel')).write(path)


def add_page(path):
    tree = ET.parse(path)
    page = ET.SubElement(tree.getroot(), 'diagram', id='trang-2', name='Trang hai')
    m = ET.SubElement(page, 'mxGraphModel')
    r = ET.SubElement(m, 'root')
    ET.SubElement(r, 'mxCell', id='0')
    ET.SubElement(r, 'mxCell', id='1', parent='0')
    tree.write(path)


# (bảng gốc, các sửa trên .drawio, các sửa trên bảng, hậu xử lý file)
SCENARIOS = {
    'unchanged': ('order.md', [], [], None),
    'moved-node': ('order.md', [move('API-1', 60, 25)], [('| Trả lỗi 400 |', '| Trả lỗi 400 kèm chi tiết |')], None),
    'text-grows': ('order.md', [], [('| Trả lỗi 400 |', '| Trả lỗi 400 kèm danh sách trường sai |')], None),
    'waypoints-kept': ('order.md', [waypoints('E3', [(560, 182), (560, 230)])],
                       [('| Trả lỗi 400 |', '| Trả lỗi 400 kèm chi tiết |')], None),
    'waypoints-untouched': ('order.md', [], [('| Gửi email xác nhận |', '| Gửi email |')], None),
    'exit-constraint-edited': ('order.md', [set_style('E4', 'exitX', '0.25')], [], None),
    'new-node': ('order.md', [], [('| E2 | edge | | | from=API-1; to=API-2 |',
                                   '| E2 | edge | | | from=API-1; to=API-1.5 |\n'
                                   '| API-1.5 | task | API | Chuẩn hoá dữ liệu | |\n'
                                   '| E2.1 | edge | | | from=API-1.5; to=API-2 |')], None),
    'deleted-rows': ('order.md', [], [('| E7.1 | edge | | | from=SVC-3; to=SVC-4; style=dashed |\n', ''),
                                      ('| SVC-4 | end | SVC | Gửi email xác nhận | |\n', '')], None),
    'new-lane': ('order.md', [], [('| USR | lane | | Người dùng | |',
                                   '| ADM | lane | | Quản trị | |\n| USR | lane | | Người dùng | |')], None),
    'node-changes-lane': ('order.md', [], [('| API-3 | task | API | Trả lỗi 400 | |',
                                            '| API-3 | task | SVC | Trả lỗi 400 | |')], None),
    'lane-grows': ('order.md', [move('SVC-1', 260, 0)], [], None),
    'moved-above-top': ('order.md', [move('USR-1', 0, -80)], [], None),
    'new-node-overlaps': ('order.md', [move('SVC-3', 0, 120)],
                          [('| E6 | edge | | | from=SVC-1; to=SVC-3 |',
                            '| E6 | edge | | | from=SVC-1; to=SVC-2.5 |\n'
                            '| SVC-2.5 | task | SVC | Kiểm tồn kho | |\n'
                            '| E6.1 | edge | | | from=SVC-2.5; to=SVC-3 |')], None),
    'freehand-note': ('order.md', [note('hand-1', 'pool', 30, 330)],
                      [('| Trả lỗi 400 |', '| Trả lỗi 400 kèm chi tiết |')], None),
    'freehand-note-in-lane': ('order.md', [note('hand-1', 'SVC', 30, 330)],
                              [('| USR | lane | | Người dùng | |',
                                '| ADM | lane | | Quản trị | |\n| USR | lane | | Người dùng | |')], None),
    'freehand-note-lane-deleted': ('order.md', [note('hand-1', 'SVC', 30, 330)], 'drop-svc', None),
    'freehand-edge-detached': ('order.md', [arrow('hand-2', 'USR-2', 'SVC-4')],
                               [('| E7.1 | edge | | | from=SVC-3; to=SVC-4; style=dashed |\n', ''),
                                ('| SVC-4 | end | SVC | Gửi email xác nhận | |\n', '')], None),
    'freehand-group': ('order.md', [group('hand-g', 'hand-g1', 'pool')], [], None),
    'no-marks': ('order.md', [strip_marks], [('| Trả lỗi 400 |', '| Trả lỗi 400 kèm chi tiết |')], None),
    'compressed': ('order.md', [move('API-1', 40, 0)], [], compress_first_page),
    'second-page': ('order.md', [], [('| Trả lỗi 400 |', '| Trả lỗi 400 kèm chi tiết |')], add_page),
    'retry-moved': ('retry.md', [], [], None),
    'freehand-object': ('order.md', [object_note('hand-o', 'API', 20, 200)], [], None),
    'bare-model': ('order.md', [move('USR-1', 30, 0)], [], bare_model),
}

# File .drawio hỏng, kèm bảng gốc. Chỉ gồm những lỗi mà thông báo do chính tool
# viết ra; thông báo của bộ đọc XML và của zlib khác nhau giữa hai bản.
BROKEN = {
    'broken-no-pages': '<mxfile host="x"><other /></mxfile>',
    'broken-no-root': '<mxfile><diagram id="a"><mxGraphModel /></diagram></mxfile>',
    'broken-padding': '<mxfile><diagram id="a">QUJ</diagram></mxfile>',
}


def drop_svc(md):
    """Bỏ lane SVC cùng mọi phần tử và cạnh chạm tới nó.

    Nhánh Yes của API-2 đổi đích sang USR-2 trước khi bỏ, để condition vẫn còn đủ
    hai nhánh.
    """
    md = md.replace('| E4 | edge | | Yes | from=API-2; to=SVC-1 |', '| E4 | edge | | Yes | from=API-2; to=USR-2 |')
    return '\n'.join(line for line in md.split('\n') if '| SVC' not in line and 'SVC-' not in line)


def main(argv):
    if len(argv) != 1:
        print(__doc__)
        return 2
    dst = argv[0]
    if os.path.isdir(dst):
        shutil.rmtree(dst)
    for name, (base, drawio_edits, table_edits, post) in SCENARIOS.items():
        d = os.path.join(dst, name)
        os.makedirs(d)
        md_path, drawio_path = os.path.join(d, 'flow.md'), os.path.join(d, 'flow.drawio')
        shutil.copy(os.path.join(FIX, base), md_path)
        if ft.main(['build', md_path, '-o', drawio_path]) != 0:
            raise SystemExit(f'ERROR {name}: build lần đầu không thành công')
        if drawio_edits:
            tree = ET.parse(drawio_path)
            root = tree.getroot().find('.//root')
            for fn in drawio_edits:
                fn(root)
            tree.write(drawio_path)
        if post:
            post(drawio_path)
        md = io.open(md_path, encoding='utf-8').read()
        if table_edits == 'drop-svc':
            md = drop_svc(md)
        else:
            for a, b in table_edits:
                if a not in md:
                    raise SystemExit(f'ERROR {name}: không thấy {a!r} trong bảng')
                md = md.replace(a, b)
        io.open(md_path, 'w', encoding='utf-8').write(md)
    for name, text in BROKEN.items():
        d = os.path.join(dst, name)
        os.makedirs(d)
        shutil.copy(os.path.join(FIX, 'order.md'), os.path.join(d, 'flow.md'))
        io.open(os.path.join(d, 'flow.drawio'), 'w', encoding='utf-8').write(text)
    print(f'{dst}: {len(SCENARIOS) + len(BROKEN)} kịch bản merge')
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
