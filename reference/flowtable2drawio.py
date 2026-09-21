#!/usr/bin/env python3
"""Dựng file draw.io kiểu activity-swimlane từ một Flow Table.

Bảng đọc được từ .md, .csv hoặc .xlsx; xem docs/input-formats-design.md.
Định dạng bảng: flow-table-format.md cạnh file này.
Toàn bộ toạ độ (ô lưới, kênh đi dây, điểm gấp, vị trí nhãn) được tính tất định
trong file này; cùng một bảng luôn ra cùng một file.

Mô hình layout:
- Mỗi lane là một cột dọc, chia thành các cột con (col, số nguyên, có thể âm).
- Luồng đi từ trên xuống theo hàng (row). Mỗi phần tử chiếm đúng một ô.
- Giữa hai hàng là kênh ngang (channel), giữa hai cột con và ở hai mép lane là
  máng dọc (gutter). Dây nào không đi thẳng được thì chỉ chạy trong kênh/máng,
  mỗi đoạn một track riêng, nên không cắt qua ô có phần tử.

Generate lại khi file đích đã có: --mode force sinh lại toàn bộ; --mode merge giữ
vị trí element, lane và waypoint của edge từ file đang có (xem docs/merge-design.md).

Exit code: 0 ổn, 1 bảng có lỗi, 2 layout tự kiểm có lỗi, 3 ảnh render lệch toạ độ,
4 file đích đã có mà chưa chọn chế độ hoặc không đọc được.
"""
import argparse
import base64
import csv
import copy
import heapq
import io
import json
import math
import os
import re
import shutil
import struct
import subprocess
import sys
import tempfile
import unicodedata
import urllib.parse
import xml.etree.ElementTree as ET
import zipfile
import zlib
from collections import defaultdict
from dataclasses import dataclass, field

NODE_TYPES = {'start', 'end', 'task', 'condition', 'external'}
ATTACH_TYPES = {'db', 'text'}
ALL_TYPES = {'lane', 'edge'} | NODE_TYPES | ATTACH_TYPES
META_KEYS = {
    'lane': set(),
    'edge': {'from', 'to', 'style', 'back'},
    'db': {'attach', 'style'},
    'text': {'attach', 'style'},
}
STYLE_VALUES = {'edge': {'highlight', 'dashed'}}
REST_MARKER = 'Phần còn lại'
RESERVED_IDS = {'0', '1', 'pool'}
HEADER = ['id', 'type', 'parent', 'content', 'metadata']

METRICS_PATH = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'data', 'verdana.json')

FONT_PATHS = [
    '/usr/share/fonts/truetype/msttcorefonts/Verdana.ttf',
    '/usr/share/fonts/truetype/msttcorefonts/verdana.ttf',
    '/Library/Fonts/Verdana.ttf',
    '/System/Library/Fonts/Supplemental/Verdana.ttf',
    'C:/Windows/Fonts/verdana.ttf',
]

HIGHLIGHT_NODE = 'fillColor=#dae8fc;strokeColor=#6c8ebf;'
HIGHLIGHT_TEXT = 'fillColor=#dae8fc;strokeColor=none;'
HIGHLIGHT_EDGE = 'strokeColor=#6c8ebf;strokeWidth=2;'
FONT = 'fontFamily=Verdana;fontSize=12;'
# Dấu nhận biết cell do tool sinh; merge dùng nó để tách cell người dùng tự vẽ.
MARK = 'flowtable=1;'
# Không bật whiteSpace=wrap: chữ đã được ngắt dòng sẵn bằng <br>, để draw.io tự
# ngắt lại thì số dòng có thể khác kích thước node đã tính.
SHAPE_STYLE = {
    'task': 'rounded=0;html=1;',
    'condition': 'rhombus;html=1;',
    'start': 'ellipse;html=1;',
    'end': 'shape=doubleEllipse;html=1;',
    'external': 'ellipse;html=1;dashed=1;',
    'db': 'shape=cylinder3;html=1;boundedLbl=1;backgroundOutline=1;size=8;',
    'text': 'text;html=1;align=left;verticalAlign=middle;spacingLeft=4;',
}
# Hình có chu vi cong/chéo: cổng lệch 0.5 sẽ bị draw.io chiếu lại lên chu vi,
# nên mỗi mặt chỉ dùng đúng điểm giữa.
NON_RECT = {'condition', 'start', 'end', 'external'}


class FlowTableError(Exception):
    pass


@dataclass
class Issue:
    level: str
    loc: str
    id: str
    msg: str

    def __str__(self):
        tag = f' [{self.id}]' if self.id else ''
        return f'{self.level.upper():7} {self.loc or "-"}{tag}: {self.msg}'


@dataclass
class Row:
    idx: int
    loc: str
    id: str
    type: str
    parent: str
    lines: list
    meta: dict

    @property
    def text(self):
        return '\n'.join(self.lines).strip()

    def styles(self):
        return set(self.style_list())

    def style_list(self):
        """Giá trị style theo đúng thứ tự viết trong ô, đã bỏ trùng.

        styles() trả về set nên duyệt nó cho ra thứ tự phụ thuộc hash, tức là
        thứ tự cảnh báo đổi giữa hai lần chạy. Chỗ nào cần duyệt thì dùng hàm
        này; chỗ nào chỉ hỏi có hay không thì dùng styles().
        """
        out = []
        for s in self.meta.get('style', '').split(','):
            s = s.strip()
            if s and s not in out:
                out.append(s)
        return out

    def is_marker(self):
        return self.type == 'text' and self.text == REST_MARKER and not self.meta.get('attach')


# ---------------------------------------------------------------- parse

BR_RE = re.compile(r'<br\s*/?>', re.I)
MD_ESC_RE = re.compile(r'\\([\\`*_{}\[\]()#+\-.!|<>~])')
MD_PIPE_RE = re.compile(r'\\\|')
CSV_DELIMS = (',', ';', '\t')
CSV_ENCODINGS = ('utf-8-sig', 'utf-8', 'cp1252')
NS_MAIN = '{http://schemas.openxmlformats.org/spreadsheetml/2006/main}'
NS_REL = '{http://schemas.openxmlformats.org/officeDocument/2006/relationships}'
NS_PKG_REL = '{http://schemas.openxmlformats.org/package/2006/relationships}'


@dataclass
class Table:
    title: str
    rows: list
    issues: list
    source: str


def split_cells(line):
    s = line.strip()
    if s.startswith('|'):
        s = s[1:]
    if s.endswith('|') and not s.endswith('\\|'):
        s = s[:-1]
    return [c.strip() for c in re.split(r'(?<!\\)\|', s)]


def unescape(s):
    return unicodedata.normalize('NFC', MD_ESC_RE.sub(r'\1', s))


def md_lines(cell):
    return [unescape(p) for p in BR_RE.split(cell)] if cell else ['']


def plain_lines(cell):
    """csv/xlsx: xuống dòng thật và thẻ br đều là xuống dòng, không xử lý escape."""
    if not cell:
        return ['']
    parts = []
    for chunk in BR_RE.split(cell):
        parts.extend(chunk.replace('\r\n', '\n').replace('\r', '\n').split('\n'))
    return [unicodedata.normalize('NFC', p.strip()) for p in parts]


def parse_meta(cell, loc, rid, issues, unesc):
    meta = {}
    for part in (cell or '').split(';'):
        part = part.strip()
        if not part:
            continue
        if '=' not in part:
            issues.append(Issue('error', loc, rid, f'metadata sai cú pháp: "{part}" (cần key=value)'))
            continue
        k, v = part.split('=', 1)
        meta[k.strip()] = unesc(v.strip())
    return meta


def find_header(grid):
    """(hàng, cột) của hàng header đầu tiên có đủ 5 tên cột liền nhau."""
    for r, row in enumerate(grid):
        cells = [str(c or '').strip().lower() for c in row]
        for c0 in range(max(1, len(cells) - 4)):
            if cells[c0:c0 + 5] == HEADER:
                return r, c0
    return None


def table_from_grid(grid, locs, split_fn):
    """Bảng nằm dưới hàng header, kết thúc ở hàng đầu tiên có cả 5 ô đều rỗng."""
    hit = find_header(grid)
    if hit is None:
        raise FlowTableError('không tìm thấy hàng header: ' + ' | '.join(HEADER))
    hr, hc = hit
    title = None
    for r in range(hr):
        for c in grid[r]:
            if str(c or '').strip():
                title = unicodedata.normalize('NFC', str(c).strip())
                break
        if title:
            break
    rows, issues = [], []
    for r in range(hr + 1, len(grid)):
        row = grid[r]
        cells = [str(row[hc + i] or '').strip() if hc + i < len(row) else '' for i in range(5)]
        if not any(cells):
            break
        rid, rtype, parent, content, meta_s = cells
        if MD_PIPE_RE.search(content) or MD_PIPE_RE.search(meta_s):
            issues.append(Issue('warning', locs[r], rid,
                                'nội dung còn escape kiểu markdown (\\|); csv/xlsx không cần escape'))
        meta = parse_meta(meta_s, locs[r], rid, issues, lambda v: unicodedata.normalize('NFC', v))
        rows.append(Row(len(rows), locs[r], rid, rtype.lower(), parent, split_fn(content), meta))
    return title, rows, issues


def read_markdown(path):
    with open(path, encoding='utf-8') as f:
        return read_markdown_text(f.read())


def read_markdown_text(text):
    lines = text.splitlines()
    title = None
    for ln in lines:
        m = re.match(r'^#\s+(.+?)\s*$', ln)
        if m:
            title = unescape(m.group(1))
            break
    start = None
    for i, ln in enumerate(lines):
        if ln.lstrip().startswith('|') and [c.lower() for c in split_cells(ln)] == HEADER:
            start = i
            break
    if start is None:
        raise FlowTableError('không tìm thấy bảng có header: ' + ' | '.join(HEADER))
    if start + 1 >= len(lines) or not re.match(r'^\s*\|[\s:\-|]+\|?\s*$', lines[start + 1]):
        raise FlowTableError(f'dòng {start + 2}: thiếu dòng phân cách |---| sau header')
    rows, issues = [], []
    i = start + 2
    while i < len(lines) and lines[i].lstrip().startswith('|'):
        loc = f'dòng {i + 1}'
        cells = split_cells(lines[i])
        i += 1
        if len(cells) != 5:
            issues.append(Issue('error', loc, '', f'dòng có {len(cells)} cột, cần 5 '
                                                  '(gạch đứng trong nội dung phải viết \\|)'))
            continue
        rid, rtype, parent, content, meta_s = cells
        meta = parse_meta(meta_s, loc, rid, issues, unescape)
        rows.append(Row(len(rows), loc, rid, rtype.lower(), parent, md_lines(content), meta))
    return Table(title, rows, issues, 'markdown')


def read_csv(path, delimiter=None, encoding=None):
    with open(path, 'rb') as f:
        raw = f.read()
    text = enc_used = None
    for enc in ([encoding] if encoding else CSV_ENCODINGS):
        try:
            text = raw.decode(enc)
            enc_used = enc
            break
        except (UnicodeDecodeError, LookupError):
            continue
    if text is None:
        raise FlowTableError(f'không giải mã được {path}; thử --encoding')
    grid = None
    for d in ([delimiter] if delimiter else CSV_DELIMS):
        cand = list(csv.reader(io.StringIO(text, newline=''), delimiter=d))
        if find_header(cand) is not None:
            grid, delimiter = cand, d
            break
    if grid is None:
        raise FlowTableError('không tìm thấy hàng header trong csv; kiểm tra dấu phân cách, hoặc dùng --delimiter')
    locs = [f'dòng {i + 1}' for i in range(len(grid))]
    title, rows, issues = table_from_grid(grid, locs, plain_lines)
    if enc_used == 'cp1252':
        issues.append(Issue('warning', '', '', 'file không phải utf-8, đã đọc theo cp1252; '
                                              'chữ tiếng Việt có thể đã hỏng từ trước'))
    shown = {',': 'dấu phẩy', ';': 'dấu chấm phẩy', '\t': 'tab'}.get(delimiter, delimiter)
    return Table(title, rows, issues, f'csv ({shown}, {enc_used})')


def col_index(ref):
    n = 0
    for ch in ref:
        if not ch.isalpha():
            break
        n = n * 26 + (ord(ch.upper()) - 64)
    return n - 1


def read_xlsx(path, sheet=None):
    try:
        zf = zipfile.ZipFile(path)
    except (zipfile.BadZipFile, OSError) as ex:
        raise FlowTableError(f'không đọc được {path}: {ex}')
    with zf:
        shared = []
        if 'xl/sharedStrings.xml' in zf.namelist():
            for si in ET.fromstring(zf.read('xl/sharedStrings.xml')):
                shared.append(''.join(t.text or '' for t in si.iter(NS_MAIN + 't')))
        rels = {}
        for rel in ET.fromstring(zf.read('xl/_rels/workbook.xml.rels')):
            rels[rel.get('Id')] = rel.get('Target')
        sheets = []
        for sh in ET.fromstring(zf.read('xl/workbook.xml')).iter(NS_MAIN + 'sheet'):
            target = rels.get(sh.get(NS_REL + 'id'), '')
            target = target[1:] if target.startswith('/xl/') else 'xl/' + target.lstrip('/')
            sheets.append((sh.get('name'), target))
        if sheet:
            sheets = [x for x in sheets if x[0] == sheet]
            if not sheets:
                raise FlowTableError(f'{path} không có sheet tên "{sheet}"')
        for name, target in sheets:
            try:
                data = zf.read(target)
            except KeyError:
                continue
            grid, locs, notes = xlsx_grid(ET.fromstring(data), shared, name)
            if find_header(grid) is None:
                continue
            title, rows, issues = table_from_grid(grid, locs, plain_lines)
            hr = find_header(grid)[0] + 1
            issues += [i for i in notes if hr <= i.row <= hr + len(rows)]
            return Table(title, rows, issues, f'xlsx sheet {name}')
    raise FlowTableError('không tìm thấy sheet nào có hàng header: ' + ' | '.join(HEADER))


@dataclass
class SheetIssue(Issue):
    row: int = 0


def xlsx_grid(sheet_el, shared, name):
    """Lưới ô của một sheet, kèm cảnh báo về ô công thức rỗng, ô gộp và hàng ẩn."""
    grid, locs, notes = [], [], []
    rows = {}
    for row_el in sheet_el.iter(NS_MAIN + 'row'):
        r = int(row_el.get('r', len(rows) + 1))
        cells = {}
        for c in row_el.iter(NS_MAIN + 'c'):
            ref = c.get('r', '')
            ci = col_index(ref) if ref else len(cells)
            t = c.get('t')
            v = c.find(NS_MAIN + 'v')
            if t == 's':
                val = shared[int(v.text)] if v is not None and v.text else ''
            elif t == 'inlineStr':
                is_el = c.find(NS_MAIN + 'is')
                val = ''.join(x.text or '' for x in is_el.iter(NS_MAIN + 't')) if is_el is not None else ''
            elif v is not None:
                val = v.text or ''
                if t in (None, 'n') and re.fullmatch(r'-?\d+\.0+', val):
                    val = val.split('.')[0]
            else:
                val = ''
                if c.find(NS_MAIN + 'f') is not None:
                    notes.append(SheetIssue('warning', f'{name}!{ref}', '',
                                            'ô công thức chưa có giá trị lưu sẵn, đọc thành rỗng', row=r))
            if val and t in (None, 'n') and ci in (0, 2):
                notes.append(SheetIssue('warning', f'{name}!{ref}', val,
                                        'ô kiểu số ở cột id/parent; định dạng cột là Text để id như 4.10 không bị đổi',
                                        row=r))
            cells[ci] = val
        rows[r] = cells
        if row_el.get('hidden') == '1':
            notes.append(SheetIssue('warning', f'{name} dòng {r}', '', 'hàng đang bị ẩn nhưng vẫn được đọc', row=r))
    for merge in sheet_el.iter(NS_MAIN + 'mergeCell'):
        ref = merge.get('ref', '')
        first = ref.split(':')[0]
        r = int(re.sub(r'\D', '', first) or 0)
        notes.append(SheetIssue('warning', f'{name}!{ref}', '',
                                'ô gộp: chỉ ô trên cùng bên trái giữ giá trị', row=r))
    top = max(rows) if rows else 0
    for r in range(1, top + 1):
        cells = rows.get(r, {})
        width = max(cells) + 1 if cells else 0
        grid.append([cells.get(i, '') for i in range(width)])
        locs.append(f'{name} dòng {r}')
    return grid, locs, notes


def load(path, sheet=None, delimiter=None, encoding=None):
    ext = os.path.splitext(path)[1].lower()
    try:
        if ext in ('.md', '.markdown', '.txt'):
            table = read_markdown(path)
        elif ext in ('.csv', '.tsv'):
            table = read_csv(path, delimiter, encoding)
        elif ext in ('.xlsx', '.xlsm'):
            table = read_xlsx(path, sheet)
        else:
            raise FlowTableError(f'đuôi file "{ext}" không hỗ trợ; dùng .md, .csv hoặc .xlsx')
    except OSError as ex:
        raise FlowTableError(f'không đọc được {path}: {ex.strerror or ex}')
    if not table.title:
        table.title = os.path.splitext(os.path.basename(path))[0]
    table.issues = list(table.issues) + validate(table.rows)
    return table


def id_key(s):
    """E4 < E4.0.1 < E4.1 < E4.2 < E4.10: so theo từng đoạn số."""
    m = re.match(r'^(.*?)(\d+(?:\.\d+)*)$', s)
    if not m:
        return (s, ())
    return (m.group(1), tuple(int(p) for p in m.group(2).split('.')))


# ---------------------------------------------------------------- validate

def validate(rows):
    issues = []

    def err(r, msg):
        issues.append(Issue('error', r.loc, r.id, msg))

    def warn(r, msg):
        issues.append(Issue('warning', r.loc, r.id, msg))

    by_id = {}
    for r in rows:
        if not r.id:
            err(r, 'id trống')
            continue
        if r.id in RESERVED_IDS:
            err(r, f'id "{r.id}" trùng id dành riêng của draw.io')
        if r.id in by_id:
            err(r, f'id trùng với {by_id[r.id].loc}')
        else:
            by_id[r.id] = r
        if r.type not in ALL_TYPES:
            err(r, f'type không hợp lệ: "{r.type}"')
    lanes = {r.id for r in rows if r.type == 'lane'}
    if not lanes:
        issues.append(Issue('error', '', '', 'bảng không có lane nào'))

    def is_node(i):
        return i in by_id and by_id[i].type in NODE_TYPES

    for r in rows:
        if r.type not in ALL_TYPES:
            continue
        if r.type in ('lane', 'edge'):
            if r.parent:
                err(r, f'{r.type} phải để trống parent')
        elif not r.is_marker():
            if not r.parent:
                err(r, 'thiếu parent (id của lane chứa phần tử)')
            elif r.parent not in lanes:
                err(r, f'parent "{r.parent}" không phải id của một lane')
        allowed = META_KEYS.get(r.type, {'style'})
        for k in r.meta:
            if k not in allowed:
                warn(r, f'metadata "{k}" không dùng cho type {r.type}, bị bỏ qua')
        valid_styles = STYLE_VALUES.get(r.type, {'highlight'})
        for s in r.style_list():
            if s not in valid_styles:
                warn(r, f'style "{s}" không dùng cho type {r.type}, bị bỏ qua')
        if 'back' in r.meta and r.meta['back'] not in ('true', 'false'):
            warn(r, f'back="{r.meta["back"]}" không hợp lệ, chỉ nhận true/false')
        if r.type == 'edge':
            for k in ('from', 'to'):
                v = r.meta.get(k)
                if not v:
                    err(r, f'edge thiếu {k}')
                elif v not in by_id:
                    err(r, f'{k}={v} trỏ tới id không tồn tại')
                elif not is_node(v):
                    err(r, f'{k}={v} là {by_id[v].type}; cạnh chỉ nối start/end/task/condition/external')
        if r.type == 'db' and not r.meta.get('attach'):
            err(r, 'db thiếu attach')
        if r.type in ATTACH_TYPES and r.meta.get('attach'):
            a = r.meta['attach']
            if a not in by_id:
                err(r, f'attach={a} trỏ tới id không tồn tại')
            elif not is_node(a):
                err(r, f'attach={a} là {by_id[a].type}; chỉ gắn được vào start/end/task/condition/external')
            elif by_id[a].parent != r.parent:
                warn(r, f'nằm ở lane {r.parent} nhưng gắn vào {a} thuộc lane {by_id[a].parent}; sẽ vẽ trong lane {by_id[a].parent}')

    edges = [r for r in rows if r.type == 'edge' and is_node(r.meta.get('from')) and is_node(r.meta.get('to'))]
    outs, ins = defaultdict(list), defaultdict(list)
    for e in edges:
        outs[e.meta['from']].append(e)
        ins[e.meta['to']].append(e)
    pos = {r.id: i for i, r in enumerate(rows) if r.id}
    for r in rows:
        if r.type not in NODE_TYPES or by_id.get(r.id) is not r:
            continue
        if r.type == 'condition' and len(outs[r.id]) < 2:
            err(r, f'condition chỉ có {len(outs[r.id])} cạnh ra, cần ít nhất 2')
        if r.type in ('task', 'condition') and not outs[r.id]:
            warn(r, f'{r.type} không có cạnh ra')
        if r.type == 'start' and ins[r.id]:
            warn(r, 'start có cạnh đi vào: ' + ', '.join(e.id for e in ins[r.id]))
        if r.type == 'end' and outs[r.id]:
            warn(r, 'end có cạnh đi ra: ' + ', '.join(e.id for e in outs[r.id]))
        if r.type == 'condition':
            for e in outs[r.id]:
                if not e.text:
                    warn(e, f'cạnh ra của condition {r.id} không có nhãn')
        es = outs[r.id]
        if es:
            p = pos[r.id] + 1
            while p < len(rows) and rows[p].type in ATTACH_TYPES and rows[p].meta.get('attach') == r.id:
                p += 1
            block = {b.id for b in rows[p:p + len(es)]}
            if block != {e.id for e in es}:
                misplaced = [e.id for e in es if e.id not in block]
                err(r, f'cạnh ra phải nằm liền sau {r.id} (và các db/text gắn vào nó); đang lệch: ' + ', '.join(misplaced))
    for e in edges:
        if e.meta.get('back') != 'true' and pos[e.meta['to']] < pos[e.meta['from']]:
            warn(e, f'{e.meta["to"]} đứng trước nguồn {e.meta["from"]} nhưng cạnh không ghi back=true')
    return issues


# ---------------------------------------------------------------- text metrics

class TTFMetrics:
    """Độ rộng glyph đọc thẳng từ bảng cmap (format 4) và hmtx của file TTF."""

    def __init__(self, path):
        with open(path, 'rb') as f:
            data = f.read()
        num = struct.unpack('>H', data[4:6])[0]
        tables = {}
        for i in range(num):
            tag, _, off, ln = struct.unpack('>4sIII', data[12 + 16 * i:28 + 16 * i])
            tables[tag.decode('latin-1')] = off
        head = tables['head']
        self.upem = struct.unpack('>H', data[head + 18:head + 20])[0]
        hhea = tables['hhea']
        nhm = struct.unpack('>H', data[hhea + 34:hhea + 36])[0]
        hmtx = tables['hmtx']
        self.adv = [struct.unpack('>H', data[hmtx + 4 * i:hmtx + 4 * i + 2])[0] for i in range(nhm)]
        self.cmap = self._cmap4(data, tables['cmap'])

    @staticmethod
    def _cmap4(data, base):
        n = struct.unpack('>H', data[base + 2:base + 4])[0]
        sub = None
        for i in range(n):
            pid, eid, off = struct.unpack('>HHI', data[base + 4 + 8 * i:base + 12 + 8 * i])
            if pid == 3 and eid in (1, 10) and struct.unpack('>H', data[base + off:base + off + 2])[0] == 4:
                sub = base + off
                break
        if sub is None:
            raise ValueError('font không có cmap format 4')
        segx2 = struct.unpack('>H', data[sub + 6:sub + 8])[0]
        seg = segx2 // 2
        ends = struct.unpack(f'>{seg}H', data[sub + 14:sub + 14 + segx2])
        so = sub + 16 + segx2
        starts = struct.unpack(f'>{seg}H', data[so:so + segx2])
        deltas = struct.unpack(f'>{seg}h', data[so + segx2:so + 2 * segx2])
        ro = so + 2 * segx2
        ranges = struct.unpack(f'>{seg}H', data[ro:ro + segx2])
        cmap = {}
        for k in range(seg):
            for c in range(starts[k], ends[k] + 1):
                if c == 0xFFFF:
                    continue
                if ranges[k] == 0:
                    g = (c + deltas[k]) & 0xFFFF
                else:
                    p = ro + 2 * k + ranges[k] + 2 * (c - starts[k])
                    g = struct.unpack('>H', data[p:p + 2])[0]
                    if g:
                        g = (g + deltas[k]) & 0xFFFF
                cmap[c] = g
        return cmap

    def width(self, s, size):
        total = 0
        for ch in s:
            g = self.cmap.get(ord(ch), 0)
            total += self.adv[g] if g < len(self.adv) else self.adv[-1]
        return total * size / self.upem


class TableMetrics:
    """Độ rộng glyph đọc từ bảng đã trích sẵn thay vì từ file font trên máy.

    Cùng một bảng thì mọi máy đo chữ giống nhau, kể cả máy không cài Verdana.
    Bảng sinh bằng tools/extract_metrics.py.
    """

    def __init__(self, path):
        with open(path, encoding='utf-8') as f:
            d = json.load(f)
        self.upem = d['upem']
        self.notdef = d['notdef']
        self.adv = {int(k): v for k, v in d['advances'].items()}

    def width(self, s, size):
        return sum(self.adv.get(ord(c), self.notdef) for c in s) * size / self.upem


class TextMeasure:
    BREAK_RE = re.compile(r'(?<=[/._,(){}=&?-])|(?<=::)')

    def __init__(self, size=12, line_h=15, font_path=None, metrics_path=None):
        self.size = size
        self.line_h = line_h
        self.font = None
        self.hard = False
        mp = metrics_path if metrics_path is not None else METRICS_PATH
        if mp and os.path.exists(mp) and not font_path:
            self.font = TableMetrics(mp)
            self.font_path = mp
            return
        for p in ([font_path] if font_path else []) + FONT_PATHS:
            if p and os.path.exists(p):
                try:
                    self.font = TTFMetrics(p)
                    self.font_path = p
                    break
                except (ValueError, KeyError, struct.error):
                    continue

    def w(self, s):
        if self.font:
            return self.font.width(s, self.size)
        # Ước lượng cho Verdana khi không có file font.
        return sum(4.2 if c == ' ' else 8.2 if (c.isupper() or c.isdigit()) else 7.0 for c in s) * self.size / 12

    def _split_long(self, word, maxw):
        chunks = [c for c in self.BREAK_RE.split(word) if c]
        out, cur = [], ''
        for c in chunks:
            if self.w(cur + c) <= maxw:
                cur += c
                continue
            if cur:
                out.append(cur)
            cur = ''
            while self.w(c) > maxw:
                self.hard = True
                k = len(c)
                while k > 1 and self.w(c[:k]) > maxw:
                    k -= 1
                out.append(c[:k])
                c = c[k:]
            cur = c
        if cur:
            out.append(cur)
        return out

    def wrap(self, lines, maxw):
        """Ngắt dòng rồi thu hẹp bề rộng tới mức nhỏ nhất vẫn giữ nguyên số dòng, để các dòng đều nhau."""
        out = []
        for line in lines:
            self.hard = False
            first = self._wrap_line(line, maxw)
            if len(first) == 1 or self.hard:
                out.extend(first)
                continue
            lo, hi = 1, int(math.ceil(maxw))
            while lo < hi:
                mid = (lo + hi) // 2
                self.hard = False
                if len(self._wrap_line(line, mid)) <= len(first) and not self.hard:
                    hi = mid
                else:
                    lo = mid + 1
            out.extend(self._wrap_line(line, hi))
        return out

    def _wrap_line(self, line, maxw):
        lines = [line]
        out = []
        for line in lines:
            line = line.strip()
            if self.w(line) <= maxw:
                out.append(line)
                continue
            cur = ''
            for word in line.split():
                cand = f'{cur} {word}' if cur else word
                if self.w(cand) <= maxw:
                    cur = cand
                    continue
                if cur:
                    out.append(cur)
                if self.w(word) <= maxw:
                    cur = word
                else:
                    parts = self._split_long(word, maxw)
                    out.extend(parts[:-1])
                    cur = parts[-1]
            out.append(cur)
        return out

    def box(self, lines):
        return (max((self.w(l) for l in lines), default=0), len(lines) * self.line_h)


# ---------------------------------------------------------------- model

@dataclass
class Config:
    task_min_w: int = 120
    task_max_w: int = 240
    cond_wrap: int = 150
    term_wrap: int = 170
    db_wrap: int = 130
    text_wrap: int = 260
    label_wrap: int = 180
    track_gap: int = 12
    gutter_margin: int = 15
    channel_margin: int = 12
    min_gutter: int = 24
    min_channel: int = 30
    attach_gap: int = 40
    lane_header: int = 30
    pool_header: int = 30
    min_lane_w: int = 120
    label_pad: int = 4


@dataclass
class Item:
    id: str
    kind: str
    lane: int
    lines: list
    w: float
    h: float
    order: int
    highlight: bool = False
    attach: str = None
    row: int = None
    col: int = None
    x: float = 0.0
    y: float = 0.0

    def box(self):
        return (self.x, self.y, self.x + self.w, self.y + self.h)


@dataclass
class Seg:
    """Một đoạn dây nằm trong kênh ('C', k) hoặc máng ('G', lane, gi)."""
    res: tuple
    lo: int
    hi: int
    key: object
    track: int = None
    # (vị trí, phía): đoạn nối vào track từ phía thấp (-1: trái/trên) hoặc phía cao (+1: phải/dưới)
    stubs: tuple = ()


@dataclass
class Edge:
    id: str
    src: str
    dst: str
    lines: list
    lw: float
    lh: float
    order: int
    dashed: bool = False
    highlight: bool = False
    back: bool = False
    case: str = None
    exit_side: str = None
    entry_side: str = 'T'
    exit_frac: tuple = (0.5, 1.0)
    entry_frac: tuple = (0.5, 0.0)
    sym: list = field(default_factory=list)
    pts: list = None
    label: tuple = None
    label_t: float = 0.0
    label_off: tuple = (0.0, 0.0)
    # Các trường dưới chỉ đặt khi merge.
    waypoints: list = None
    constraints: dict = None
    auto: bool = False
    label_mode: str = 'computed'


SIDE_FRAC = {'R': (1.0, 0.5), 'L': (0.0, 0.5), 'B': (0.5, 1.0), 'T': (0.5, 0.0)}
OPP = {'R': 'L', 'L': 'R'}


def rnd(v):
    return int(math.ceil(v / 2.0) * 2)


class Layout:
    def __init__(self, rows, cfg=None, tm=None):
        self.cfg = cfg or Config()
        self.tm = tm or TextMeasure()
        self.warnings = []
        self.lanes = [r for r in rows if r.type == 'lane']
        self.lane_idx = {r.id: i for i, r in enumerate(self.lanes)}
        self.items = {}
        by_id = {r.id: r for r in rows}
        for r in rows:
            if r.type in NODE_TYPES or (r.type in ATTACH_TYPES and not r.is_marker()):
                attach = r.meta.get('attach') if r.type in ATTACH_TYPES else None
                lane_id = by_id[attach].parent if attach else r.parent
                lines, w, h = self.size_item(r.type, r.lines)
                self.items[r.id] = Item(r.id, r.type, self.lane_idx[lane_id], lines, w, h, r.idx,
                                        'highlight' in r.styles(), attach)
        self.edges = []
        for r in rows:
            if r.type != 'edge':
                continue
            lines = self.tm.wrap(r.lines, self.cfg.label_wrap) if r.text else []
            lw, lh = self.tm.box(lines) if lines else (0, 0)
            st = r.styles()
            self.edges.append(Edge(r.id, r.meta['from'], r.meta['to'], lines, lw + 2 * self.cfg.label_pad, lh + 2,
                                   r.idx, 'dashed' in st, 'highlight' in st, r.meta.get('back') == 'true'))
        self.outs, self.ins = defaultdict(list), defaultdict(list)
        for e in self.edges:
            self.outs[e.src].append(e)
            self.ins[e.dst].append(e)
        self.segs = []
        self.origin = (40.0, 40.0)

    # ------------------------------------------------ sizing

    def size_item(self, kind, lines):
        tm, cfg = self.tm, self.cfg
        if kind == 'task':
            wl = tm.wrap(lines, cfg.task_max_w - 32)
            tw, th = tm.box(wl)
            return wl, rnd(max(cfg.task_min_w, tw + 32)), rnd(max(50, th + 20))
        if kind == 'condition':
            wl = tm.wrap(lines, cfg.cond_wrap)
            tw, th = tm.box(wl)
            # Chữ nằm lọt hình thoi khi tw/W + th/H <= 1.
            return wl, rnd(max(100, tw / 0.6 + 10)), rnd(max(60, th / 0.4 + 10))
        if kind in ('start', 'end', 'external'):
            wl = tm.wrap(lines, cfg.term_wrap)
            tw, th = tm.box(wl)
            pad = 12 if kind == 'end' else 0
            return wl, rnd(max(80, tw * 1.42 + 24 + pad)), rnd(max(44, th * 1.42 + 16 + pad))
        if kind == 'db':
            wl = tm.wrap(lines, cfg.db_wrap)
            tw, th = tm.box(wl)
            return wl, rnd(max(100, tw + 24)), rnd(max(60, th + 34))
        wl = tm.wrap(lines, cfg.text_wrap)
        tw, th = tm.box(wl)
        return wl, rnd(tw + 20), rnd(th + 12)

    # ------------------------------------------------ placement

    def gkey(self, it, col=None):
        return (it.lane, it.col if col is None else col)

    def branch_offset(self, e):
        u = self.items[e.src]
        same = [x for x in self.outs[e.src] if not x.back and self.items[x.dst].lane == u.lane]
        return len(same) - 1 - same.index(e)

    def topo(self):
        flow = [it for it in self.items.values() if not it.attach]
        indeg = {it.id: 0 for it in flow}
        for e in self.edges:
            if not e.back:
                indeg[e.dst] += 1
        heap = [(self.items[i].order, i) for i, d in indeg.items() if d == 0]
        heapq.heapify(heap)
        out = []
        while heap:
            _, n = heapq.heappop(heap)
            out.append(n)
            for e in self.outs[n]:
                if e.back:
                    continue
                indeg[e.dst] -= 1
                if indeg[e.dst] == 0:
                    heapq.heappush(heap, (self.items[e.dst].order, e.dst))
        rest = sorted((i for i in indeg if i not in set(out)), key=lambda i: self.items[i].order)
        if rest:
            self.warnings.append('có vòng lặp chưa đánh dấu back=true, xếp theo thứ tự bảng: ' + ', '.join(rest))
        return out + rest

    def branch_drift(self, e):
        """Nhánh này rốt cuộc đi sang lane bên nào: -1 trái, 1 phải, 0 chưa rõ.

        Đi dọc nhánh cho tới cạnh đầu tiên rời khỏi lane. Dừng ở node hợp nhánh
        vì từ đó trở đi là luồng chung, không còn là hướng của riêng nhánh này.
        """
        v = self.items[e.dst]
        seen = {v.id}
        queue = [v.id]
        while queue:
            nid = queue.pop(0)
            for x in sorted(self.outs[nid], key=lambda x: x.order):
                if x.back:
                    continue
                w = self.items[x.dst]
                if w.lane != v.lane:
                    return -1 if w.lane < v.lane else 1
                if w.id in seen:
                    continue
                if len([y for y in self.ins[w.id] if not y.back]) > 1:
                    continue
                seen.add(w.id)
                queue.append(w.id)
        return 0

    def branch_slots(self, nid):
        """Cột tương đối cho mọi cạnh ra cùng lane của một node.

        Nhánh chính (cạnh ra cuối cùng) giữ cột 0. Các nhánh phụ, gần nhánh chính
        trước, nhận cột lệch sang phía mà nhánh đó dẫn tới, để mũi tên rời nhánh
        không phải vòng ngược qua node khác. Nhánh không rõ hướng thì xen kẽ phải
        rồi trái. Mặt nào đã có mũi tên ngang thì mọi nhánh phụ dồn sang mặt kia.
        """
        u = self.items[nid]
        same = [x for x in self.outs[nid] if not x.back and self.items[x.dst].lane == u.lane]
        res = {}
        if not same:
            return res
        res[same[-1].id] = 0
        busy = self.hside[nid]
        only_left = 'R' in busy and 'L' not in busy
        only_right = 'L' in busy and 'R' not in busy
        nxt = {1: 1, -1: 1}
        for e in reversed(same[:-1]):
            if only_left:
                side = -1
            elif only_right:
                side = 1
            else:
                side = self.branch_drift(e) or (1 if nxt[1] <= nxt[-1] else -1)
            res[e.id] = side * nxt[side]
            nxt[side] += 1
        return res

    def branch_slot(self, e):
        return self.branch_slots(e.src).get(e.id, 0)

    def merge_col(self, same, v):
        """Cột cho node có nhiều nhánh cùng lane đi vào.

        Các nhánh đó rẽ ra từ một node chung; luồng chung nên quay về đúng cột
        của node rẽ gần nhất, thay vì bám theo cột của nhánh được xếp sau cùng.
        Không tìm được node chung thì trả None để dùng cách cũ.
        """
        sets = []
        for e in same:
            seen, stack = set(), [e.src]
            while stack:
                n = stack.pop()
                if n in seen:
                    continue
                seen.add(n)
                for x in self.ins[n]:
                    w = self.items[x.src]
                    if not x.back and w.lane == v.lane and w.row is not None:
                        stack.append(x.src)
            sets.append(seen)
        common = set.intersection(*sets)
        if not common:
            return None
        return self.items[max(common, key=lambda i: (self.items[i].row, self.items[i].order))].col

    def side_branches(self, v):
        return sum(1 for e in self.outs[v.id]
                   if not e.back and self.items[e.dst].lane == v.lane and self.branch_offset(e) > 0)

    def attach_side(self, v):
        right = left = False
        for e in self.outs[v.id] + self.ins[v.id]:
            if e.back:
                continue
            o = self.items[e.dst if e.src == v.id else e.src]
            if o.lane > v.lane:
                right = True
            elif o.lane < v.lane:
                left = True
            elif e.src == v.id and self.branch_slot(e) > 0:
                right = True
            elif e.src == v.id and self.branch_slot(e) < 0:
                left = True
        if not right:
            return 1
        return -1 if not left else 1

    def place(self):
        occ = {}
        hspans = []
        self.hside = defaultdict(set)
        attachments = defaultdict(list)
        for it in sorted(self.items.values(), key=lambda i: i.order):
            if it.attach:
                attachments[it.attach].append(it)

        def in_span(key, r):
            return any(hr == r and a < key < b for hr, a, b in hspans)

        def blocked(lane, c, r):
            return (lane, c, r) in occ or in_span((lane, c), r)

        max_row = -1
        self.topo_order = self.topo()
        for vid in self.topo_order:
            v = self.items[vid]
            preds = [e for e in self.ins[vid] if not e.back and self.items[e.src].row is not None]
            same = [e for e in preds if self.items[e.src].lane == v.lane]
            side_branch = 0
            col = 0
            if same:
                e0 = max(same, key=lambda e: (self.items[e.src].row, e.order))
                slot = self.branch_slot(e0)
                col = self.items[e0.src].col + slot
                side_branch = (slot > 0) - (slot < 0)
                if len(same) > 1:
                    mc = self.merge_col(same, v)
                    if mc is not None:
                        col, side_branch = mc, 0
            if preds:
                row = max(self.items[e.src].row + 1 for e in preds)
            else:
                row = 0 if v.kind == 'start' else max_row + 1
            placed = False
            nonback_in = [e for e in self.ins[vid] if not e.back]
            needs_sides = v.kind in NON_RECT and self.side_branches(v) >= 2
            if (len(preds) == 1 and len(nonback_in) == 1 and not needs_sides
                    and (not same or side_branch)):
                u = self.items[preds[0].src]
                r = u.row
                a, b = sorted([self.gkey(u), (v.lane, col)])
                free = not blocked(v.lane, col, r)
                free = free and not any(k[2] == r and a < (k[0], k[1]) < b for k in occ)
                free = free and not any(hr == r and a2 < b and a < b2 for hr, a2, b2 in hspans)
                if free:
                    row = r
                    hspans.append((r, a, b))
                    placed = True
                    to_right = (v.lane, col) > self.gkey(u)
                    self.hside[u.id].add('R' if to_right else 'L')
                    self.hside[v.id].add('L' if to_right else 'R')
            tries = 0
            while not placed and blocked(v.lane, col, row):
                if side_branch and tries < 3:
                    col += side_branch
                else:
                    row += 1
                tries += 1
            v.row, v.col = row, col
            occ[(v.lane, col, row)] = vid
            max_row = max(max_row, row)
            for a in attachments[vid]:
                side = self.attach_side(v)
                cands = [side, -side, 2 * side, -2 * side]
                c = next((col + d for d in cands if not blocked(v.lane, col + d, row)), None)
                if c is None:
                    c = next(col + d for d in range(3, 50) if (v.lane, col + d, row) not in occ)
                    self.warnings.append(f'{a.id}: không còn ô trống cạnh {vid}, đặt xa hơn')
                a.row, a.col = row, c
                occ[(v.lane, c, row)] = a.id
        self.occ = occ
        self.attachments = attachments
        self.nrows = max_row + 1
        self.cols = defaultdict(list)
        for (lane, c, _r) in occ:
            if c not in self.cols[lane]:
                self.cols[lane].append(c)
        for lane in range(len(self.lanes)):
            self.cols[lane].sort()
        self.xord = {}
        n = 0
        for lane in range(len(self.lanes)):
            cols = self.cols[lane]
            for gi in range(len(cols) + 1):
                self.xord[('G', lane, gi)] = n
                n += 1
                if gi < len(cols):
                    self.xord[('c', lane, cols[gi])] = n
                    n += 1

    # ------------------------------------------------ routing

    def cells_between(self, a, b, r):
        """Các ô (lane, col, row) nằm ngặt giữa hai cột toàn cục a < b trên hàng r."""
        a, b = min(a, b), max(a, b)
        out = []
        for lane in range(a[0], b[0] + 1):
            for c in self.cols[lane]:
                if a < (lane, c) < b:
                    out.append((lane, c, r))
        return out

    def gutter(self, lane, col, side):
        i = self.cols[lane].index(col)
        return i if side == 'L' else i + 1

    def seg(self, res, lo, hi, key, stubs=()):
        s = Seg(res, min(lo, hi), max(lo, hi), key, stubs=tuple(stubs))
        self.segs.append(s)
        return s

    def route(self):
        it = self.items
        cells_h = defaultdict(set)
        cells_v = defaultdict(set)
        self.side_out = defaultdict(list)
        self.side_in = defaultdict(list)
        occ = self.occ

        def face(u, v):
            if (v.lane, v.col) == (u.lane, u.col):
                return 'R'
            return 'R' if (v.lane, v.col) > (u.lane, u.col) else 'L'

        # A: thẳng đứng trong cùng cột
        for e in self.edges:
            u, v = it[e.src], it[e.dst]
            if e.back or (u.lane, u.col) != (v.lane, v.col) or v.row <= u.row:
                continue
            cells = [(u.lane, u.col, r) for r in range(u.row + 1, v.row)]
            if self.side_out[(u.id, 'B')]:
                continue
            if all(c not in occ and cells_v[c] <= {v.id} for c in cells):
                e.case, e.exit_side = 'A', 'B'
                for c in cells:
                    cells_v[c].add(v.id)
                self.side_out[(u.id, 'B')].append(e)
                self.side_in[(v.id, 'T')].append(e)

        # B: thẳng ngang cùng hàng
        for e in self.edges:
            u, v = it[e.src], it[e.dst]
            if e.case or e.back or v.row != u.row or (u.lane, u.col) == (v.lane, v.col):
                continue
            s = face(u, v)
            if self.side_out[(u.id, s)] or self.side_in[(u.id, s)]:
                continue
            if self.side_out[(v.id, OPP[s])] or self.side_in[(v.id, OPP[s])]:
                continue
            if s in self.attach_sides(u) or OPP[s] in self.attach_sides(v):
                continue
            cells = self.cells_between(self.gkey(u), self.gkey(v), u.row)
            if all(c not in occ and not cells_h[c] for c in cells):
                e.case, e.exit_side, e.entry_side = 'B', s, OPP[s]
                for c in cells:
                    cells_h[c].add(v.id)
                self.side_out[(u.id, s)].append(e)
                self.side_in[(v.id, OPP[s])].append(e)

        # C: chữ L, ra mặt bên rồi rẽ xuống đỉnh đích
        for e in self.edges:
            u, v = it[e.src], it[e.dst]
            if e.case or e.back or v.row <= u.row or (u.lane, u.col) == (v.lane, v.col):
                continue
            s = face(u, v)
            if self.side_out[(u.id, s)] or self.side_in[(u.id, s)]:
                continue
            if s in self.attach_sides(u):
                continue
            turn = (v.lane, v.col, u.row)
            hcells = self.cells_between(self.gkey(u), self.gkey(v), u.row) + [turn]
            vcells = [turn] + [(v.lane, v.col, r) for r in range(u.row + 1, v.row)]
            ok = all(c not in occ and not cells_h[c] for c in hcells)
            ok = ok and all(c not in occ and cells_v[c] <= {v.id} for c in vcells)
            if ok:
                e.case, e.exit_side = 'C', s
                for c in hcells:
                    cells_h[c].add(v.id)
                for c in vcells:
                    cells_v[c].add(v.id)
                self.side_out[(u.id, s)].append(e)
                self.side_in[(v.id, 'T')].append(e)
                e.sym = [(('col', v.lane, v.col), ('src',))]

        # D: tổng quát qua kênh và máng
        for e in self.edges:
            if e.case:
                continue
            u, v = it[e.src], it[e.dst]
            s = face(u, v)
            e.case = 'D'
            choices = [s, OPP[s]] if e.back or v.row <= u.row else [s, 'B', OPP[s]]
            e.exit_side = next((c for c in choices if self.side_free(u, c)), s)
            self.side_out[(u.id, e.exit_side)].append(e)
            self.side_in[(v.id, 'T')].append(e)
            vcol = ('col', v.lane, v.col)
            ch_v = ('C', v.row)
            if e.exit_side == 'B':
                ch1 = ('C', u.row + 1)
                if v.row == u.row + 1:
                    s1 = self.seg(ch1, self.xord[('c', u.lane, u.col)], self.xord[('c', v.lane, v.col)], ('to', v.id),
                                  [(self.xord[('c', u.lane, u.col)], -1), (self.xord[('c', v.lane, v.col)], 1)])
                    e.sym = [(('src',), ('seg', s1)), (vcol, ('seg', s1))]
                    continue
                vcells = [(v.lane, v.col, r) for r in range(u.row + 1, v.row)]
                if all(c not in occ and cells_v[c] <= {v.id} for c in vcells) and (u.lane, u.col) != (v.lane, v.col):
                    for c in vcells:
                        cells_v[c].add(v.id)
                    s1 = self.seg(ch1, self.xord[('c', u.lane, u.col)], self.xord[('c', v.lane, v.col)], None,
                                  [(self.xord[('c', u.lane, u.col)], -1), (self.xord[('c', v.lane, v.col)], 1)])
                    e.sym = [(('src',), ('seg', s1)), (vcol, ('seg', s1))]
                    continue
                gside = 'R' if (u.lane, u.col) >= (v.lane, v.col) else 'L'
                gt = ('G', v.lane, self.gutter(v.lane, v.col, gside))
                s1 = self.seg(ch1, self.xord[('c', u.lane, u.col)], self.xord[gt], None,
                              [(self.xord[('c', u.lane, u.col)], -1)])
                s2 = self.seg(gt, 2 * (u.row + 1), 2 * v.row, ('to', v.id))
                s3 = self.seg(ch_v, self.xord[gt], self.xord[('c', v.lane, v.col)], ('to', v.id),
                              [(self.xord[('c', v.lane, v.col)], 1)])
                e.sym = [(('src',), ('seg', s1)), (('seg', s2), ('seg', s1)),
                         (('seg', s2), ('seg', s3)), (vcol, ('seg', s3))]
            else:
                gs = ('G', u.lane, self.gutter(u.lane, u.col, e.exit_side))
                s1 = self.seg(gs, 2 * u.row + 1, 2 * v.row, ('to', v.id),
                              [(2 * u.row + 1, -1 if e.exit_side == 'R' else 1)])
                s2 = self.seg(ch_v, self.xord[gs], self.xord[('c', v.lane, v.col)], ('to', v.id),
                              [(self.xord[('c', v.lane, v.col)], 1)])
                e.sym = [(('seg', s1), ('src',)), (('seg', s1), ('seg', s2)), (vcol, ('seg', s2))]

        self.assign_ports()
        self.assign_tracks()

    def attach_sides(self, u):
        """Các mặt của u đang có db hoặc text đứng sát."""
        return {('R' if a.col > u.col else 'L') for a in self.attachments.get(u.id, ())}

    def side_free(self, u, side):
        if side in self.attach_sides(u):
            return False
        if self.side_in[(u.id, side)]:
            return False
        used = self.side_out[(u.id, side)]
        if u.kind in NON_RECT:
            return not used
        return not any(x.case in ('B', 'C') for x in used)

    def assign_ports(self):
        for (nid, side), es in self.side_out.items():
            u = self.items[nid]
            if not es:
                continue
            if u.kind in NON_RECT or len(es) == 1:
                for e in es:
                    e.exit_frac = SIDE_FRAC[side]
                continue
            fixed = [e for e in es if e.case in ('A', 'B', 'C')]
            loose = [e for e in es if e not in fixed]
            for e in fixed:
                e.exit_frac = SIDE_FRAC[side]
            n = len(loose)
            if fixed:
                fr = [0.25, 0.75, 0.125, 0.875, 0.375, 0.625][:n] if n <= 6 else [(i + 1) / (n + 1) for i in range(n)]
                fr.sort()
            else:
                fr = [(i + 1) / (n + 1) for i in range(n)]
            # Dây rẽ về phía nào thì lấy cổng ở phía đó để các đoạn đầu không cắt nhau.
            loose.sort(key=lambda e: (self.items[e.dst].row, e.order) if side in ('L', 'R')
                       else (self.items[e.dst].lane, self.items[e.dst].col, e.order))
            for e, f in zip(loose, fr):
                e.exit_frac = (f, 1.0) if side == 'B' else (1.0 if side == 'R' else 0.0, f)
        for e in self.edges:
            e.entry_frac = SIDE_FRAC[e.entry_side]

    def assign_tracks(self):
        """Tô màu khoảng cho từng kênh/máng.

        Ràng buộc thứ tự: hai đoạn có chân nối vào cùng một vị trí từ hai phía
        (vd hai node cùng hàng ở hai bên một máng) thì đoạn phía thấp phải nằm
        track nhỏ hơn, nếu không hai chân ngang/dọc sẽ đè lên nhau.
        """
        by_res = defaultdict(list)
        for s in self.segs:
            by_res[s.res].append(s)
        self.ntracks = {}

        def overlap(a, b):
            return a.lo <= b.hi and b.lo <= a.hi

        def must_precede(a, b):
            if a.key is not None and a.key == b.key:
                return False
            return any(pa == pb and da < db for pa, da in a.stubs for pb, db in b.stubs)

        def order_ok(tracks, s, ti):
            for tj, tl in enumerate(tracks):
                for t in tl:
                    if must_precede(s, t) and not ti < tj:
                        return False
                    if must_precede(t, s) and not tj < ti:
                        return False
            return True

        for res, segs in by_res.items():
            tracks = []
            for s in segs:
                def fits(ti, bus_only):
                    tl = tracks[ti]
                    if bus_only and not any(t.key == s.key for t in tl):
                        return False
                    if not all((s.key is not None and t.key == s.key) or not overlap(t, s) for t in tl):
                        return False
                    return order_ok(tracks, s, ti)

                chosen = None
                if s.key is not None:
                    chosen = next((ti for ti in range(len(tracks)) if fits(ti, True)), None)
                if chosen is None:
                    chosen = next((ti for ti in range(len(tracks)) if fits(ti, False)), None)
                if chosen is None:
                    lo = max((tj + 1 for tj, tl in enumerate(tracks) for t in tl if must_precede(t, s)), default=0)
                    hi = min((tj for tj, tl in enumerate(tracks) for t in tl if must_precede(s, t)), default=len(tracks))
                    if lo > hi:
                        self.warnings.append(f'không xếp được thứ tự track trong {res}')
                    chosen = max(lo, min(hi, len(tracks))) if lo <= hi else len(tracks)
                    tracks.insert(chosen, [])
                tracks[chosen].append(s)
            for ti, tl in enumerate(tracks):
                for t in tl:
                    t.track = ti
            self.ntracks[res] = len(tracks)

    # ------------------------------------------------ geometry

    def label_reservations(self, spans=None):
        """Chừa chỗ cho nhãn: máng bên phía cổng ra (ra ngang) hoặc kênh dưới nguồn (ra đáy)."""
        lz_g = defaultdict(float)
        lz_c = defaultdict(float)
        for e in self.edges:
            if not e.lines:
                continue
            u = self.items[e.src]
            if e.exit_side == 'B':
                lz_c[u.row + 1] = max(lz_c[u.row + 1], e.lh + 8)
                continue
            gi = self.gutter(u.lane, u.col, e.exit_side)
            key = (u.lane, gi, 'l' if e.exit_side == 'R' else 'r')
            crosses = self.items[e.dst].lane != u.lane
            if e.case == 'D' or (crosses and spans is not None):
                lz_g[key] = max(lz_g[key], e.lw + 8)
            elif spans is not None:
                deficit = e.lw + 16 - spans.get(e.id, 0)
                if deficit > 0:
                    lz_g[key] = max(lz_g[key], deficit)
        return lz_g, lz_c

    def hug_plan(self):
        """db/text nào được đặt bám sát node neo, và ở mặt nào.

        Chỉ bám khi mặt đó không có dây nào ra hoặc vào: đoạn dây ngang sát node
        sẽ cắt qua chỗ định đặt. Mặt bận thì giữ cách cũ, tức căn giữa ô bên cạnh.
        """
        hugs = {}
        for aid, atts in self.attachments.items():
            u = self.items[aid]
            for side in ('L', 'R'):
                near = [a for a in atts if (a.col > u.col) == (side == 'R')
                        and abs(a.col - u.col) == 1]
                if len(near) != 1:
                    continue
                if self.side_out[(u.id, side)] or self.side_in[(u.id, side)]:
                    continue
                hugs[near[0].id] = (u, side)
        return hugs

    def compute_geometry(self, lz_g, lz_c):
        cfg = self.cfg
        self.hugs = self.hug_plan()
        col_w = defaultdict(float)
        row_h = defaultdict(float)
        for it in self.items.values():
            if it.id not in self.hugs:
                col_w[(it.lane, it.col)] = max(col_w[(it.lane, it.col)], it.w)
            row_h[it.row] = max(row_h[it.row], it.h)
        az_g = defaultdict(float)
        for aid, (u, side) in self.hugs.items():
            gi = self.gutter(u.lane, u.col, side)
            key = (u.lane, gi, 'l' if side == 'R' else 'r')
            az_g[key] = max(az_g[key], cfg.attach_gap + self.items[aid].w)
        self.az_g = az_g
        self.gut_x, self.gut_w, self.col_x = {}, {}, {}
        self.lane_x, self.lane_w = [], []
        x = 0.0
        for lane, lrow in enumerate(self.lanes):
            cols = self.cols[lane]
            widths = []
            for gi in range(len(cols) + 1):
                n = self.ntracks.get(('G', lane, gi), 0)
                inner = 2 * cfg.gutter_margin + (n - 1) * cfg.track_gap if n else 0
                lzl, lzr = lz_g.get((lane, gi, 'l'), 0), lz_g.get((lane, gi, 'r'), 0)
                azl, azr = az_g.get((lane, gi, 'l'), 0), az_g.get((lane, gi, 'r'), 0)
                widths.append(azl + azr + lzl + lzr + max(cfg.min_gutter, inner))
            total = sum(widths) + sum(col_w[(lane, c)] for c in cols)
            head = self.tm.w(lrow.text) + 30
            need = max(cfg.min_lane_w, head) - total
            if need > 0:
                widths[0] += need / 2
                widths[-1] += need / 2
                total += need
            self.lane_x.append(x)
            self.lane_w.append(total)
            cx = x
            for gi in range(len(cols) + 1):
                self.gut_x[(lane, gi)] = cx
                self.gut_w[(lane, gi)] = widths[gi]
                cx += widths[gi]
                if gi < len(cols):
                    self.col_x[(lane, cols[gi])] = (cx, col_w[(lane, cols[gi])])
                    cx += col_w[(lane, cols[gi])]
            x += total
        self.pool_w = x
        top = cfg.pool_header + cfg.lane_header
        self.chan_y, self.chan_h, self.row_y = {}, {}, {}
        y = top
        for k in range(self.nrows + 1):
            n = self.ntracks.get(('C', k), 0)
            inner = 2 * cfg.channel_margin + (n - 1) * cfg.track_gap if n else 0
            h = max(cfg.min_channel, lz_c.get(k, 0) + inner)
            self.chan_y[k], self.chan_h[k] = y, h
            y += h
            if k < self.nrows:
                self.row_y[k] = (y, row_h[k])
                y += row_h[k]
        self.pool_h = y
        self.lz_g, self.lz_c = lz_g, lz_c
        for it in self.items.values():
            cx0, cw = self.col_x[(it.lane, it.col)]
            ry0, rh = self.row_y[it.row]
            it.x = cx0 + (cw - it.w) / 2
            it.y = ry0 + (rh - it.h) / 2
        for aid, (u, side) in self.hugs.items():
            a = self.items[aid]
            a.x = u.x + u.w + cfg.attach_gap if side == 'R' else u.x - cfg.attach_gap - a.w

    def track_x(self, s):
        _, lane, gi = s.res
        return (self.gut_x[(lane, gi)] + self.az_g.get((lane, gi, 'l'), 0)
                + self.lz_g.get((lane, gi, 'l'), 0)
                + self.cfg.gutter_margin + s.track * self.cfg.track_gap)

    def track_y(self, s):
        k = s.res[1]
        return self.chan_y[k] + self.lz_c.get(k, 0) + self.cfg.channel_margin + s.track * self.cfg.track_gap

    def resolve_paths(self):
        for e in self.edges:
            u, v = self.items[e.src], self.items[e.dst]
            p0 = (u.x + e.exit_frac[0] * u.w, u.y + e.exit_frac[1] * u.h)
            p1 = (v.x + e.entry_frac[0] * v.w, v.y + e.entry_frac[1] * v.h)

            def rx(s):
                if s[0] == 'col':
                    cx0, cw = self.col_x[(s[1], s[2])]
                    return v.x + v.w / 2 if (s[1], s[2]) == (v.lane, v.col) else cx0 + cw / 2
                if s[0] == 'seg':
                    return self.track_x(s[1])
                return p0[0]

            def ry(s):
                if s[0] == 'seg':
                    return self.track_y(s[1])
                return p0[1]

            pts = [p0] + [(rx(a), ry(b)) for a, b in e.sym] + [p1]
            clean = [pts[0]]
            for p in pts[1:]:
                if abs(p[0] - clean[-1][0]) < 0.01 and abs(p[1] - clean[-1][1]) < 0.01:
                    continue
                clean.append(p)
            # Bỏ điểm nằm giữa hai đoạn thẳng hàng.
            out = [clean[0]]
            for i in range(1, len(clean) - 1):
                a, b, c = out[-1], clean[i], clean[i + 1]
                if (abs(a[0] - b[0]) < 0.01 and abs(b[0] - c[0]) < 0.01) or \
                   (abs(a[1] - b[1]) < 0.01 and abs(b[1] - c[1]) < 0.01):
                    continue
                out.append(b)
            out.append(clean[-1])
            e.pts = [(round(x, 2), round(y, 2)) for x, y in out]

    def horizontal_spans(self):
        """Bề dài phần ngang đầu tiên của cạnh B/C (chỗ đặt nhãn) sau lần tính đầu."""
        spans = {}
        for e in self.edges:
            if e.case in ('B', 'C') and e.lines and e.pts:
                (x1, _), (x2, _) = e.pts[0], e.pts[1]
                spans[e.id] = abs(x2 - x1)
        return spans

    # ------------------------------------------------ labels

    def place_labels(self):
        boxes = [(it.id, it.box()) for it in self.items.values()]
        boxes.append(('header', (-1e6, -1e6, 1e6, self.cfg.pool_header + self.cfg.lane_header)))
        segs = [(None, (x, 0), (x, self.pool_h)) for x in self.lane_x[1:]]
        for e in self.edges:
            for a, b in zip(e.pts, e.pts[1:]):
                segs.append((e.id, a, b))
        placed = []
        for e in sorted(self.edges, key=lambda x: x.order):
            if not e.lines:
                continue
            best = None
            for cand in self.label_candidates(e):
                box = cand[0]
                cost = 0.0
                for _, bb in boxes:
                    cost += area(box, bb) * 4
                for _, lb in placed:
                    cost += area(box, lb) * 4
                for eid, a, b in segs:
                    if eid != e.id and seg_hits(a, b, box):
                        cost += 100
                if best is None or cost < best[0]:
                    best = (cost, cand)
                if cost == 0:
                    break
            if best is None:
                self.warnings.append(f'{e.id}: đường quá ngắn, không có chỗ đặt nhãn')
                continue
            cost, (box, dist, anchor) = best
            if cost > 0:
                self.warnings.append(f'{e.id}: nhãn "{" ".join(e.lines)}" không tìm được chỗ trống hoàn toàn')
            placed.append((e.id, box))
            e.label = box
            total = path_len(e.pts)
            e.label_t = round(2 * dist / total - 1, 4) if total else 0
            cx, cy = (box[0] + box[2]) / 2, (box[1] + box[3]) / 2
            e.label_off = (round(cx - anchor[0], 2), round(cy - anchor[1], 2))

    def label_candidates(self, e):
        w, h = e.lw, e.lh
        acc = 0.0
        out = []
        for a, b in zip(e.pts, e.pts[1:]):
            L = abs(b[0] - a[0]) + abs(b[1] - a[1])
            if L >= 12:
                if abs(a[1] - b[1]) < 0.01:
                    d = 1 if b[0] > a[0] else -1
                    for cx in (a[0] + d * (6 + w / 2), (a[0] + b[0]) / 2):
                        ax = min(max(cx, min(a[0], b[0])), max(a[0], b[0]))
                        for cy in (a[1] - 3 - h / 2, a[1] + 3 + h / 2):
                            box = (cx - w / 2, cy - h / 2, cx + w / 2, cy + h / 2)
                            out.append((box, acc + abs(ax - a[0]), (ax, a[1])))
                else:
                    d = 1 if b[1] > a[1] else -1
                    for cy in (a[1] + d * (6 + h / 2), (a[1] + b[1]) / 2):
                        ay = min(max(cy, min(a[1], b[1])), max(a[1], b[1]))
                        for cx in (a[0] + 5 + w / 2, a[0] - 5 - w / 2):
                            box = (cx - w / 2, cy - h / 2, cx + w / 2, cy + h / 2)
                            out.append((box, acc + abs(ay - a[1]), (a[0], ay)))
            acc += L
        return out

    # ------------------------------------------------ checks

    def check(self):
        issues = []
        boxes = {i: it.box() for i, it in self.items.items()}
        ids = sorted(boxes)
        for i, a in enumerate(ids):
            for b in ids[i + 1:]:
                if area(boxes[a], boxes[b]) > 0:
                    issues.append(('error', f'{a} và {b} chồng lên nhau'))
        for it in self.items.values():
            lx, lw = self.lane_x[it.lane], self.lane_w[it.lane]
            if it.x < lx or it.x + it.w > lx + lw:
                issues.append(('error', f'{it.id} tràn ra ngoài lane'))
        allsegs = []
        for e in self.edges:
            pts = e.pts
            for k, (a, b) in enumerate(zip(pts, pts[1:])):
                if abs(a[0] - b[0]) > 0.01 and abs(a[1] - b[1]) > 0.01:
                    issues.append(('error', f'{e.id}: đoạn {k} không vuông góc'))
                for nid, bx in boxes.items():
                    if nid in (e.src, e.dst) and (k == 0 or k == len(pts) - 2):
                        continue
                    if seg_hits(a, b, shrink(bx, 1)):
                        issues.append(('error', f'{e.id}: dây cắt qua {nid}'))
                allsegs.append((e, a, b))
        for i, (e1, a1, b1) in enumerate(allsegs):
            for e2, a2, b2 in allsegs[i + 1:]:
                if e1 is e2 or e1.dst == e2.dst or e1.src == e2.src:
                    continue
                if collinear_overlap(a1, b1, a2, b2) > 1:
                    issues.append(('error', f'{e1.id} và {e2.id} chồng dây'))
        labels = [(e, e.label) for e in self.edges if e.label]
        for i, (e, lb) in enumerate(labels):
            for nid, bx in boxes.items():
                if area(lb, bx) > 0:
                    issues.append(('warning', f'nhãn {e.id} đè lên {nid}'))
            for e2, lb2 in labels[i + 1:]:
                if area(lb, lb2) > 0:
                    issues.append(('warning', f'nhãn {e.id} đè nhãn {e2.id}'))
            for e2, a, b in allsegs:
                if e2 is not e and seg_hits(a, b, lb):
                    issues.append(('warning', f'nhãn {e.id} đè dây {e2.id}'))
        seen = set()
        out = []
        for x in issues:
            if x not in seen:
                seen.add(x)
                out.append(x)
        return out

    # ------------------------------------------------ run

    def run(self):
        self.topo_order = []
        self.place()
        self.route()
        lz_g, lz_c = self.label_reservations()
        self.compute_geometry(lz_g, lz_c)
        self.resolve_paths()
        lz_g, lz_c = self.label_reservations(self.horizontal_spans())
        self.compute_geometry(lz_g, lz_c)
        self.resolve_paths()
        self.place_labels()
        return self

    def to_json(self):
        return {
            'pool': {'w': self.pool_w, 'h': self.pool_h},
            'lanes': [{'id': l.id, 'x': self.lane_x[i], 'w': self.lane_w[i]} for i, l in enumerate(self.lanes)],
            'items': {i: {'kind': it.kind, 'row': it.row, 'col': it.col, 'x': it.x, 'y': it.y, 'w': it.w, 'h': it.h}
                      for i, it in self.items.items()},
            'edges': {e.id: {'case': e.case, 'exit': e.exit_side, 'points': e.pts, 'label': e.label}
                      for e in self.edges},
        }


# ---------------------------------------------------------------- geometry helpers

def area(a, b):
    w = min(a[2], b[2]) - max(a[0], b[0])
    h = min(a[3], b[3]) - max(a[1], b[1])
    return w * h if w > 0 and h > 0 else 0.0


def shrink(b, d):
    return (b[0] + d, b[1] + d, b[2] - d, b[3] - d)


def seg_hits(a, b, box):
    x1, x2 = sorted((a[0], b[0]))
    y1, y2 = sorted((a[1], b[1]))
    return x1 < box[2] and x2 > box[0] and y1 < box[3] and y2 > box[1]


def collinear_overlap(a1, b1, a2, b2):
    if abs(a1[1] - b1[1]) < 0.01 and abs(a2[1] - b2[1]) < 0.01 and abs(a1[1] - a2[1]) < 0.5:
        lo, hi = max(min(a1[0], b1[0]), min(a2[0], b2[0])), min(max(a1[0], b1[0]), max(a2[0], b2[0]))
        return hi - lo
    if abs(a1[0] - b1[0]) < 0.01 and abs(a2[0] - b2[0]) < 0.01 and abs(a1[0] - a2[0]) < 0.5:
        lo, hi = max(min(a1[1], b1[1]), min(a2[1], b2[1])), min(max(a1[1], b1[1]), max(a2[1], b2[1]))
        return hi - lo
    return 0.0


def path_len(pts):
    return sum(abs(b[0] - a[0]) + abs(b[1] - a[1]) for a, b in zip(pts, pts[1:]))


# ---------------------------------------------------------------- xml

def html_lines(lines):
    return '<br>'.join(l.replace('&', '&amp;').replace('<', '&lt;').replace('>', '&gt;') for l in lines)


def fmt(v):
    return ('%.2f' % v).rstrip('0').rstrip('.')


def to_drawio(lay, title, extras=(), other_pages=()):
    cfg = lay.cfg
    ox, oy = lay.origin
    mxfile = ET.Element('mxfile', host='flowtable2drawio')
    diagram = ET.SubElement(mxfile, 'diagram', id='flowtable', name=(title or 'Flow')[:80])
    model = ET.SubElement(diagram, 'mxGraphModel', grid='1', gridSize='10', guides='1', tooltips='1',
                          connect='1', arrows='1', fold='1', page='1', pageScale='1',
                          pageWidth=fmt(lay.pool_w + 2 * ox), pageHeight=fmt(lay.pool_h + 2 * oy),
                          math='0', shadow='0')
    root = ET.SubElement(model, 'root')
    ET.SubElement(root, 'mxCell', id='0')
    ET.SubElement(root, 'mxCell', id='1', parent='0')

    def geo(cell, x, y, w, h):
        ET.SubElement(cell, 'mxGeometry', x=fmt(x), y=fmt(y), width=fmt(w), height=fmt(h), **{'as': 'geometry'})

    pool = ET.SubElement(root, 'mxCell', id='pool', value=html_lines([title or 'Flow']), vertex='1', parent='1',
                         style='swimlane;html=1;childLayout=stackLayout;horizontalStack=1;resizeParent=1;'
                               f'resizeParentMax=0;startSize={cfg.pool_header};collapsible=0;'
                               'fontStyle=1;' + FONT + MARK)
    geo(pool, ox, oy, lay.pool_w, lay.pool_h)
    for i, lane in enumerate(lay.lanes):
        c = ET.SubElement(root, 'mxCell', id=lane.id, value=html_lines(lane.lines), vertex='1', parent='pool',
                          style=f'swimlane;html=1;startSize={cfg.lane_header};collapsible=0;' + FONT + MARK)
        geo(c, lay.lane_x[i], cfg.pool_header, lay.lane_w[i], lay.pool_h - cfg.pool_header)
    for it in sorted(lay.items.values(), key=lambda i: i.order):
        style = SHAPE_STYLE[it.kind]
        if it.highlight:
            style += HIGHLIGHT_TEXT if it.kind == 'text' else HIGHLIGHT_NODE
        lane_id = lay.lanes[it.lane].id
        c = ET.SubElement(root, 'mxCell', id=it.id, value=html_lines(it.lines), vertex='1', parent=lane_id,
                          style=style + FONT + MARK)
        geo(c, it.x - lay.lane_x[it.lane], it.y - cfg.pool_header, it.w, it.h)
    for e in sorted(lay.edges, key=lambda x: x.order):
        style = ('edgeStyle=orthogonalEdgeStyle;rounded=0;orthogonalLoop=1;jettySize=auto;html=1;'
                 'labelBackgroundColor=default;')
        if e.auto:
            pass
        elif e.constraints is not None:
            style += ''.join(f'{k}={v};' for k, v in e.constraints.items())
        else:
            style += (f'exitX={fmt(e.exit_frac[0])};exitY={fmt(e.exit_frac[1])};exitDx=0;exitDy=0;'
                      f'entryX={fmt(e.entry_frac[0])};entryY={fmt(e.entry_frac[1])};entryDx=0;entryDy=0;')
        if e.dashed:
            style += 'dashed=1;'
        if e.highlight:
            style += HIGHLIGHT_EDGE
        c = ET.SubElement(root, 'mxCell', id=e.id, value=html_lines(e.lines), edge='1', parent='pool',
                          source=e.src, target=e.dst, style=style + FONT + MARK)
        g = ET.SubElement(c, 'mxGeometry', relative='1', **{'as': 'geometry'})
        with_label = bool(e.label) and e.label_mode == 'computed' and not e.auto
        if with_label:
            g.set('x', fmt(e.label_t))
        if e.auto:
            pts = []
        elif e.waypoints is not None:
            pts = e.waypoints
        else:
            pts = e.pts[1:-1]
        if pts:
            arr = ET.SubElement(g, 'Array', **{'as': 'points'})
            for x, y in pts:
                ET.SubElement(arr, 'mxPoint', x=fmt(x), y=fmt(y))
        if with_label:
            ET.SubElement(g, 'mxPoint', x=fmt(e.label_off[0]), y=fmt(e.label_off[1]), **{'as': 'offset'})
    for el in extras:
        root.append(el)
    for page in other_pages:
        mxfile.append(page)
    ET.indent(mxfile, space='  ')
    return ET.tostring(mxfile, encoding='unicode')


# ---------------------------------------------------------------- merge

CONSTRAINT_KEYS = ('exitX', 'exitY', 'exitDx', 'exitDy', 'exitPerimeter',
                   'entryX', 'entryY', 'entryDx', 'entryDy', 'entryPerimeter')
TABLE_ID_RE = re.compile(r'^(?:[A-Z][A-Z0-9_]*-\d+(?:\.\d+)*|E\d+(?:\.\d+)*)$')
LAYER_IDS = ('0', '1')


def parse_style(raw):
    out = {}
    for part in (raw or '').split(';'):
        if not part:
            continue
        k, _, v = part.partition('=')
        out[k] = v if _ else None
    return out


def num(el, key, default=0.0):
    try:
        return float(el.get(key, default)) if el is not None else default
    except ValueError:
        return default


@dataclass
class OldCell:
    id: str
    elem: ET.Element
    cell: ET.Element
    order: int

    @property
    def parent(self):
        return self.cell.get('parent', '')

    @property
    def vertex(self):
        return self.cell.get('vertex') == '1'

    @property
    def edge(self):
        return self.cell.get('edge') == '1'

    @property
    def style_raw(self):
        return self.cell.get('style', '') or ''

    @property
    def style(self):
        return parse_style(self.style_raw)

    @property
    def geo(self):
        return self.cell.find('mxGeometry')

    def points(self):
        g = self.geo
        arr = g.find("Array[@as='points']") if g is not None else None
        return [(num(p, 'x'), num(p, 'y')) for p in arr.findall('mxPoint')] if arr is not None else []


@dataclass
class OldDiagram:
    cells: dict
    other_pages: list

    def origin(self, cid):
        """Gốc toạ độ tuyệt đối của hệ con của cell cid (vertex lồng nhau cộng dồn)."""
        x = y = 0.0
        seen = set()
        while cid in self.cells and cid not in LAYER_IDS and cid not in seen:
            seen.add(cid)
            c = self.cells[cid]
            if not c.vertex:
                break
            x += num(c.geo, 'x')
            y += num(c.geo, 'y')
            cid = c.parent
        return x, y

    def abs_box(self, cid):
        c = self.cells[cid]
        px, py = self.origin(c.parent)
        g = c.geo
        return (px + num(g, 'x'), py + num(g, 'y'), num(g, 'width'), num(g, 'height'))


def decode_diagram(text):
    data = base64.b64decode(text)
    xml = zlib.decompress(data, -15).decode('utf-8')
    return ET.fromstring(urllib.parse.unquote(xml))


def read_drawio(path):
    try:
        root = ET.parse(path).getroot()
    except (ET.ParseError, OSError) as ex:
        raise FlowTableError(f'không đọc được {path}: {ex}')
    others = []
    if root.tag == 'mxGraphModel':
        model = root
    else:
        pages = root.findall('diagram')
        if not pages:
            raise FlowTableError(f'{path} không có trang nào')
        model = pages[0].find('mxGraphModel')
        if model is None:
            try:
                model = decode_diagram((pages[0].text or '').strip())
            except (ValueError, zlib.error, ET.ParseError) as ex:
                raise FlowTableError(f'không giải nén được trang đầu của {path}: {ex}')
        others = pages[1:]
    groot = model.find('root')
    if groot is None:
        raise FlowTableError(f'{path} không có mxGraphModel/root')
    cells = {}
    for i, el in enumerate(list(groot)):
        if el.tag == 'mxCell':
            cell, cid = el, el.get('id')
        else:
            cell, cid = el.find('mxCell'), el.get('id')
            if cell is None:
                continue
        if cid:
            cells[cid] = OldCell(cid, el, cell, i)
    return OldDiagram(cells, others)


@dataclass
class MergeReport:
    pinned: list = field(default_factory=list)
    placed: list = field(default_factory=list)
    shifted: list = field(default_factory=list)
    lane_changed: list = field(default_factory=list)
    rerouted: list = field(default_factory=list)
    kept_edges: list = field(default_factory=list)
    labels_centered: list = field(default_factory=list)
    removed: list = field(default_factory=list)
    lanes_grown: list = field(default_factory=list)
    freehand_kept: list = field(default_factory=list)
    freehand_dropped: list = field(default_factory=list)
    freehand_detached: list = field(default_factory=list)
    overlaps: list = field(default_factory=list)

    def lines(self):
        def row(label, xs):
            return f'merge: {label} ({len(xs)}): ' + (', '.join(xs) if xs else '-')
        return [
            row('giữ vị trí', self.pinned),
            row('element mới', self.placed),
            row('element mới phải dịch xuống vì chồng', self.shifted),
            row('đổi lane trong bảng hoặc bị kéo sang lane khác, đặt lại', self.lane_changed),
            row('edge giữ waypoint', self.kept_edges),
            row('edge để draw.io tự đi dây, cần chỉnh tay', self.rerouted),
            row('edge đã sửa tay, nhãn về giữa đường', self.labels_centered),
            row('xoá vì không còn trong bảng', self.removed),
            row('lane được nới', self.lanes_grown),
            row('cell tự vẽ giữ lại', self.freehand_kept),
            row('cell tự vẽ bỏ đi (cha đã bị xoá)', self.freehand_dropped),
            row('cell tự vẽ mất đầu nối', self.freehand_detached),
            row('element chồng nhau', self.overlaps),
        ]


class Merger:
    PAD = 10
    STEP = 10

    def __init__(self, lay, old):
        self.lay = lay
        self.old = old
        self.cfg = lay.cfg
        self.rep = MergeReport()

    # ---- nhận diện cell do tool sinh

    def is_generated(self, c):
        if c.id in LAYER_IDS:
            return True
        lane_like = c.parent == 'pool' and c.style_raw.startswith('swimlane')
        shaped = c.id == 'pool' or c.id in self.table_ids or bool(TABLE_ID_RE.match(c.id)) or lane_like
        if self.has_mark:
            return shaped and 'flowtable' in c.style
        return shaped

    def run(self):
        lay, old, cfg = self.lay, self.old, self.cfg
        cells = old.cells
        self.table_ids = set(lay.items) | {l.id for l in lay.lanes} | {e.id for e in lay.edges}
        self.has_mark = any('flowtable' in c.style for c in cells.values())
        self.gen = {cid for cid, c in cells.items() if self.is_generated(c)}

        pool = cells.get('pool')
        if pool is not None and pool.vertex and 'pool' in self.gen:
            lay.origin = old.abs_box('pool')[:2]
        ox, oy = lay.origin
        self.ox, self.oy = ox, oy

        # Ảnh chụp layout mới tính, dùng làm khoảng lệch cho element mới.
        self.fresh_c = {i: (it.x + it.w / 2, it.y + it.h / 2) for i, it in lay.items.items()}
        self.fresh_lane_x = list(lay.lane_x)
        self.fresh_lane_w = list(lay.lane_w)
        self.fresh_wp = {e.id: e.pts[1:-1] for e in lay.edges}
        self.fresh_frac = {e.id: (e.exit_frac, e.entry_frac) for e in lay.edges}

        self.layout_lanes()
        self.movables = []
        self.final = {}
        self.place_old_items()
        self.place_new_items()
        self.route_edges()
        extras = self.collect_freehand()
        self.grow_lanes()
        self.fit_vertical()
        self.finish()
        for cid in sorted(self.gen):
            if cid in LAYER_IDS or cid == 'pool' or cid in self.table_ids:
                continue
            self.rep.removed.append(cid)
        return extras, old.other_pages, self.rep

    # ---- lane

    def old_lane(self, cid):
        c = self.old.cells.get(cid)
        if c is None or not c.vertex or cid not in self.gen or c.parent != 'pool':
            return None
        return c

    def layout_lanes(self):
        lay, old = self.lay, self.old
        new_x, new_w = [], []
        x = 0.0
        for i, lane in enumerate(lay.lanes):
            oc = self.old_lane(lane.id)
            w = num(oc.geo, 'width') if oc is not None else lay.lane_w[i]
            new_x.append(x)
            new_w.append(w)
            x += w
        # Các khoảng lane cũ theo toạ độ trong pool, kèm độ dịch sang vị trí mới.
        olds = []
        for cid, c in old.cells.items():
            if c.vertex and c.parent == 'pool' and cid in self.gen and c.style_raw.startswith('swimlane'):
                olds.append((num(c.geo, 'x'), num(c.geo, 'width'), cid))
        olds.sort()
        idx = {l.id: i for i, l in enumerate(lay.lanes)}
        self.old_spans = []
        for k, (x0, w0, cid) in enumerate(olds):
            if cid in idx:
                target = idx[cid]
                dx = new_x[target] - x0
            else:
                nxt = next((idx[c2] for _, _, c2 in olds[k + 1:] if c2 in idx), None)
                target = nxt if nxt is not None else len(lay.lanes) - 1
                base = new_x[nxt] if nxt is not None else x
                dx = base - x0
            self.old_spans.append((x0, x0 + w0, dx, target))
        lay.lane_x, lay.lane_w = new_x, new_w

    def map_x(self, x):
        """(độ dịch, lane mới) cho một toạ độ x cũ trong pool."""
        if not self.old_spans:
            return 0.0, 0
        for x0, x1, dx, t in self.old_spans:
            if x0 <= x < x1:
                return dx, t
        if x < self.old_spans[0][0]:
            return self.old_spans[0][2], self.old_spans[0][3]
        return self.old_spans[-1][2], self.old_spans[-1][3]

    # ---- element

    def add_movable(self, lane, shift, rel=False):
        """rel=True: toạ độ tương đối theo lane, lane dịch thì tự dịch theo."""
        self.movables.append((lane, shift, rel))

    def item_movable(self, it):
        def shift(dx, dy, it=it):
            it.x += dx
            it.y += dy
        self.add_movable(it.lane, shift)

    def place_old_items(self):
        lay, old = self.lay, self.old
        self.pinned = set()
        for it in sorted(lay.items.values(), key=lambda i: i.order):
            c = old.cells.get(it.id)
            if c is None or not c.vertex or it.id not in self.gen:
                continue
            lane_id = lay.lanes[it.lane].id
            if c.parent != lane_id:
                self.rep.lane_changed.append(it.id)
                continue
            ax, ay, w, h = old.abs_box(it.id)
            cx, cy = ax + w / 2 - self.ox, ay + h / 2 - self.oy
            oc = self.old_lane(lane_id)
            dx = lay.lane_x[it.lane] - num(oc.geo, 'x') if oc is not None else 0.0
            it.x, it.y = cx + dx - it.w / 2, cy - it.h / 2
            self.final[it.id] = it
            self.pinned.add(it.id)
            self.item_movable(it)
            self.rep.pinned.append(it.id)

    def neighbors(self, nid):
        lay = self.lay
        out = [e.src for e in lay.ins[nid] if not e.back] + [e.dst for e in lay.outs[nid] if not e.back]
        out += [e.src for e in lay.ins[nid] if e.back] + [e.dst for e in lay.outs[nid] if e.back]
        return out

    def find_anchor(self, v):
        lay = self.lay
        for e in sorted(lay.ins[v.id], key=lambda e: (e.back, e.order)):
            if e.src in self.final:
                return e.src
        for e in sorted(lay.outs[v.id], key=lambda e: (e.back, e.order)):
            if e.dst in self.final:
                return e.dst
        seen, frontier = {v.id}, [v.id]
        while frontier:
            nxt = []
            for n in frontier:
                for m in self.neighbors(n):
                    if m in seen:
                        continue
                    if m in self.final:
                        return m
                    seen.add(m)
                    nxt.append(m)
            frontier = sorted(nxt, key=lambda i: lay.items[i].order)
        return None

    def obstacles(self, skip):
        boxes = [it.box() for i, it in self.final.items() if i != skip]
        boxes += self.freehand_boxes
        return boxes

    def put(self, it, cx, cy):
        lay = self.lay
        it.x, it.y = cx - it.w / 2, cy - it.h / 2
        obs = self.obstacles(it.id)
        m = self.PAD
        moved = 0
        while any(area((it.x - m, it.y - m, it.x + it.w + m, it.y + it.h + m), b) > 0 for b in obs):
            it.y += self.STEP
            moved += 1
        if moved:
            self.rep.shifted.append(it.id)
        self.final[it.id] = it
        self.item_movable(it)
        self.rep.placed.append(it.id)

    def lane_rel_x(self, it):
        lay = self.lay
        fx = self.fresh_c[it.id][0] - self.fresh_lane_x[it.lane]
        lx, lw = lay.lane_x[it.lane], lay.lane_w[it.lane]
        cx = lx + fx
        if lw >= it.w + 2 * self.PAD:
            cx = min(max(cx, lx + it.w / 2 + self.PAD), lx + lw - it.w / 2 - self.PAD)
        return cx

    def place_new_items(self):
        lay = self.lay
        self.freehand_boxes = self.freehand_vertex_boxes()
        order = {n: k for k, n in enumerate(lay.topo_order)}
        pending = [it for it in lay.items.values() if it.id not in self.final]
        flow = sorted((it for it in pending if not it.attach), key=lambda i: (order.get(i.id, 1e9), i.order))
        atts = sorted((it for it in pending if it.attach), key=lambda i: i.order)
        for it in flow:
            a = self.find_anchor(it)
            if a is None:
                bottom = max((b[3] for b in self.obstacles(it.id)), default=self.cfg.pool_header + self.cfg.lane_header)
                self.put(it, self.lane_rel_x(it), bottom + self.cfg.min_channel + it.h / 2)
                continue
            self.put_relative(it, a)
        for it in atts:
            self.put_relative(it, it.attach)

    def put_relative(self, it, a):
        lay = self.lay
        an = lay.items[a]
        fa, fv = self.fresh_c[a], self.fresh_c[it.id]
        ac = (an.x + an.w / 2, an.y + an.h / 2)
        if an.lane == it.lane:
            cx = ac[0] + fv[0] - fa[0]
        else:
            cx = self.lane_rel_x(it)
        self.put(it, cx, ac[1] + fv[1] - fa[1])

    # ---- edge

    def route_edges(self):
        lay, old = self.lay, self.old
        for e in lay.edges:
            c = old.cells.get(e.id)
            keep = (c is not None and c.edge and e.id in self.gen
                    and c.cell.get('source') == e.src and c.cell.get('target') == e.dst
                    and e.src in self.pinned and e.dst in self.pinned)
            if not keep:
                e.auto = True
                self.rep.rerouted.append(e.id)
                continue
            px, py = old.origin(c.parent)
            pts = []
            for x, y in c.points():
                ax, ay = px + x - self.ox, py + y - self.oy
                dx, lane = self.map_x(ax)
                pts.append([ax + dx, ay, lane])
            st = c.style
            e.constraints = {k: st[k] for k in CONSTRAINT_KEYS if k in st and st[k] is not None}
            e.waypoints = [(x, y) for x, y, _ in pts]
            self.rep.kept_edges.append(e.id)
            if not self.untouched(e):
                e.label_mode = 'none'
                if e.lines:
                    self.rep.labels_centered.append(e.id)
            for p in pts:
                def shift(dx, dy, p=p, e=e):
                    p[0] += dx
                    p[1] += dy
                    e.waypoints = [(q[0], q[1]) for q in e._wp_refs]
                self.add_movable(p[2], shift)
            e._wp_refs = pts

    def untouched(self, e):
        fresh = self.fresh_wp[e.id]
        if len(fresh) != len(e.waypoints):
            return False
        if any(abs(a[0] - b[0]) > 2 or abs(a[1] - b[1]) > 2 for a, b in zip(fresh, e.waypoints)):
            return False
        (ex, ey), (nx, ny) = self.fresh_frac[e.id]
        want = {'exitX': ex, 'exitY': ey, 'entryX': nx, 'entryY': ny}
        for k, v in want.items():
            try:
                if k not in e.constraints or abs(float(e.constraints[k]) - v) > 1e-3:
                    return False
            except ValueError:
                return False
        return True

    # ---- cell tự vẽ

    def freehand_ids(self):
        return [cid for cid, c in sorted(self.old.cells.items(), key=lambda kv: kv[1].order)
                if cid not in self.gen]

    def frame_of(self, c):
        """'lane:<id>' | 'pool' | 'abs' | 'nested'."""
        p = c.parent
        if p in LAYER_IDS:
            return 'abs'
        if p == 'pool' and 'pool' in self.gen:
            return 'pool'
        if p in self.gen and self.old_lane(p) is not None:
            return 'lane:' + p
        return 'nested'

    def freehand_vertex_boxes(self):
        boxes = []
        idx = {l.id: i for i, l in enumerate(self.lay.lanes)}
        for cid in self.freehand_ids():
            c = self.old.cells[cid]
            if not c.vertex:
                continue
            ax, ay, w, h = self.old.abs_box(cid)
            x, y = ax - self.ox, ay - self.oy
            fr = self.frame_of(c)
            if fr.startswith('lane:') and fr[5:] in idx:
                oc = self.old_lane(fr[5:])
                x += self.lay.lane_x[idx[fr[5:]]] - num(oc.geo, 'x')
            elif fr in ('pool', 'abs'):
                x += self.map_x(x + w / 2)[0]
            boxes.append((x, y, x + w, y + h))
        return boxes

    def collect_freehand(self):
        lay, old = self.lay, self.old
        idx = {l.id: i for i, l in enumerate(lay.lanes)}
        alive = set(lay.items) | set(idx) | {e.id for e in lay.edges} | {'pool'} | set(LAYER_IDS)
        self.fh_geoms = []
        kept = set()
        for cid in self.freehand_ids():
            p = old.cells[cid].parent
            if p in self.gen and p not in alive:
                ok = self.old_lane(p) is not None
            else:
                ok = p in self.gen or p in LAYER_IDS or p in kept
            if ok:
                kept.add(cid)
            else:
                self.rep.freehand_dropped.append(cid)
        extras = []
        for cid in self.freehand_ids():
            if cid not in kept:
                continue
            c = old.cells[cid]
            p = c.parent
            el = copy.deepcopy(c.elem)
            cell = el if el.tag == 'mxCell' else el.find('mxCell')
            g = cell.find('mxGeometry')
            fr = self.frame_of(c)
            if fr.startswith('lane:') and fr[5:] not in idx:
                ax, ay = old.origin(p)
                cell.set('parent', 'pool')
                if g is not None and c.vertex:
                    g.set('x', fmt(num(g, 'x') + ax - self.ox))
                    g.set('y', fmt(num(g, 'y') + ay - self.oy))
                self.shift_points(g, ax - self.ox, ay - self.oy, None)
                fr = 'fixed'
                if c.vertex and g is not None:
                    self.fh_geoms.append((g, 'pool', None))
            if c.edge:
                self.detach(c, cell, g, alive, kept)
            self.track_freehand(c, cell, g, fr, idx)
            extras.append(el)
            self.rep.freehand_kept.append(cid)
        return extras

    @staticmethod
    def shift_points(g, dx, dy, pred):
        if g is None:
            return
        for pt in g.iter('mxPoint'):
            if pt.get('as') == 'offset':
                continue
            if pred is None or pred(pt):
                pt.set('x', fmt(num(pt, 'x') + dx))
                pt.set('y', fmt(num(pt, 'y') + dy))

    def detach(self, c, cell, g, alive, kept):
        old = self.old
        for end, pt_name in (('source', 'sourcePoint'), ('target', 'targetPoint')):
            ref = cell.get(end)
            if not ref or ref in kept or (ref in alive and ref in self.gen):
                continue
            if ref not in old.cells:
                continue
            ax, ay, w, h = old.abs_box(ref)
            px, py = old.origin(c.parent)
            del cell.attrib[end]
            if g is None:
                g = ET.SubElement(cell, 'mxGeometry', relative='1', **{'as': 'geometry'})
            for pt in g.findall('mxPoint'):
                if pt.get('as') == pt_name:
                    g.remove(pt)
            ET.SubElement(g, 'mxPoint', x=fmt(ax + w / 2 - px), y=fmt(ay + h / 2 - py), **{'as': pt_name})
            self.rep.freehand_detached.append(f'{c.id}.{end}')

    def track_freehand(self, c, cell, g, fr, idx):
        """Cho cell tự vẽ di chuyển cùng lane khi lane dịch hoặc nới."""
        if g is None or fr == 'nested':
            return
        if fr.startswith('lane:'):
            lane = idx[fr[5:]]

            def shift(dx, dy, g=g):
                self._shift_geo(g, dx, dy)
            self.add_movable(lane, shift, rel=True)
            self.fh_geoms.append((g, 'lane', lane))
            return
        if fr == 'fixed':
            return
        ox, oy = (0.0, 0.0) if fr == 'pool' else (self.ox, self.oy)
        if c.vertex:
            x = num(g, 'x') - ox + num(g, 'width') / 2
            dx, lane = self.map_x(x)
            self._shift_geo(g, dx, 0)
            self.add_movable(lane, lambda ddx, ddy, g=g: self._shift_geo(g, ddx, ddy))
            self.fh_geoms.append((g, fr, lane))
        else:
            for pt in g.iter('mxPoint'):
                if pt.get('as') == 'offset':
                    continue
                dx, lane = self.map_x(num(pt, 'x') - ox)
                pt.set('x', fmt(num(pt, 'x') + dx))
                self.add_movable(lane, lambda ddx, ddy, pt=pt: (
                    pt.set('x', fmt(num(pt, 'x') + ddx)), pt.set('y', fmt(num(pt, 'y') + ddy))))

    @staticmethod
    def _shift_geo(g, dx, dy):
        if g.get('relative') == '1':
            for pt in g.iter('mxPoint'):
                if pt.get('as') == 'offset':
                    continue
                pt.set('x', fmt(num(pt, 'x') + dx))
                pt.set('y', fmt(num(pt, 'y') + dy))
            return
        g.set('x', fmt(num(g, 'x') + dx))
        g.set('y', fmt(num(g, 'y') + dy))

    # ---- nới lane, co giãn pool

    def grow_lanes(self):
        lay = self.lay
        self.lane_children = defaultdict(list)
        for i in range(len(lay.lanes)):
            members = [it for it in self.final.values() if it.lane == i]
            if not members:
                continue
            lx, lw = lay.lane_x[i], lay.lane_w[i]
            left = min(it.x for it in members) - lx
            right = max(it.x + it.w for it in members) - lx
            gl = max(0.0, self.PAD - left)
            gr = max(0.0, right - (lw - self.PAD))
            if not gl and not gr:
                continue
            self.rep.lanes_grown.append(f'{lay.lanes[i].id} +{fmt(gl + gr)}px')
            lay.lane_w[i] += gl + gr
            for j in range(i + 1, len(lay.lanes)):
                lay.lane_x[j] += gl + gr
            for lane, shift, rel in self.movables:
                if lane == i and gl:
                    shift(gl, 0)
                elif lane > i and not rel:
                    shift(gl + gr, 0)

    def fit_vertical(self):
        lay, cfg = self.lay, self.cfg
        top = cfg.pool_header + cfg.lane_header + self.PAD
        ys = [it.y for it in self.final.values()]
        ys += [y for e in lay.edges if e.waypoints for _, y in e.waypoints]
        if ys and min(ys) < top:
            dy = top - min(ys)
            for _lane, shift, _rel in self.movables:
                shift(0, dy)

    def finish(self):
        lay, cfg = self.lay, self.cfg
        bottoms = [it.y + it.h for it in self.final.values()]
        bottoms += [y for e in lay.edges if e.waypoints for _, y in e.waypoints]
        bottoms += [b[3] for b in self.freehand_vertex_boxes_now()]
        # Cùng khoảng chừa như kênh cuối của layout mới, để merge không đổi chiều cao khi bảng không đổi.
        lay.pool_h = max([cfg.pool_header + cfg.lane_header + 60] + [b + cfg.min_channel for b in bottoms])
        lay.pool_w = sum(lay.lane_w)
        items = sorted(self.final.values(), key=lambda i: i.order)
        for k, a in enumerate(items):
            for b in items[k + 1:]:
                if area(a.box(), b.box()) > 0:
                    self.rep.overlaps.append(f'{a.id}/{b.id}')

    def freehand_vertex_boxes_now(self):
        out = []
        for g, fr, lane in self.fh_geoms:
            x, y, w, h = num(g, 'x'), num(g, 'y'), num(g, 'width'), num(g, 'height')
            if fr == 'lane':
                x, y = x + self.lay.lane_x[lane], y + self.cfg.pool_header
            elif fr == 'abs':
                x, y = x - self.ox, y - self.oy
            out.append((x, y, x + w, y + h))
        return out


def merge_into(lay, old):
    return Merger(lay, old).run()


# ---------------------------------------------------------------- render + verify

def drawio_bin():
    return shutil.which('drawio') or shutil.which('draw.io')


def export(src, out, fmt_, scale=None):
    exe = drawio_bin()
    if not exe:
        raise FlowTableError('không tìm thấy drawio CLI')
    cmd = [exe, '-x', '-f', fmt_, '-o', out]
    if scale:
        cmd += ['-s', str(scale)]
    cmd += [src, '--no-sandbox']
    res = None
    for _ in range(2):
        try:
            res = subprocess.run(cmd, capture_output=True, text=True, timeout=90)
            break
        except subprocess.TimeoutExpired:
            continue
    if res is None:
        raise FlowTableError(f'drawio export quá thời gian: {" ".join(cmd)}')
    if res.returncode != 0 or not os.path.exists(out):
        raise FlowTableError(f'drawio export lỗi: {res.stderr.strip() or res.stdout.strip()}')


PATH_NUM_RE = re.compile(r'[-+]?\d*\.?\d+(?:e[-+]?\d+)?')


def verify_svg(lay, svg_path, tol=2.0):
    """So đường dây trong SVG do draw.io vẽ với toạ độ đã tính."""
    tree = ET.parse(svg_path)
    ns = '{http://www.w3.org/2000/svg}'
    cells = {}
    for g in tree.iter(ns + 'g'):
        cid = g.get('data-cell-id')
        if cid:
            cells[cid] = g
    if not cells:
        return ['SVG không có data-cell-id, bỏ qua kiểm render']
    anchor = lay.items[min(lay.items, key=lambda i: lay.items[i].order)]
    rect = None
    for el in cells.get(anchor.id, ET.Element('x')).iter():
        if el.tag in (ns + 'rect',) and el.get('width'):
            rect = (float(el.get('x')), float(el.get('y')))
            break
        if el.tag == ns + 'ellipse':
            rect = (float(el.get('cx')) - float(el.get('rx')), float(el.get('cy')) - float(el.get('ry')))
            break
        if el.tag == ns + 'path' and el.get('d'):
            nums = [float(n) for n in PATH_NUM_RE.findall(el.get('d'))]
            xs, ys = nums[0::2], nums[1::2]
            rect = (min(xs), min(ys))
            break
    if rect is None:
        return [f'không tìm thấy hình của {anchor.id} trong SVG, bỏ qua kiểm render']
    dx, dy = rect[0] - anchor.x, rect[1] - anchor.y
    problems = []
    for e in lay.edges:
        g = cells.get(e.id)
        if g is None:
            problems.append(f'{e.id}: không có trong SVG')
            continue
        path = next((p for p in g.iter(ns + 'path') if p.get('d') and p.get('fill') == 'none'), None)
        if path is None:
            problems.append(f'{e.id}: không đọc được path')
            continue
        nums = [float(n) for n in PATH_NUM_RE.findall(path.get('d'))]
        got = [(nums[i] - dx, nums[i + 1] - dy) for i in range(0, len(nums) - 1, 2)]
        want = e.pts
        # Đầu mũi tên làm draw.io rút ngắn đoạn cuối, chỉ so các điểm trước đó.
        if len(got) != len(want):
            problems.append(f'{e.id}: draw.io vẽ {len(got)} điểm, tính ra {len(want)} điểm')
            continue
        for k, (a, b) in enumerate(zip(got[:-1], want[:-1])):
            if abs(a[0] - b[0]) > tol or abs(a[1] - b[1]) > tol:
                problems.append(f'{e.id}: điểm {k} lệch, vẽ ({a[0]:.0f},{a[1]:.0f}) thay vì ({b[0]:.0f},{b[1]:.0f})')
                break
        last_g, last_w, prev_w = got[-1], want[-1], want[-2]
        if abs(last_w[0] - prev_w[0]) < 0.01:
            bad = abs(last_g[0] - last_w[0]) > tol
        else:
            bad = abs(last_g[1] - last_w[1]) > tol
        if bad:
            problems.append(f'{e.id}: đoạn cuối lệch trục')
    return problems


# ---------------------------------------------------------------- cli

def print_issues(issues):
    for x in sorted(issues, key=lambda i: i.level != 'error'):
        print(x)
    ne = sum(1 for x in issues if x.level == 'error')
    nw = len(issues) - ne
    print(f'check: {ne} lỗi, {nw} cảnh báo')
    return ne


def read_table(args):
    return load(args.file, getattr(args, 'sheet', None), getattr(args, 'delimiter', None),
                getattr(args, 'encoding', None))


def cmd_check(args):
    try:
        table = read_table(args)
    except FlowTableError as ex:
        print(f'ERROR   {ex}')
        return 1
    print(f'check: đọc bảng từ {table.source}')
    return 1 if print_issues(table.issues) else 0


def choose_mode(args, out):
    """Trả về 'new' | 'force' | 'merge', hoặc None nếu phải dừng."""
    if not os.path.exists(out):
        return 'new'
    if args.mode:
        return args.mode
    if not sys.stdin.isatty():
        print(f'ERROR   {out} đã tồn tại; chọn --mode merge (giữ vị trí, lane, waypoint đã sửa tay) '
              'hoặc --mode force (sinh lại toàn bộ)')
        return None
    ans = input(f'{out} đã tồn tại. [m]erge giữ chỉnh sửa tay / [f]orce sinh lại toàn bộ / [q]uit: ').strip().lower()
    return {'m': 'merge', 'merge': 'merge', 'f': 'force', 'force': 'force'}.get(ans)


def cmd_build(args):
    try:
        table = read_table(args)
    except FlowTableError as ex:
        print(f'ERROR   {ex}')
        return 1
    title, rows = table.title, table.rows
    print(f'check: đọc bảng từ {table.source}')
    if print_issues(table.issues):
        print('build: dừng vì bảng có lỗi')
        return 1
    cfg = Config()
    for k in vars(cfg):
        v = getattr(args, k, None)
        if v is not None:
            setattr(cfg, k, v)
    tm = TextMeasure(font_path=args.font)
    if not tm.font:
        print('WARNING không đọc được font Verdana, dùng độ rộng ước lượng')
    out = args.output or os.path.splitext(args.file)[0] + '.drawio'
    mode = choose_mode(args, out)
    if mode is None:
        print('build: dừng, chưa chọn chế độ ghi')
        return 4
    old = None
    if mode == 'merge':
        try:
            old = read_drawio(out)
        except FlowTableError as ex:
            print(f'ERROR   {ex}; không ghi đè. Dùng --mode force nếu muốn sinh lại toàn bộ')
            return 4
    lay = Layout(rows, cfg, tm).run()
    fresh_issues = lay.check()
    extras, pages, report = [], [], None
    if old is not None:
        extras, pages, report = merge_into(lay, old)
    backup = None
    if mode in ('merge', 'force') and not args.no_backup:
        backup = out + '.bak'
        shutil.copy2(out, backup)
    xml = to_drawio(lay, args.title or title, extras, pages)
    with open(out, 'w', encoding='utf-8') as f:
        f.write(xml)
    labels = {'new': 'tạo mới', 'force': 'force, sinh lại toàn bộ', 'merge': 'merge, giữ chỉnh sửa tay'}
    print(f'build: chế độ {labels[mode]}')
    print(f'build: đã ghi {out} ({len(lay.lanes)} lane, {len(lay.items)} phần tử, {len(lay.edges)} cạnh, '
          f'{fmt(lay.pool_w)}x{fmt(lay.pool_h)}px)')
    if backup:
        print(f'build: bản sao file cũ {backup}')
    if args.layout_json:
        with open(args.layout_json, 'w', encoding='utf-8') as f:
            json.dump(lay.to_json(), f, ensure_ascii=False, indent=1)
    for w in lay.warnings:
        print(f'WARNING layout: {w}')
    code = 0
    if report is not None:
        for line in report.lines():
            print(line)
        print('layout: merge không tự kiểm dây và nhãn; xem danh sách ở trên và ảnh PNG')
    else:
        for level, msg in fresh_issues:
            print(f'{level.upper():7} layout: {msg}')
        nerr = sum(1 for l, _ in fresh_issues if l == 'error')
        print(f'layout: {nerr} lỗi, {len(fresh_issues) - nerr} cảnh báo')
        code = 2 if nerr else 0
    if args.png or args.verify:
        if not drawio_bin():
            print('WARNING không có drawio CLI, bỏ qua xuất ảnh và kiểm render')
            return code
        if report is not None and args.verify:
            print('render: bỏ qua kiểm render ở chế độ merge (đường dây giữ từ file hoặc do draw.io tự đi)')
            args.verify = False
        try:
            return export_and_verify(args, out, lay, code)
        except FlowTableError as ex:
            print(f'ERROR   {ex}')
            return code or 3
    return code


def export_and_verify(args, out, lay, code):
    if args.png:
        png = os.path.splitext(out)[0] + '.png' if args.png is True else args.png
        export(out, png, 'png', scale=2)
        print(f'png: {png}')
    if args.verify:
        with tempfile.TemporaryDirectory() as d:
            svg = os.path.join(d, 'render.svg')
            export(out, svg, 'svg')
            probs = verify_svg(lay, svg)
        for p in probs:
            print(f'ERROR   render: {p}')
        print(f'render: {len(probs)} điểm lệch')
        if probs and not code:
            code = 3
    return code


def main(argv=None):
    ap = argparse.ArgumentParser(description='Flow Table (.md, .csv, .xlsx) --> draw.io activity-swimlane')
    sub = ap.add_subparsers(dest='cmd', required=True)
    def input_flags(sp):
        sp.add_argument('--sheet', help='xlsx: tên sheet chứa bảng, mặc định sheet đầu tiên có hàng header')
        sp.add_argument('--delimiter', help='csv: dấu phân cách, mặc định tự đoán trong , ; tab')
        sp.add_argument('--encoding', help='csv: bảng mã, mặc định thử utf-8-sig, utf-8, cp1252')

    c = sub.add_parser('check', help='kiểm tra bảng theo flow-table-format')
    c.add_argument('file')
    input_flags(c)
    c.set_defaults(fn=cmd_check)
    b = sub.add_parser('build', help='kiểm tra, dựng layout và ghi file .drawio')
    b.add_argument('file')
    input_flags(b)
    b.add_argument('-o', '--output', help='mặc định: cạnh file nguồn, cùng tên, đuôi .drawio')
    b.add_argument('--title', help='tiêu đề pool, mặc định lấy heading # đầu tiên')
    b.add_argument('--layout-json', help='ghi toạ độ đã tính ra file json')
    b.add_argument('--png', nargs='?', const=True, help='xuất ảnh PNG (mặc định cạnh file .drawio)')
    b.add_argument('--verify', action='store_true', help='xuất SVG bằng drawio CLI và so với toạ độ đã tính')
    b.add_argument('--font', help='đường dẫn Verdana.ttf nếu không nằm ở vị trí mặc định')
    b.add_argument('--mode', choices=['merge', 'force'],
                   help='khi file đích đã có: merge giữ vị trí/lane/waypoint đã sửa tay, force sinh lại toàn bộ')
    b.add_argument('--no-backup', action='store_true', help='không tạo <file>.drawio.bak trước khi ghi đè')
    for k, v in vars(Config()).items():
        b.add_argument('--' + k.replace('_', '-'), dest=k, type=int, default=None, help=f'mặc định {v}')
    b.set_defaults(fn=cmd_build)
    args = ap.parse_args(argv)
    return args.fn(args)


if __name__ == '__main__':
    sys.exit(main())
