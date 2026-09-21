import csv
import os
import shutil
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.dirname(HERE))
sys.path.insert(0, HERE)

import flowtable2drawio as ft  # noqa: E402
from xlsx_fixture import write_xlsx  # noqa: E402

HEADER = ['id', 'type', 'parent', 'content', 'metadata']
BODY = [
    ['A', 'lane', '', 'Lane A', ''],
    ['B', 'lane', '', 'Lane B', ''],
    ['A-1', 'start', 'A', 'Bắt đầu', ''],
    ['E1', 'edge', '', 'gửi', 'from=A-1; to=B-1'],
    ['B-1', 'task', 'B', 'Xử lý', ''],
    ['E2', 'edge', '', '', 'from=B-1; to=B-2'],
    ['B-2', 'end', 'B', 'Xong', ''],
]


class Temp(unittest.TestCase):
    def setUp(self):
        self.dir = tempfile.mkdtemp()
        self.addCleanup(shutil.rmtree, self.dir)

    def path(self, name):
        return os.path.join(self.dir, name)

    def write_csv(self, name, rows, delimiter=',', encoding='utf-8'):
        p = self.path(name)
        with open(p, 'w', encoding=encoding, newline='') as f:
            csv.writer(f, delimiter=delimiter).writerows(rows)
        return p

    def errors(self, table):
        return [i for i in table.issues if i.level == 'error']

    def warnings(self, table):
        return [i.msg for i in table.issues if i.level == 'warning']


class CsvTest(Temp):
    def test_comma_semicolon_tab_are_detected(self):
        for name, d in (('a.csv', ','), ('b.csv', ';'), ('c.csv', '\t')):
            table = ft.load(self.write_csv(name, [HEADER] + BODY, delimiter=d))
            self.assertEqual(self.errors(table), [], name)
            self.assertEqual(len(table.rows), len(BODY), name)

    def test_bom_and_cp1252_fallback(self):
        table = ft.load(self.write_csv('bom.csv', [HEADER] + BODY, encoding='utf-8-sig'))
        self.assertEqual(self.errors(table), [])
        self.assertIn('utf-8-sig', table.source)
        legacy = self.write_csv('legacy.csv', [HEADER] + [['A', 'lane', '', 'Café', '']], encoding='cp1252')
        table = ft.load(legacy)
        self.assertTrue(any('cp1252' in w for w in self.warnings(table)))

    def test_flags_override_detection(self):
        p = self.write_csv('semi.csv', [HEADER] + BODY, delimiter=';')
        with self.assertRaises(ft.FlowTableError):
            ft.load(p, delimiter=',')
        self.assertEqual(self.errors(ft.load(p, delimiter=';', encoding='utf-8')), [])

    def test_notes_above_and_below_table(self):
        rows = [['Luồng thử'], ['người viết: BA'], []] + [HEADER] + BODY + [[], ['ghi chú cuối file']]
        table = ft.load(self.write_csv('notes.csv', rows))
        self.assertEqual(table.title, 'Luồng thử')
        self.assertEqual(len(table.rows), len(BODY))

    def test_title_falls_back_to_filename(self):
        table = ft.load(self.write_csv('ten-file.csv', [HEADER] + BODY))
        self.assertEqual(table.title, 'ten-file')

    def test_newline_and_br_both_split_lines(self):
        rows = [HEADER, ['A', 'lane', '', 'Lane A', ''], ['A-1', 'task', 'A', 'dòng 1\ndòng 2<br>dòng 3', '']]
        table = ft.load(self.write_csv('multi.csv', rows))
        self.assertEqual(table.rows[1].lines, ['dòng 1', 'dòng 2', 'dòng 3'])

    def test_pipe_is_literal_and_markdown_escape_warns(self):
        rows = [HEADER, ['A', 'lane', '', 'Lane A', ''], ['A-1', 'task', 'A', 'a | b', ''],
                ['A-2', 'task', 'A', r'c \| d', '']]
        table = ft.load(self.write_csv('pipe.csv', rows))
        self.assertEqual(table.rows[1].lines, ['a | b'])
        self.assertEqual(table.rows[2].lines, [r'c \| d'])
        self.assertTrue(any('escape kiểu markdown' in w for w in self.warnings(table)))

    def test_unsupported_extension(self):
        p = self.path('x.json')
        open(p, 'w').close()
        with self.assertRaises(ft.FlowTableError):
            ft.load(p)

    def test_missing_file_message(self):
        with self.assertRaises(ft.FlowTableError):
            ft.load(self.path('khong-co.csv'))

    def test_build_writes_drawio_next_to_csv(self):
        p = self.write_csv('flow.csv', [HEADER] + BODY)
        self.assertEqual(ft.main(['build', p]), 0)
        self.assertTrue(os.path.exists(self.path('flow.drawio')))


def rows_at(start, table_rows):
    return [(start + i, r) for i, r in enumerate(table_rows)]


class XlsxTest(Temp):
    def build_book(self, name='flow.xlsx', top=(), hidden=(), merges=(), extra_sheet=True, body=None):
        body = body or BODY
        rows = list(top) + rows_at(len(top) + 1, [HEADER] + body)
        sheets = []
        if extra_sheet:
            sheets.append(('Huong dan', [(1, ['Sheet hướng dẫn, không có bảng'])], (), ()))
        sheets.append(('Flow', rows, hidden, merges))
        p = self.path(name)
        write_xlsx(p, sheets)
        return p

    def test_first_sheet_with_header_is_used(self):
        table = ft.load(self.build_book())
        self.assertEqual(table.source, 'xlsx sheet Flow')
        self.assertEqual(self.errors(table), [])
        self.assertEqual([r.id for r in table.rows], [r[0] for r in BODY])

    def test_sheet_flag_selects_sheet(self):
        p = self.build_book()
        with self.assertRaises(ft.FlowTableError):
            ft.load(p, sheet='Huong dan')
        with self.assertRaises(ft.FlowTableError):
            ft.load(p, sheet='Khong ton tai')
        self.assertEqual(ft.load(p, sheet='Flow').source, 'xlsx sheet Flow')

    def test_title_from_cell_above_header(self):
        table = ft.load(self.build_book(top=[(1, ['Luồng xlsx']), (2, [])]))
        self.assertEqual(table.title, 'Luồng xlsx')

    def test_shared_strings_and_extra_column(self):
        body = [list(r) for r in BODY]
        body[2] = [('A-1', 'shared'), ('start', 'shared'), 'A', ('Bắt đầu', 'shared'), '']
        body[0] = body[0] + ['ghi chú riêng']
        table = ft.load(self.build_book(body=body))
        self.assertEqual(self.errors(table), [])
        self.assertEqual(table.rows[2].id, 'A-1')
        self.assertEqual(table.rows[2].lines, ['Bắt đầu'])

    def test_numeric_cells_are_trimmed_and_id_warns(self):
        body = [list(r) for r in BODY]
        body.append([('4.0', 'num'), 'task', 'A', ('12.0', 'num'), ''])
        body.append(['A-9', 'task', 'A', 'x', ''])
        table = ft.load(self.build_book(body=body))
        row = table.rows[len(BODY)]
        self.assertEqual(row.id, '4')
        self.assertEqual(row.lines, ['12'])
        self.assertTrue(any('ô kiểu số' in w for w in self.warnings(table)))

    def test_formula_cells(self):
        body = [list(r) for r in BODY]
        body.append(['A-8', 'task', 'A', ('Giá trị đã tính', 'formula'), ''])
        body.append(['A-9', 'task', 'A', ('', 'formula_empty'), ''])
        table = ft.load(self.build_book(body=body))
        self.assertEqual(table.rows[len(BODY)].lines, ['Giá trị đã tính'])
        self.assertTrue(any('chưa có giá trị lưu sẵn' in w for w in self.warnings(table)))

    def test_hidden_row_inside_table_warns(self):
        table = ft.load(self.build_book(hidden=(4,)))
        self.assertTrue(any('đang bị ẩn' in w for w in self.warnings(table)))
        self.assertEqual(len(table.rows), len(BODY))

    def test_merged_cell_warns_only_inside_table(self):
        outside = ft.load(self.build_book(top=[(1, ['Tiêu đề']), (2, [])], merges=('A1:C1',)))
        self.assertFalse(any('Ô gộp' in w or 'ô gộp' in w for w in self.warnings(outside)))
        inside = ft.load(self.build_book(merges=('A3:B3',)))
        self.assertTrue(any('ô gộp' in w for w in self.warnings(inside)))

    def test_table_stops_at_empty_row(self):
        rows = rows_at(1, [HEADER] + BODY) + [(len(BODY) + 3, ['ghi chú dưới bảng'])]
        p = self.path('gap.xlsx')
        write_xlsx(p, [('Flow', rows, (), ())])
        self.assertEqual(len(ft.load(p).rows), len(BODY))

    def test_build_from_xlsx(self):
        p = self.build_book()
        self.assertEqual(ft.main(['build', p]), 0)
        self.assertTrue(os.path.exists(self.path('flow.drawio')))


class SameResultTest(Temp):
    """Cùng nội dung bảng thì ba định dạng cho ra cùng một layout."""

    def test_md_csv_xlsx_match(self):
        md = ['# Luồng thử', '', '| ' + ' | '.join(HEADER) + ' |', '|---|---|---|---|---|']
        md += ['| ' + ' | '.join(r) + ' |' for r in BODY]
        p_md = self.path('flow.md')
        with open(p_md, 'w', encoding='utf-8') as f:
            f.write('\n'.join(md) + '\n')
        p_csv = self.write_csv('flow.csv', [['Luồng thử'], []] + [HEADER] + BODY)
        p_xlsx = self.path('flow.xlsx')
        write_xlsx(p_xlsx, [('Flow', [(1, ['Luồng thử']), (2, [])] + rows_at(3, [HEADER] + BODY), (), ())])
        out = []
        for p in (p_md, p_csv, p_xlsx):
            table = ft.load(p)
            self.assertEqual(table.title, 'Luồng thử', p)
            out.append(ft.to_drawio(ft.Layout(table.rows).run(), table.title))
        self.assertEqual(out[0], out[1])
        self.assertEqual(out[0], out[2])


if __name__ == '__main__':
    unittest.main()
