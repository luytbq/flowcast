import os
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.dirname(HERE))

import flowtable2drawio as ft  # noqa: E402

FIX = os.path.join(HERE, 'fixtures')

HEAD = '| id | type | parent | content | metadata |\n|---|---|---|---|---|\n'


def table(*rows):
    return HEAD + '\n'.join(rows) + '\n'


def issues_of(md):
    t = ft.read_markdown_text(md)
    return t.issues + ft.validate(t.rows)


def build(name):
    table = ft.load(os.path.join(FIX, name))
    assert not [i for i in table.issues if i.level == 'error']
    return ft.Layout(table.rows).run()


class ParseTest(unittest.TestCase):
    def test_escaped_pipe_and_br_become_lines(self):
        rows = ft.read_markdown_text(table(
            '| A | lane | | Lane | |',
            '| A-1 | task | A | a \\| b<br>c \\<x\\> | style=highlight |')).rows
        self.assertEqual(rows[1].lines, ['a | b', 'c <x>'])
        self.assertEqual(rows[1].meta, {'style': 'highlight'})

    def test_title_from_first_heading(self):
        self.assertEqual(ft.read_markdown_text('# Tiêu đề\n\n' + table('| A | lane | | Lane | |')).title,
                         'Tiêu đề')

    def test_missing_table_raises(self):
        with self.assertRaises(ft.FlowTableError):
            ft.read_markdown_text('# x\n\nkhông có bảng')

    def test_id_key_orders_numeric_segments(self):
        ids = ['E4.10', 'E4.1', 'E5', 'E4', 'E4.0.1', 'E4.9', 'E4.2']
        self.assertEqual(sorted(ids, key=ft.id_key), ['E4', 'E4.0.1', 'E4.1', 'E4.2', 'E4.9', 'E4.10', 'E5'])


class ValidateTest(unittest.TestCase):
    LANE = '| A | lane | | Lane | |'

    def errors(self, *rows):
        return [i.msg for i in issues_of(table(self.LANE, *rows)) if i.level == 'error']

    def warnings(self, *rows):
        return [i.msg for i in issues_of(table(self.LANE, *rows)) if i.level == 'warning']

    def test_order_example_is_clean(self):
        with open(os.path.join(FIX, 'order.md'), encoding='utf-8') as f:
            self.assertEqual(issues_of(f.read()), [])

    def test_duplicate_id(self):
        self.assertTrue(any('trùng' in m for m in self.errors('| A-1 | start | A | x | |', '| A-1 | end | A | y | |')))

    def test_dangling_reference(self):
        errs = self.errors('| A-1 | start | A | x | |', '| E1 | edge | | | from=A-1; to=A-9 |')
        self.assertTrue(any('to=A-9' in m for m in errs))

    def test_edge_missing_endpoint(self):
        self.assertTrue(any('thiếu to' in m for m in self.errors('| A-1 | start | A | x | |', '| E1 | edge | | | from=A-1 |')))

    def test_db_without_attach(self):
        self.assertTrue(any('db thiếu attach' in m for m in self.errors('| A-1 | db | A | T | |')))

    def test_parent_must_be_lane(self):
        self.assertTrue(any('không phải id của một lane' in m for m in self.errors('| A-1 | task | B | x | |')))

    def test_condition_needs_two_outgoing_edges(self):
        errs = self.errors('| A-1 | condition | A | ? | |', '| E1 | edge | | y | from=A-1; to=A-2 |', '| A-2 | end | A | z | |')
        self.assertTrue(any('cần ít nhất 2' in m for m in errs))

    def test_outgoing_edges_must_follow_source(self):
        errs = self.errors('| A-1 | start | A | x | |', '| A-2 | end | A | y | |', '| E1 | edge | | | from=A-1; to=A-2 |')
        self.assertTrue(any('phải nằm liền sau A-1' in m for m in errs))

    def test_attached_rows_may_sit_between_node_and_edges(self):
        errs = self.errors('| A-1 | start | A | x | |', '| A-2 | text | A | note | attach=A-1 |',
                           '| E1 | edge | | | from=A-1; to=A-3 |', '| A-3 | end | A | y | |')
        self.assertEqual(errs, [])

    def test_warnings(self):
        warns = self.warnings(
            '| A-1 | start | A | x | |', '| E1 | edge | | | from=A-1; to=A-2 |',
            '| A-2 | condition | A | ? | |', '| E2 | edge | | | from=A-2; to=A-1 |', '| E3 | edge | | ok | from=A-2; to=A-3 |',
            '| A-3 | end | A | y | |', '| E4 | edge | | | from=A-3; to=A-4 |',
            '| A-4 | task | A | t | |')
        joined = '\n'.join(warns)
        self.assertIn('không có nhãn', joined)
        self.assertIn('start có cạnh đi vào', joined)
        self.assertIn('end có cạnh đi ra', joined)
        self.assertIn('không ghi back=true', joined)
        self.assertIn('task không có cạnh ra', joined)

    def test_back_edge_suppresses_order_warning(self):
        warns = self.warnings(
            '| A-1 | start | A | x | |', '| E1 | edge | | | from=A-1; to=A-2 |',
            '| A-2 | task | A | t | |', '| E2 | edge | | | from=A-2; to=A-2.1; back=true |')
        self.assertFalse(any('back=true' in m for m in warns))


class TrackTest(unittest.TestCase):
    def test_low_side_stub_gets_lower_track(self):
        lay = ft.Layout([])
        right = lay.seg(('G', 0, 1), 3, 9, ('to', 'X'), [(3, 1)])
        left = lay.seg(('G', 0, 1), 3, 7, ('to', 'Y'), [(3, -1)])
        lay.assign_tracks()
        self.assertLess(left.track, right.track)

    def test_same_target_shares_track(self):
        lay = ft.Layout([])
        a = lay.seg(('C', 4), 1, 5, ('to', 'X'))
        b = lay.seg(('C', 4), 3, 5, ('to', 'X'))
        c = lay.seg(('C', 4), 2, 6, None)
        lay.assign_tracks()
        self.assertEqual(a.track, b.track)
        self.assertNotEqual(a.track, c.track)


class AttachTest(unittest.TestCase):
    def layout(self, *rows):
        t = ft.read_markdown_text(table(*rows))
        assert not [i for i in ft.validate(t.rows) if i.level == 'error']
        return ft.Layout(t.rows).run()

    def gap(self, lay, anchor, att):
        u, a = lay.items[anchor], lay.items[att]
        lo, hi = (u, a) if u.x < a.x else (a, u)
        return hi.x - (lo.x + lo.w)

    def test_db_sits_attach_gap_from_its_anchor(self):
        lay = self.layout(
            '| A | lane | | A | |', '| B | lane | | B | |',
            '| A-1 | start | A | bat dau | |',
            '| E1 | edge | | nhan dai de noi rong mang | from=A-1; to=B-1 |',
            '| B-1 | task | B | xu ly | |',
            '| B-2 | db | B | DB.X | attach=B-1 |',
            '| E2 | edge | | | from=B-1; to=B-3 |',
            '| B-3 | end | B | xong | |')
        self.assertAlmostEqual(self.gap(lay, 'B-1', 'B-2'), lay.cfg.attach_gap)

    def test_attach_gap_is_configurable(self):
        rows = ('| A | lane | | A | |', '| A-1 | start | A | bat dau | |',
                '| A-1.1 | text | A | ghi chu | attach=A-1 |',
                '| E1 | edge | | | from=A-1; to=A-2 |', '| A-2 | end | A | xong | |')
        t = ft.read_markdown_text(table(*rows))
        lay = ft.Layout(t.rows, ft.Config(attach_gap=90)).run()
        self.assertAlmostEqual(self.gap(lay, 'A-1', 'A-1.1'), 90)

    def test_attachment_takes_the_side_the_edge_does_not_use(self):
        lay = self.layout(
            '| A | lane | | A | |', '| B | lane | | B | |',
            '| A-1 | start | A | bat dau | |',
            '| A-1.1 | db | A | DB.X | attach=A-1 |',
            '| E1 | edge | | | from=A-1; to=B-1 |',
            '| B-1 | end | B | xong | |')
        self.assertLess(lay.items['A-1.1'].x, lay.items['A-1'].x)
        self.assertEqual(lay.edges[0].exit_side, 'R')
        self.assertAlmostEqual(self.gap(lay, 'A-1', 'A-1.1'), lay.cfg.attach_gap)
        self.assertFalse([m for lv, m in lay.check() if lv == 'error'])

    def test_edge_leaves_by_the_bottom_when_its_side_holds_an_attachment(self):
        lay = self.layout(
            '| A | lane | | A | |', '| B | lane | | B | |', '| C | lane | | C | |',
            '| A-1 | start | A | bat dau | |',
            '| E1 | edge | | | from=A-1; to=B-1 |',
            '| B-1 | task | B | xu ly | |',
            '| B-2 | db | B | DB.X | attach=B-1 |',
            '| E2 | edge | | | from=B-1; to=C-1 |',
            '| C-1 | end | C | xong | |')
        self.assertEqual(lay.items['B-2'].col, 1)
        self.assertEqual([e for e in lay.edges if e.id == 'E2'][0].exit_side, 'B')
        self.assertAlmostEqual(self.gap(lay, 'B-1', 'B-2'), lay.cfg.attach_gap)
        self.assertFalse([m for lv, m in lay.check() if lv == 'error'])

    def test_only_the_neighbouring_attachment_hugs(self):
        lay = self.layout(
            '| A | lane | | A | |', '| B | lane | | B | |',
            '| A-1 | start | A | bat dau | |',
            '| E1 | edge | | | from=A-1; to=B-1 |',
            '| B-1 | task | B | xu ly | |',
            '| B-2 | db | B | DB.X | attach=B-1 |',
            '| B-3 | db | B | DB.Y | attach=B-1 |',
            '| E2 | edge | | | from=B-1; to=B-4 |',
            '| B-4 | end | B | xong | |')
        self.assertEqual(list(lay.hugs), ['B-2'])
        self.assertAlmostEqual(self.gap(lay, 'B-1', 'B-2'), lay.cfg.attach_gap)
        self.assertFalse([m for lv, m in lay.check() if lv == 'error'])


class BranchSideTest(unittest.TestCase):
    def layout(self, *rows):
        t = ft.read_markdown_text(table(*rows))
        assert not [i for i in ft.validate(t.rows) if i.level == 'error']
        return ft.Layout(t.rows).run()

    def test_branch_goes_to_the_side_its_target_lane_is_on(self):
        """Nhánh phụ dẫn sang lane bên trái thì đặt bên trái, để mũi tên ra khỏi
        nhánh không phải vòng ngược qua node khác."""
        lay = self.layout(
            '| L | lane | | L | |', '| M | lane | | M | |',
            '| M-1 | start | M | bat dau | |',
            '| E1 | edge | | | from=M-1; to=M-2 |',
            '| M-2 | condition | M | chia? | |',
            '| E2 | edge | | No | from=M-2; to=M-3 |',
            '| E3 | edge | | Yes | from=M-2; to=M-4 |',
            '| M-3 | task | M | tra loi | |',
            '| E4 | edge | | | from=M-3; to=L-1 |',
            '| L-1 | end | L | nhan loi | |',
            '| M-4 | end | M | xong | |')
        self.assertLess(lay.items['M-3'].col, lay.items['M-2'].col)
        self.assertFalse([m for lv, m in lay.check() if lv == 'error'])

    def test_branch_with_no_lane_change_keeps_alternating(self):
        lay = self.layout(
            '| M | lane | | M | |',
            '| M-1 | start | M | bat dau | |',
            '| E1 | edge | | | from=M-1; to=M-2 |',
            '| M-2 | condition | M | chia? | |',
            '| E2 | edge | | a | from=M-2; to=M-3 |',
            '| E3 | edge | | b | from=M-2; to=M-4 |',
            '| E4 | edge | | c | from=M-2; to=M-5 |',
            '| M-3 | end | M | mot | |', '| M-4 | end | M | hai | |',
            '| M-5 | end | M | ba | |')
        cols = sorted(lay.items[i].col for i in ('M-3', 'M-4', 'M-5'))
        self.assertEqual(cols, [-1, 0, 1])


class MergeAndRowTest(unittest.TestCase):
    def layout(self, *rows):
        t = ft.read_markdown_text(table(*rows))
        assert not [i for i in ft.validate(t.rows) if i.level == 'error']
        return ft.Layout(t.rows).run()

    def diamond(self):
        return self.layout(
            '| M | lane | | M | |',
            '| M-1 | start | M | bat dau | |',
            '| E1 | edge | | | from=M-1; to=M-2 |',
            '| M-2 | condition | M | chia? | |',
            '| E2 | edge | | No | from=M-2; to=M-3 |',
            '| E3 | edge | | Yes | from=M-2; to=M-4 |',
            '| M-3 | task | M | ghi log | |',
            '| E4 | edge | | | from=M-3; to=M-4 |',
            '| M-4 | end | M | xong | |')

    def test_side_branch_may_share_the_row_of_the_node_it_leaves(self):
        lay = self.diamond()
        self.assertEqual(lay.items['M-3'].row, lay.items['M-2'].row)
        self.assertNotEqual(lay.items['M-3'].col, lay.items['M-2'].col)
        self.assertEqual([e for e in lay.edges if e.id == 'E2'][0].case, 'B')

    def test_merge_node_returns_to_the_column_it_branched_from(self):
        lay = self.diamond()
        self.assertEqual(lay.items['M-4'].col, lay.items['M-2'].col)
        self.assertGreater(lay.items['M-4'].row, lay.items['M-3'].row)
        self.assertFalse([m for lv, m in lay.check() if lv == 'error'])

    def test_merge_without_a_common_branch_point_keeps_the_old_column(self):
        lay = self.layout(
            '| M | lane | | M | |',
            '| M-1 | start | M | mot | |',
            '| E1 | edge | | | from=M-1; to=M-3 |',
            '| M-3 | end | M | chung | |',
            '| T | text | M | Phần còn lại | |',
            '| M-2 | start | M | hai | |',
            '| E2 | edge | | | from=M-2; to=M-3 |')
        self.assertFalse([m for lv, m in lay.check() if lv == 'error'])


class LayoutTest(unittest.TestCase):
    def test_order_placement(self):
        lay = build('order.md')
        cell = {i: (lay.lanes[it.lane].id, it.col, it.row) for i, it in lay.items.items()}
        self.assertEqual(cell['USR-1'][2], cell['API-1'][2])
        self.assertEqual(cell['API-3'], ('API', 0, cell['API-2'][2] + 1))
        self.assertEqual(cell['SVC-1'][2], cell['API-2'][2])
        self.assertEqual(cell['SVC-2'], ('SVC', 1, cell['SVC-1'][2]))
        self.assertGreater(cell['USR-2'][2], cell['SVC-3'][2])
        self.assertEqual(lay.check(), [])

    def test_order_routes(self):
        lay = build('order.md')
        case = {e.id: e.case for e in lay.edges}
        self.assertEqual(case['E1'], 'B')
        self.assertEqual(case['E2'], 'A')
        self.assertEqual(case['E4'], 'B')

    def test_retry_fixture_has_no_geometry_errors(self):
        lay = build('retry.md')
        self.assertEqual([m for lvl, m in lay.check() if lvl == 'error'], [])
        e7 = next(e for e in lay.edges if e.id == 'E7')
        self.assertEqual(e7.case, 'D')
        self.assertLess(e7.pts[-1][1], e7.pts[0][1])
        self.assertGreater(lay.items['BANK-2'].row, lay.items['CLI-2'].row)

    def test_every_path_is_orthogonal_and_ends_on_ports(self):
        for name in ('order.md', 'retry.md'):
            lay = build(name)
            for e in lay.edges:
                for a, b in zip(e.pts, e.pts[1:]):
                    self.assertTrue(abs(a[0] - b[0]) < 0.01 or abs(a[1] - b[1]) < 0.01, (name, e.id))

    def test_output_is_deterministic(self):
        a = ft.to_drawio(build('retry.md'), 't')
        b = ft.to_drawio(build('retry.md'), 't')
        self.assertEqual(a, b)


class CliTest(unittest.TestCase):
    def test_build_writes_next_to_source(self):
        with tempfile.TemporaryDirectory() as d:
            src = os.path.join(d, 'order.md')
            with open(os.path.join(FIX, 'order.md'), encoding='utf-8') as f, open(src, 'w', encoding='utf-8') as g:
                g.write(f.read())
            self.assertEqual(ft.main(['build', src]), 0)
            self.assertTrue(os.path.exists(os.path.join(d, 'order.drawio')))

    def test_build_stops_on_table_error(self):
        with tempfile.TemporaryDirectory() as d:
            src = os.path.join(d, 'bad.md')
            with open(src, 'w', encoding='utf-8') as g:
                g.write(table('| A | lane | | L | |', '| A-1 | db | A | T | |'))
            self.assertEqual(ft.main(['build', src]), 1)
            self.assertFalse(os.path.exists(os.path.join(d, 'bad.drawio')))


if __name__ == '__main__':
    unittest.main()
