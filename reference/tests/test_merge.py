import base64
import os
import shutil
import sys
import tempfile
import unittest
import urllib.parse
import xml.etree.ElementTree as ET
import zlib

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.dirname(HERE))

import flowtable2drawio as ft  # noqa: E402

FIX = os.path.join(HERE, 'fixtures')


def cells(path):
    root = ET.parse(path).getroot().find('.//root')
    return {c.get('id'): c for c in root if c.get('id')}


def geo(path, cid):
    g = cells(path)[cid].find('mxGeometry')
    return tuple(float(g.get(k, 0)) for k in ('x', 'y', 'width', 'height'))


def center(path, cid):
    x, y, w, h = geo(path, cid)
    return (x + w / 2, y + h / 2)


def points(path, cid):
    g = cells(path)[cid].find('mxGeometry')
    arr = g.find("Array[@as='points']")
    return [(float(p.get('x')), float(p.get('y'))) for p in arr.findall('mxPoint')] if arr is not None else []


class MergeCase(unittest.TestCase):
    """Mỗi ca: build lần đầu, sửa tay file drawio, sửa bảng, rồi build lại ở chế độ merge."""

    fixture = 'order.md'

    def setUp(self):
        self.dir = tempfile.mkdtemp()
        self.addCleanup(shutil.rmtree, self.dir)
        self.md = os.path.join(self.dir, 'flow.md')
        self.drawio = os.path.join(self.dir, 'flow.drawio')
        shutil.copy(os.path.join(FIX, self.fixture), self.md)
        self.assertEqual(ft.main(['build', self.md]), 0)

    def edit_table(self, old, new):
        with open(self.md, encoding='utf-8') as f:
            s = f.read()
        self.assertIn(old, s)
        with open(self.md, 'w', encoding='utf-8') as f:
            f.write(s.replace(old, new))

    def edit_drawio(self, fn):
        tree = ET.parse(self.drawio)
        fn(tree.getroot().find('.//root'))
        tree.write(self.drawio)

    def merge(self, *extra):
        return ft.main(['build', self.md, '--mode', 'merge'] + list(extra))


class KeepGeometryTest(MergeCase):
    def move(self, cid, dx, dy):
        def fn(root):
            g = [c for c in root if c.get('id') == cid][0].find('mxGeometry')
            g.set('x', str(float(g.get('x')) + dx))
            g.set('y', str(float(g.get('y')) + dy))
        self.edit_drawio(fn)

    def test_moved_node_keeps_center(self):
        self.move('API-1', 60, 25)
        moved = center(self.drawio, 'API-1')
        self.edit_table('| Trả lỗi 400 |', '| Trả lỗi 400 kèm chi tiết |')
        self.assertEqual(self.merge(), 0)
        self.assertEqual(center(self.drawio, 'API-1'), moved)

    def test_text_change_keeps_center_and_resizes(self):
        before = center(self.drawio, 'API-3')
        w_before = geo(self.drawio, 'API-3')[2]
        self.edit_table('| Trả lỗi 400 |', '| Trả lỗi 400 kèm danh sách trường sai |')
        self.assertEqual(self.merge(), 0)
        self.assertEqual(center(self.drawio, 'API-3'), before)
        self.assertGreater(geo(self.drawio, 'API-3')[2], w_before)

    def test_edge_waypoints_kept(self):
        def fn(root):
            g = [c for c in root if c.get('id') == 'E4'][0].find('mxGeometry')
            arr = ET.SubElement(g, 'Array', {'as': 'points'})
            ET.SubElement(arr, 'mxPoint', x='560', y='182')
            ET.SubElement(arr, 'mxPoint', x='560', y='230')
        self.edit_drawio(fn)
        self.edit_table('| Trả lỗi 400 |', '| Trả lỗi 400 kèm chi tiết |')
        self.assertEqual(self.merge(), 0)
        self.assertEqual(points(self.drawio, 'E4'), [(560.0, 182.0), (560.0, 230.0)])

    def test_unchanged_table_keeps_all_geometry(self):
        before = {cid: geo(self.drawio, cid) for cid in cells(self.drawio) if cid not in ('0', '1')}
        pts = {cid: points(self.drawio, cid) for cid in before}
        self.assertEqual(self.merge(), 0)
        for cid, box in before.items():
            if cid == 'pool':
                continue
            self.assertEqual(geo(self.drawio, cid), box, cid)
            self.assertEqual(points(self.drawio, cid), pts[cid], cid)


class ChangeTableTest(MergeCase):
    def test_new_node_placed_below_pinned_source(self):
        self.edit_table('| E2 | edge | | | from=API-1; to=API-2 |',
                        '| E2 | edge | | | from=API-1; to=API-1.5 |\n'
                        '| API-1.5 | task | API | Tính phí | |\n'
                        '| E2.1 | edge | | | from=API-1.5; to=API-2 |')
        api1 = center(self.drawio, 'API-1')
        self.assertEqual(self.merge(), 0)
        self.assertEqual(center(self.drawio, 'API-1'), api1)
        new = center(self.drawio, 'API-1.5')
        self.assertGreater(new[1], api1[1])
        self.assertEqual(points(self.drawio, 'E2'), [])

    def test_deleted_rows_disappear(self):
        self.edit_table('| E7.1 | edge | | | from=SVC-3; to=SVC-4; style=dashed |\n', '')
        self.edit_table('| SVC-4 | end | SVC | Gửi email xác nhận | |\n', '')
        self.assertEqual(self.merge(), 0)
        ids = cells(self.drawio)
        self.assertNotIn('SVC-4', ids)
        self.assertNotIn('E7.1', ids)

    def test_new_lane_shifts_old_items_right(self):
        before = center(self.drawio, 'API-1')
        self.edit_table('| USR | lane | | Người dùng | |',
                        '| AUD | lane | | Audit | |\n| USR | lane | | Người dùng | |')
        self.assertEqual(self.merge(), 0)
        lane_w = geo(self.drawio, 'AUD')[2]
        self.assertGreater(lane_w, 0)
        self.assertAlmostEqual(center(self.drawio, 'API-1')[0], before[0], places=6)
        self.assertAlmostEqual(geo(self.drawio, 'USR')[0], lane_w, places=6)

    def test_node_moved_to_other_lane_is_replaced(self):
        self.edit_table('| API-3 | task | API | Trả lỗi 400 | |', '| API-3 | task | SVC | Trả lỗi 400 | |')
        self.assertEqual(self.merge(), 0)
        self.assertEqual(cells(self.drawio)['API-3'].get('parent'), 'SVC')
        x = geo(self.drawio, 'API-3')[0]
        self.assertGreaterEqual(x, 0)
        self.assertLess(x, geo(self.drawio, 'SVC')[2])


class FreehandTest(MergeCase):
    def add_note(self, root, cid='hand-1', parent='API'):
        note = ET.SubElement(root, 'mxCell', id=cid, value='Ghi chú tay',
                             style='shape=note;whiteSpace=wrap;html=1;', vertex='1', parent=parent)
        ET.SubElement(note, 'mxGeometry', x='30', y='330', width='120', height='40', **{'as': 'geometry'})

    def test_note_survives_merge(self):
        self.edit_drawio(self.add_note)
        box = geo(self.drawio, 'hand-1')
        self.edit_table('| Trả lỗi 400 |', '| Trả lỗi 400 kèm chi tiết |')
        self.assertEqual(self.merge(), 0)
        self.assertEqual(geo(self.drawio, 'hand-1'), box)
        self.assertEqual(cells(self.drawio)['hand-1'].get('parent'), 'API')

    def test_note_moves_into_pool_when_lane_deleted(self):
        self.edit_drawio(lambda root: self.add_note(root, parent='SVC'))
        lane_x = geo(self.drawio, 'SVC')[0]
        box = geo(self.drawio, 'hand-1')
        for row in ('| SVC | lane | | Order Service | |\n', '| SVC-1 | task | SVC | Tạo đơn hàng | |\n',
                    '| SVC-2 | db | SVC | DB.ORDER | attach=SVC-1 |\n', '| SVC-3 | task | SVC | Trả kết quả | |\n',
                    '| SVC-4 | end | SVC | Gửi email xác nhận | |\n',
                    '| E4 | edge | | Yes | from=API-2; to=SVC-1 |\n', '| E6 | edge | | | from=SVC-1; to=SVC-3 |\n',
                    '| E7 | edge | | | from=SVC-3; to=USR-2 |\n',
                    '| E7.1 | edge | | | from=SVC-3; to=SVC-4; style=dashed |\n'):
            self.edit_table(row, '')
        self.edit_table('| E3 | edge | | No | from=API-2; to=API-3 |',
                        '| E3 | edge | | No | from=API-2; to=API-3 |\n| E4 | edge | | Yes | from=API-2; to=USR-2 |')
        self.assertEqual(self.merge(), 0)
        c = cells(self.drawio)['hand-1']
        self.assertEqual(c.get('parent'), 'pool')
        self.assertAlmostEqual(geo(self.drawio, 'hand-1')[0], box[0] + lane_x, places=6)

    def test_freehand_edge_to_deleted_node_is_detached(self):
        def fn(root):
            self.add_note(root)
            arrow = ET.SubElement(root, 'mxCell', id='hand-2', style='endArrow=classic;html=1;',
                                  edge='1', parent='1', source='hand-1', target='SVC-4')
            ET.SubElement(arrow, 'mxGeometry', relative='1', **{'as': 'geometry'})
        self.edit_drawio(fn)
        self.edit_table('| E7.1 | edge | | | from=SVC-3; to=SVC-4; style=dashed |\n', '')
        self.edit_table('| SVC-4 | end | SVC | Gửi email xác nhận | |\n', '')
        self.assertEqual(self.merge(), 0)
        arrow = cells(self.drawio)['hand-2']
        self.assertIsNone(arrow.get('target'))
        self.assertEqual(arrow.get('source'), 'hand-1')
        self.assertTrue(arrow.find("mxGeometry/mxPoint[@as='targetPoint']") is not None)


class ModeTest(MergeCase):
    def test_second_build_without_mode_stops(self):
        self.assertEqual(ft.main(['build', self.md]), 4)

    def test_force_drops_manual_edits(self):
        self.edit_drawio(lambda root: FreehandTest.add_note(self, root))
        self.assertEqual(ft.main(['build', self.md, '--mode', 'force']), 0)
        self.assertNotIn('hand-1', cells(self.drawio))

    def test_backup_written_unless_disabled(self):
        self.assertEqual(self.merge(), 0)
        self.assertTrue(os.path.exists(self.drawio + '.bak'))
        os.remove(self.drawio + '.bak')
        self.assertEqual(self.merge('--no-backup'), 0)
        self.assertFalse(os.path.exists(self.drawio + '.bak'))

    def test_broken_target_is_not_overwritten(self):
        with open(self.drawio, 'w', encoding='utf-8') as f:
            f.write('<mxfile><diagram>khong-phai-xml')
        self.assertEqual(self.merge(), 4)
        with open(self.drawio, encoding='utf-8') as f:
            self.assertTrue(f.read().startswith('<mxfile><diagram>'))

    def test_compressed_target_is_read(self):
        tree = ET.parse(self.drawio)
        model = tree.getroot().find('diagram/mxGraphModel')
        raw = ET.tostring(model, encoding='unicode')
        co = zlib.compressobj(9, zlib.DEFLATED, -15)
        data = co.compress(urllib.parse.quote(raw, safe='~()*!.\'').encode()) + co.flush()
        page = tree.getroot().find('diagram')
        page.remove(model)
        page.text = base64.b64encode(data).decode()
        tree.write(self.drawio)
        api1 = center(self.drawio + '.tmp', 'API-1') if False else None
        self.assertEqual(self.merge(), 0)
        self.assertIn('API-1', cells(self.drawio))


class OtherPagesTest(MergeCase):
    def test_other_pages_survive(self):
        tree = ET.parse(self.drawio)
        page = ET.SubElement(tree.getroot(), 'diagram', id='p2', name='Ghi chú')
        m = ET.SubElement(page, 'mxGraphModel')
        r = ET.SubElement(m, 'root')
        ET.SubElement(r, 'mxCell', id='0')
        ET.SubElement(r, 'mxCell', id='1', parent='0')
        tree.write(self.drawio)
        self.assertEqual(self.merge(), 0)
        names = [d.get('name') for d in ET.parse(self.drawio).getroot().findall('diagram')]
        self.assertEqual(names[-1], 'Ghi chú')


if __name__ == '__main__':
    unittest.main()
