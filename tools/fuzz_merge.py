"""Fuzz cho merge: so CLI Go với bản tham chiếu trên file .drawio sửa tay ngẫu nhiên.

    python3 tools/fuzz_merge.py --go <flowcast> [--n 300] [--seed 0]
    python3 tools/fuzz_merge.py --emit conformance/merge --seeds 3,17

Mỗi seed chọn một case markdown trong conformance/cases, sinh file .drawio bằng
bản tham chiếu, rồi sửa ngẫu nhiên như người dùng làm trong draw.io: kéo node,
kéo node sang lane khác, nới lane, thêm điểm gấp, sửa điểm neo, vẽ ghi chú, mũi
tên và nhóm, xoá cell. Bảng cũng bị sửa: thêm node, đổi lane, sửa chữ, xoá node.
Cuối cùng chạy build --mode merge ở cả hai bản và so từng byte đầu ra lẫn file.

--emit ghi các seed đã chọn thành kịch bản fuzz-<seed> trong conformance/merge,
để bộ đối chiếu giữ lại những seed phủ được nhánh mà kịch bản viết tay chưa phủ.
"""
import argparse
import contextlib
import io
import os
import random
import shutil
import subprocess
import sys
import tempfile
import xml.etree.ElementTree as ET

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
TOOL = os.path.join(ROOT, 'reference', 'flowtable2drawio.py')
CASES = os.path.join(ROOT, 'conformance', 'cases')
sys.path.insert(0, os.path.join(ROOT, 'reference'))

import flowtable2drawio as ft  # noqa: E402


def bases():
    out = []
    for f in sorted(os.listdir(CASES)):
        if f.endswith('.md') and f[:2].isdigit() and int(f[:2]) < 90:
            out.append(f)
    return out


def delta(rng, lo, hi):
    v = rng.uniform(lo, hi)
    return float(rng.choice([str(round(v)), str(round(v, 1)), str(round(v * 2) / 2), repr(v)]))


def fnum(s):
    try:
        return float(s)
    except (TypeError, ValueError):
        return 0.0


def num_str(rng, lo, hi):
    if rng.random() < 0.03:
        return rng.choice(['abc', '', ' 12 ', '1_0'])
    v = rng.uniform(lo, hi)
    return rng.choice([str(round(v)), str(round(v, 1)), str(round(v * 2) / 2), repr(v)])


def rows_of(md):
    """(chỉ số dòng, 5 ô) của các dòng dữ liệu tách được gọn."""
    out = []
    for i, line in enumerate(md.split('\n')):
        t = line.strip()
        if not t.startswith('|') or '\\|' in t:
            continue
        cells = [c.strip() for c in t.strip('|').split('|')]
        if len(cells) != 5 or cells[0] in ('id', '') or set(cells[0]) <= set('-: '):
            continue
        out.append((i, cells))
    return out


def edit_table(rng, md):
    lines = md.split('\n')
    rows = rows_of(md)
    lanes = [c[0] for _, c in rows if c[1] == 'lane']
    nodes = [(i, c) for i, c in rows if c[1] in ('task', 'condition', 'start', 'end', 'external')]
    if not lanes or not nodes:
        return md
    extra = []
    for k in range(rng.randint(0, 3)):
        op = rng.choice(['grow', 'add', 'add', 'lane', 'drop', 'newlane', 'droplane'])
        i, c = rng.choice(nodes)
        if op == 'grow':
            c[3] = (c[3] + ' kèm thêm chữ cho dài ra') if c[3] else 'Chữ mới'
        elif op == 'add':
            lane = rng.choice(lanes)
            nid, eid = f'{lane}-9{k}', f'E9{k}'
            if rng.random() < 0.5:
                extra += [f'| {nid} | task | {lane} | Mới {k} | |', f'| {eid} | edge | | | from={c[0]}; to={nid} |']
            else:
                extra += [f'| {nid} | task | {lane} | Mới {k} | |', f'| {eid} | edge | | | from={nid}; to={c[0]} |']
        elif op == 'lane':
            c[2] = rng.choice(lanes)
        elif op == 'drop':
            lines[i] = None
            for j, e in rows:
                if e[1] == 'edge' and (f'from={c[0]};' in e[4] + ';' or f'to={c[0]};' in e[4] + ';'):
                    lines[j] = None
            continue
        elif op == 'droplane' and len(lanes) > 1:
            lid = rng.choice(lanes)
            gone = {lid} | {n[0] for _, n in nodes if n[2] == lid}
            for j, e in rows:
                refs = {x.split('=', 1)[1].strip() for x in e[4].split(';') if '=' in x}
                if e[0] in gone or (e[1] != 'lane' and e[2] in gone) or refs & gone:
                    lines[j] = None
            lanes.remove(lid)
            continue
        elif op == 'newlane':
            lid = f'N{k}'
            extra.insert(0, f'| {lid} | lane | | Lane mới {k} | |')
            c[2] = lid
            lanes.append(lid)
        if lines[i] is not None:
            lines[i] = '| ' + ' | '.join(c) + ' |'
    if extra:
        # Lane mới phải đứng trước node dùng nó, nên chèn ngay sau dòng lane đầu.
        first_lane = next(i for i, c in rows if c[1] == 'lane')
        lanes_new = [x for x in extra if ' | lane | ' in x]
        rest = [x for x in extra if ' | lane | ' not in x]
        lines[first_lane] = '\n'.join([lines[first_lane]] + lanes_new) if lines[first_lane] else lines[first_lane]
        last = max(i for i, _ in rows)
        lines[last] = '\n'.join([lines[last] or ''] + rest)
    return '\n'.join(x for x in lines if x is not None)


def edit_drawio(rng, tree):
    mxfile = tree.getroot()
    root = mxfile.find('.//root')
    cells = [c for c in root if c.tag == 'mxCell']
    lanes = [c for c in cells if c.get('parent') == 'pool' and c.get('vertex') == '1']
    # Danh sách chứ không phải set: thứ tự duyệt set đổi theo PYTHONHASHSEED, và
    # seed phải cho ra cùng một kịch bản ở mọi tiến trình.
    lane_ids = [c.get('id') for c in lanes]
    items = [c for c in cells if c.get('parent') in lane_ids]
    edges = [c for c in cells if c.get('edge') == '1']
    notes = []

    def geo(c):
        return c.find('mxGeometry')

    def add_note(k):
        parent = rng.choice(['pool', '1', 'khong-co'] + [c.get('id') for c in lanes + items[:2]] + notes)
        cid = rng.choice([f'hand-{k}', f'hand-{k}', f'X_Y-{k}', f'E9{k}', f'Q-{k}.1', f'hand-{k}\n'])
        wrap = rng.random() < 0.3
        holder = ET.SubElement(root, 'object', id=cid, label='x') if wrap else None
        n = ET.SubElement(holder if wrap else root, 'mxCell', style='text;html=1;', vertex='1', parent=parent)
        if not wrap:
            n.set('id', cid)
            n.set('value', 'Ghi chú')
        ET.SubElement(n, 'mxGeometry', x=num_str(rng, -50, 900), y=num_str(rng, 0, 500),
                      width=num_str(rng, 20, 200), height=num_str(rng, 20, 80), **{'as': 'geometry'})
        notes.append(cid)

    for k in range(rng.randint(1, 7)):
        op = rng.choice(['move', 'move', 'move', 'reparent', 'lanew', 'lanex', 'wp', 'wp', 'constraint',
                         'note', 'note', 'arrow', 'group', 'unmark', 'unmark-all', 'pool', 'delete', 'dup'])
        if op == 'move' and items:
            g = geo(rng.choice(items))
            g.set('x', str(fnum(g.get('x')) + delta(rng, -300, 300)))
            g.set('y', str(fnum(g.get('y')) + delta(rng, -150, 150)))
        elif op == 'reparent' and items and len(lanes) > 1:
            rng.choice(items).set('parent', rng.choice(lanes).get('id'))
        elif op == 'lanew' and lanes:
            g = geo(rng.choice(lanes))
            g.set('width', str(max(10.0, fnum(g.get('width')) + delta(rng, -80, 200))))
        elif op == 'lanex' and lanes:
            geo(rng.choice(lanes)).set('x', num_str(rng, 0, 900))
        elif op == 'wp' and edges:
            g = geo(rng.choice(edges))
            for old in g.findall("Array[@as='points']"):
                g.remove(old)
            arr = ET.SubElement(g, 'Array', {'as': 'points'})
            for _ in range(rng.randint(0, 3)):
                ET.SubElement(arr, 'mxPoint', x=num_str(rng, 0, 1000), y=num_str(rng, -20, 600))
        elif op == 'constraint' and edges:
            c = rng.choice(edges)
            key = rng.choice(['exitX', 'exitY', 'entryX', 'entryY', 'exitPerimeter', 'exitDx'])
            parts = [p for p in c.get('style', '').split(';') if p and not p.startswith(key + '=')]
            if rng.random() < 0.7:
                parts.append(rng.choice([f'{key}={rng.choice(["0", "0.25", "1", "0.5", "abc", ""])}', key]))
            c.set('style', ';'.join(parts) + ';')
        elif op == 'note':
            add_note(k)
        elif op == 'arrow':
            ends = [c.get('id') for c in items] + notes + ['khong-co']
            a = ET.SubElement(root, 'mxCell', id=f'hand-a{k}', style='endArrow=classic;html=1;', edge='1',
                              parent=rng.choice(['1', 'pool'] + list(lane_ids)))
            for end in ('source', 'target'):
                if rng.random() < 0.8:
                    a.set(end, rng.choice(ends))
            g = ET.SubElement(a, 'mxGeometry', relative='1', **{'as': 'geometry'})
            if rng.random() < 0.5:
                arr = ET.SubElement(g, 'Array', {'as': 'points'})
                ET.SubElement(arr, 'mxPoint', x=num_str(rng, 0, 900), y=num_str(rng, 0, 500))
            if rng.random() < 0.3:
                ET.SubElement(g, 'mxPoint', x='3', y='4', **{'as': 'offset'})
            if rng.random() < 0.3:
                ET.SubElement(g, 'mxPoint', x=num_str(rng, 0, 900), y='5', **{'as': 'sourcePoint'})
        elif op == 'group':
            parent = rng.choice(['pool', '1'] + list(lane_ids))
            g = ET.SubElement(root, 'mxCell', id=f'hand-g{k}', value='', style='group;', vertex='1', parent=parent)
            ET.SubElement(g, 'mxGeometry', x=num_str(rng, 0, 800), y=num_str(rng, 0, 400), width='200',
                          height='80', **{'as': 'geometry'})
            c = ET.SubElement(root, 'mxCell', id=f'hand-g{k}c', value='trong', style='text;', vertex='1',
                              parent=f'hand-g{k}')
            ET.SubElement(c, 'mxGeometry', x='10', y='10', width='100', height='30', **{'as': 'geometry'})
        elif op == 'unmark':
            c = rng.choice(cells)
            if c.get('style'):
                c.set('style', c.get('style').replace('flowtable=1;', ''))
        elif op == 'unmark-all':
            for c in root.iter('mxCell'):
                if c.get('style'):
                    c.set('style', c.get('style').replace('flowtable=1;', ''))
        elif op == 'dup' and items:
            # Hai cell cùng id: cell sau thắng nhưng giữ chỗ của cell trước.
            src = rng.choice(items)
            d = ET.SubElement(root, 'mxCell', id=src.get('id'), value='bản sao', style='text;', vertex='1',
                              parent=rng.choice(['pool', '1'] + list(lane_ids)))
            ET.SubElement(d, 'mxGeometry', x=num_str(rng, 0, 400), y=num_str(rng, 0, 300), width='60',
                          height='30', **{'as': 'geometry'})
        elif op == 'pool':
            g = geo(next(c for c in cells if c.get('id') == 'pool'))
            g.set('x', num_str(rng, 0, 200))
            g.set('y', num_str(rng, 0, 200))
        elif op == 'delete':
            victim = rng.choice(items + edges + lanes) if items else None
            if victim is not None and victim in list(root):
                root.remove(victim)


def make(seed, d):
    """Sinh kịch bản của seed vào thư mục d: flow.md và flow.drawio."""
    rng = random.Random(seed)
    base = rng.choice(bases())
    md_path, drawio_path = os.path.join(d, 'flow.md'), os.path.join(d, 'flow.drawio')
    shutil.copy(os.path.join(CASES, base), md_path)
    with contextlib.redirect_stdout(io.StringIO()):
        code = ft.main(['build', md_path, '-o', drawio_path])
    if code not in (0, 2) or not os.path.exists(drawio_path):
        return None
    tree = ET.parse(drawio_path)
    edit_drawio(rng, tree)
    tree.write(drawio_path)
    md = io.open(md_path, encoding='utf-8').read()
    io.open(md_path, 'w', encoding='utf-8').write(edit_table(rng, md))
    return base


def run(argv, cwd):
    p = subprocess.run(argv, capture_output=True, text=True, stdin=subprocess.DEVNULL, cwd=cwd)
    files = {n: open(os.path.join(cwd, n), 'rb').read() for n in sorted(os.listdir(cwd))}
    return p.stdout.replace(cwd, '$T'), p.returncode, files


def compare(seed, go):
    tmp = tempfile.mkdtemp()
    try:
        src = os.path.join(tmp, 'src')
        os.makedirs(src)
        base = make(seed, src)
        if base is None:
            return None, None
        out = {}
        for side, argv in (('py', [sys.executable, TOOL]), ('go', [go])):
            d = os.path.join(tmp, side)
            shutil.copytree(src, d)
            out[side] = run(argv + ['build', 'flow.md', '--mode', 'merge'], d)
        return base, out['py'] == out['go'] or out
    finally:
        shutil.rmtree(tmp)


def main(argv):
    ap = argparse.ArgumentParser()
    ap.add_argument('--go')
    ap.add_argument('--n', type=int, default=300)
    ap.add_argument('--seed', type=int, default=0)
    ap.add_argument('--emit')
    ap.add_argument('--seeds', default='')
    a = ap.parse_args(argv)
    if a.emit:
        for s in [int(x) for x in a.seeds.split(',') if x]:
            d = os.path.join(a.emit, f'fuzz-{s}')
            if os.path.isdir(d):
                shutil.rmtree(d)
            os.makedirs(d)
            if make(s, d) is None:
                raise SystemExit(f'ERROR seed {s} không sinh được kịch bản')
        return 0
    bad = 0
    for s in range(a.seed, a.seed + a.n):
        base, res = compare(s, os.path.abspath(a.go))
        if res is None or res is True:
            continue
        bad += 1
        py, go = res['py'], res['go']
        print(f'LỆCH seed {s} ({base}): exit {py[1]} / {go[1]}')
        if py[0] != go[0]:
            for lp, lg in zip(py[0].split('\n'), go[0].split('\n')):
                if lp != lg:
                    print(f'  py: {lp}\n  go: {lg}')
                    break
        for n in sorted(set(py[2]) | set(go[2])):
            if py[2].get(n) != go[2].get(n):
                print(f'  file khác: {n}')
    print(f'{a.n} seed, {bad} lệch')
    return 1 if bad else 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
