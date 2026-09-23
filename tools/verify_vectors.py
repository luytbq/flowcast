"""Sinh vector đối chiếu cho kiểm render.

    python3 tools/verify_vectors.py conformance/verify-vectors.json

Kiểm render xuất SVG bằng drawio CLI rồi so đường dây draw.io vẽ với toạ độ đã
tính. Phần so là một hàm thuần trên SVG, nên chốt được mà không cần drawio lúc
chạy test: file này xuất SVG thật cho một số case, cố tình làm lệch từng kiểu
một, và ghi lại những gì verify_svg của bản tham chiếu báo.

SVG được lược bớt, chỉ giữ svg, g, rect, ellipse và path, vì kiểm render không
đọc gì khác và SVG gốc nặng vài chục KB.
"""
import copy
import json
import os
import random
import subprocess
import sys
import tempfile
import xml.etree.ElementTree as ET

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.join(ROOT, 'reference'))

import flowtable2drawio as ft  # noqa: E402

NS = '{http://www.w3.org/2000/svg}'
KEEP = {NS + t for t in ('svg', 'g', 'rect', 'ellipse', 'path')}
CASES = ['01-linear', '03-condition-two', '05-merge-node', '06-back-edge', '11-db-attach', '15-route-l-shape',
         '16-route-general', '17-track-sharing', '18-labels-crowded', '21-styles', '22-many-lanes',
         '23-external', '28-tracks-fan-in', '62-route-c-attach-side', '66-route-back-no-bottom']


def strip(el):
    for c in list(el):
        if c.tag not in KEEP:
            el.remove(c)
        else:
            strip(c)
        c.tail = None
    el.text = None


def cells(root):
    return {g.get('data-cell-id'): g for g in root.iter(NS + 'g') if g.get('data-cell-id')}


def edge_path(g):
    return next((p for p in g.iter(NS + 'path') if p.get('d') and p.get('fill') == 'none'), None)


def nudge(d, k, delta):
    """Cộng delta vào số thứ k trong thuộc tính d của path."""
    parts = ft.PATH_NUM_RE.split(d)
    nums = ft.PATH_NUM_RE.findall(d)
    nums[k] = repr(float(nums[k]) + delta)
    out = parts[0]
    for n, p in zip(nums, parts[1:]):
        out += n + p
    return out


def variants(rng, root, lay):
    yield 'nguyên', root
    edges = [e for e in lay.edges]
    anchor = min(lay.items.values(), key=lambda i: i.order).id
    for name in ('lệch điểm giữa', 'lệch điểm cuối', 'lệch dưới ngưỡng', 'thêm điểm', 'bỏ cạnh', 'bỏ path',
                 'lệch hình neo', 'bỏ hình neo', 'không có id', 'id trùng',
                 'neo có rect thiếu width', 'neo có rect ngoài namespace', 'neo vẽ bằng path'):
        r = copy.deepcopy(root)
        cs = cells(r)
        e = rng.choice(edges)
        g = cs.get(e.id)
        p = edge_path(g) if g is not None else None
        if p is None:
            continue
        n = len(ft.PATH_NUM_RE.findall(p.get('d')))
        if name == 'lệch điểm giữa' and n >= 6:
            p.set('d', nudge(p.get('d'), rng.choice([0, 1, 2, 3]), rng.choice([-6.0, 3.0, 40.0])))
        elif name == 'lệch điểm cuối':
            p.set('d', nudge(p.get('d'), rng.choice([n - 2, n - 1]), rng.choice([-5.0, 9.0])))
        elif name == 'lệch dưới ngưỡng':
            p.set('d', nudge(p.get('d'), rng.randrange(n), 1.5))
        elif name == 'thêm điểm':
            p.set('d', p.get('d') + ' L 1 2')
        elif name == 'bỏ cạnh':
            for parent in r.iter():
                if g in list(parent):
                    parent.remove(g)
        elif name == 'bỏ path':
            p.set('fill', 'black')
        elif name == 'lệch hình neo':
            a = cs.get(anchor)
            for el in a.iter():
                if el.tag == NS + 'rect' and el.get('width'):
                    el.set('x', repr(float(el.get('x')) + 3))
                    break
                if el.tag == NS + 'ellipse':
                    el.set('cx', repr(float(el.get('cx')) + 3))
                    break
                if el.tag == NS + 'path' and el.get('d'):
                    el.set('d', nudge(el.get('d'), 0, 3.0))
                    break
        elif name == 'bỏ hình neo':
            a = cs.get(anchor)
            for el in list(a.iter()):
                for c in list(el):
                    if c.tag in (NS + 'rect', NS + 'ellipse', NS + 'path'):
                        el.remove(c)
        elif name == 'neo có rect thiếu width':
            # rect không có width là hình nền hay vùng bắt chuột, không phải hình node.
            cs[anchor].insert(0, ET.Element(NS + 'rect', x='999', y='999'))
        elif name == 'neo có rect ngoài namespace':
            cs[anchor].insert(0, ET.Element('{urn:x-khac}rect', x='999', y='999', width='10'))
        elif name == 'neo vẽ bằng path':
            a = cs[anchor]
            for el in list(a.iter()):
                for c in list(el):
                    if c.tag in (NS + 'rect', NS + 'ellipse', NS + 'path'):
                        el.remove(c)
            # Điểm đầu của path không phải góc trên trái: hình neo là góc nhỏ nhất.
            ET.SubElement(a, NS + 'path', d='M 200 300 L 100 300 L 100 200 L 200 200 Z')
        elif name == 'không có id':
            for x in r.iter(NS + 'g'):
                x.attrib.pop('data-cell-id', None)
        elif name == 'id trùng':
            dup = copy.deepcopy(g)
            edge_path(dup).set('d', nudge(edge_path(dup).get('d'), 0, 25.0))
            r.append(dup)
        else:
            continue
        yield name, r


def main(argv):
    rng = random.Random(41)
    out = []
    with tempfile.TemporaryDirectory() as tmp:
        for case in CASES:
            table = ft.load(os.path.join(ROOT, 'conformance', 'cases', case + '.md'))
            lay = ft.Layout(table.rows, ft.Config(), ft.TextMeasure()).run()
            src = os.path.join(ROOT, 'conformance', 'golden', case + '.drawio')
            svg = os.path.join(tmp, case + '.svg')
            subprocess.run(['drawio', '-x', '-f', 'svg', '-o', svg, src, '--no-sandbox'],
                           check=True, capture_output=True, timeout=120)
            root = ET.parse(svg).getroot()
            strip(root)
            for name, r in variants(rng, root, lay):
                path = os.path.join(tmp, 'v.svg')
                ET.ElementTree(r).write(path, encoding='unicode')
                text = open(path, encoding='utf-8').read()
                out.append({'case': case, 'variant': name, 'svg': text, 'problems': ft.verify_svg(lay, path)})
    with open(argv[0], 'w', encoding='utf-8') as f:
        json.dump(out, f, ensure_ascii=False, indent=0)
    print(len(out), 'vector,', sum(1 for v in out if v['problems']), 'có điểm lệch')


if __name__ == '__main__':
    main(sys.argv[1:])
