"""Sinh vector đối chiếu cho pha tự kiểm hình học.

    python3 tools/check_vectors.py conformance/check-vectors.json

Engine đúng thì tự kiểm không tìm thấy gì: trên cả bộ case chỉ 2 bảng có phát hiện,
và không bảng nào chạm tới một lỗi nào. Nên không kiểm được tự kiểm bằng đầu ra của
một engine đúng.

Tự kiểm là một hàm thuần trên hình học, nên ở đây cho nó ăn hình học hỏng: chạy các
bản tham chiếu đã bị đột biến, gồm các đột biến ở tools/mutants/ và vài đột biến phá
hỏng có chủ đích bên dưới, trên bộ case và trên bảng sinh ngẫu nhiên. Hình học nào
làm tự kiểm báo gì đó thì được giữ lại, cùng danh sách phát hiện. Bản Go chạy tự kiểm
trên đúng hình học đó và phải ra đúng danh sách đó.

Mọi số thực ghi bằng float.hex() để đọc lại đúng từng bit.
"""
import io
import os
import random
import runpy
import sys
import types

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
REF = os.path.join(ROOT, 'reference', 'flowtable2drawio.py')
sys.path.insert(0, os.path.join(ROOT, 'tools'))

import fuzz_cases  # noqa: E402

# Phá hỏng có chủ đích, mỗi cái nhắm một loại phát hiện mà các đột biến thông
# thường khó sinh ra.
BREAKERS = {
    'mọi node cùng một y': ("            it.y = ry0 + (rh - it.h) / 2", "            it.y = 90.0"),
    'lane hẹp một nửa': ("            self.lane_w.append(total)", "            self.lane_w.append(total / 2)"),
    'dây nối thẳng hai đầu': ("            pts = [p0] + [(rx(a), ry(b)) for a, b in e.sym] + [p1]", "            pts = [p0, p1]"),
    'mọi đoạn cùng một track': ("                    t.track = ti", "                    t.track = 0"),
    'nhãn không tránh node': ("                    cost += area(box, bb) * 4", "                    cost += 0"),
}

# Mỗi loại phát hiện giữ tối đa chừng này vector, để file không phình mà vẫn đủ
# đa dạng.
PER_KIND = 12


# Hình học dựng tay, nằm đúng trên các ngưỡng của tự kiểm. Engine hỏng không
# tình cờ rơi vào đúng ngưỡng, nên chúng được dựng thẳng; phát hiện mong đợi vẫn
# do bản Python tính. Node ở xa các dây trừ khi dây được dựng để cắt qua chúng.
FAR = [('X', 0, (500, 500, 40, 40)), ('Y', 0, (600, 500, 40, 40))]
SYNTHETIC = {
    # Đoạn giữa của cạnh cắt qua chính node nguồn. Chỉ đoạn đầu và đoạn cuối
    # được chạm node nguồn và đích.
    'tay: đoạn giữa cắt qua node nguồn': {
        'lanes': [(-50, 800)],
        'items': [('S', 0, (0, 0, 100, 50)), ('T', 0, (0, 200, 100, 50))],
        'edges': [('E1', 'S', 'T', [(100, 25), (120, 25), (120, 40), (-10, 40), (-10, 200), (50, 200)], None)],
    },
    # Hai dây khác nguồn khác đích chồng đúng 2 điểm ảnh.
    'tay: dây chồng 2 điểm ảnh': {
        'lanes': [(-50, 800)],
        'items': FAR,
        'edges': [('E1', 'A', 'B', [(0, 100), (52, 100)], None), ('E2', 'C', 'D', [(50, 100), (200, 100)], None)],
    },
    # Hai dây ngang cách nhau 0.3, dưới ngưỡng 0.5 nên coi là cùng đường.
    'tay: hai dây ngang cách 0.3': {
        'lanes': [(-50, 800)],
        'items': FAR,
        'edges': [('E1', 'A', 'B', [(0, 100), (100, 100)], None), ('E2', 'C', 'D', [(20, 100.3), (80, 100.3)], None)],
    },
    # Hai dây dọc cách nhau 0.3.
    'tay: hai dây dọc cách 0.3': {
        'lanes': [(-50, 800)],
        'items': FAR,
        'edges': [('E1', 'A', 'B', [(100, 0), (100, 100)], None), ('E2', 'C', 'D', [(100.3, 20), (100.3, 80)], None)],
    },
}


def synthetic():
    import types
    out = []
    ft = fuzz_cases.load_module('flowtable2drawio', io.open(REF, encoding='utf-8').read(), REF)
    for name, g in SYNTHETIC.items():
        items = {}
        for iid, lane, (x, y, w, h) in g['items']:
            items[iid] = types.SimpleNamespace(id=iid, lane=lane, x=float(x), y=float(y), w=float(w), h=float(h),
                                               box=lambda x=x, y=y, w=w, h=h: (x, y, x + w, y + h))
        edges = [types.SimpleNamespace(id=eid, src=a, dst=b, pts=[(float(px), float(py)) for px, py in pts],
                                       label=label) for eid, a, b, pts, label in g['edges']]
        fake = types.SimpleNamespace(items=items, edges=edges,
                                     lane_x=[float(x) for x, _ in g['lanes']], lane_w=[float(w) for _, w in g['lanes']])
        out.append({'from': name, 'geometry': geometry(fake),
                    'findings': [[lvl, msg] for lvl, msg in ft.Layout.check(fake)]})
    return out


def kind_of(msg):
    for key in ('chồng lên nhau', 'tràn ra ngoài lane', 'không vuông góc', 'dây cắt qua',
                'chồng dây', 'đè lên', 'đè nhãn', 'đè dây'):
        if key in msg:
            return key
    return 'khác'


def geometry(lay):
    hx = lambda v: float(v).hex()  # noqa: E731
    return {
        'lanes': [[hx(lay.lane_x[i]), hx(lay.lane_w[i])] for i in range(len(lay.lanes))],
        'items': [{'id': it.id, 'lane': it.lane, 'box': [hx(it.x), hx(it.y), hx(it.w), hx(it.h)]}
                  for it in lay.items.values()],
        'edges': [{'id': e.id, 'src': e.src, 'dst': e.dst,
                   'pts': [[hx(x), hx(y)] for x, y in (e.pts or [])],
                   'label': [hx(v) for v in e.label] if e.label else None}
                  for e in lay.edges],
    }


def run(ft, md):
    try:
        t = ft.read_markdown_text(md)
        if any(i.level == 'error' for i in t.issues + ft.validate(t.rows)):
            return None
        lay = ft.Layout(t.rows, ft.Config(), ft.TextMeasure()).run()
        return lay, lay.check()
    except Exception:
        return None


def main(argv):
    if len(argv) != 1:
        print(__doc__)
        return 2
    src = io.open(REF, encoding='utf-8').read()
    table = dict(BREAKERS)
    for f in ('route.py', 'geometry.py'):
        table.update(runpy.run_path(os.path.join(ROOT, 'tools', 'mutants', f))['M'])
    variants = [('gốc', src)] + [(k, src.replace(a, b)) for k, (a, b) in table.items() if src.count(a) == 1]

    cases = os.path.join(ROOT, 'conformance', 'cases')
    tables = [io.open(os.path.join(cases, n), encoding='utf-8').read()
              for n in sorted(os.listdir(cases)) if n.endswith('.md')]
    rng = random.Random(21)
    tables += [fuzz_cases.gen(rng) for _ in range(400)]
    # Bảng nhỏ trước, để mỗi loại phát hiện được đại diện bởi những hình học dễ đọc.
    tables.sort(key=len)

    kept, per_kind, seen = [], {}, set()
    for name, variant in variants:
        ft = fuzz_cases.load_module('flowtable2drawio', variant, REF)
        for md in tables:
            r = run(ft, md)
            if r is None or not r[1]:
                continue
            lay, findings = r
            kinds = sorted({kind_of(m) for _, m in findings})
            sig = (tuple(kinds), len(findings) > 3)
            if sig in seen or all(per_kind.get(k, 0) >= PER_KIND for k in kinds):
                continue
            seen.add(sig)
            for k in kinds:
                per_kind[k] = per_kind.get(k, 0) + 1
            kept.append({'from': name, 'geometry': geometry(lay),
                         'findings': [[lvl, msg] for lvl, msg in findings]})

    kept += synthetic()

    import json
    with open(argv[0], 'w', encoding='utf-8') as fh:
        json.dump(kept, fh, ensure_ascii=False, separators=(',', ':'))
        fh.write('\n')
    print(f'{argv[0]}: {len(kept)} vector')
    for k in sorted(per_kind):
        print(f'  {per_kind[k]:3}  {k}')
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
