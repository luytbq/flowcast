"""Sinh bảng ngẫu nhiên để tìm case giết từng đột biến của bản tham chiếu.

    python3 tools/fuzz_cases.py tools/mutants/route.py --stage route --seconds 150 --out /tmp/found

Dùng khi mutate.sh báo nhiều đột biến bỏ lọt ở một pha mà điều kiện bị hình học
ràng buộc chặt, khiến thiết kế case bằng tay chậm và suy luận dễ sai. Ở pha đi
dây, cách này giết 16 trên 18 đột biến trong 150 giây, gồm cả hai nhánh từng bị
nghi là thừa.

File đột biến định nghĩa M = {tên: (chuỗi gốc, chuỗi thay)} trên
reference/flowtable2drawio.py. Mỗi đột biến bị giết được giữ lại một bảng nhỏ
nhất, rồi thu nhỏ tiếp bằng cách bỏ từng dòng khi bảng vẫn hợp lệ và vẫn giết.
Đột biến sống sót qua mọi bảng là ứng viên tương đương, đáng bỏ công chứng minh.
Sống qua một lần chạy ngắn chưa nói lên gì: ở pha đi dây có đột biến chỉ chết
với bảng 24 dòng, và một lần chạy 20 giây không gặp nó. Chạy đủ lâu trước khi
bắt tay vào chứng minh.

Kết quả tất định theo --seed.
"""
import argparse
import io
import os
import random
import runpy
import sys
import time
import types

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
REF = os.path.join(ROOT, 'reference', 'flowtable2drawio.py')
DUMP = os.path.join(ROOT, 'reference', 'dump.py')
HEAD = '| id | type | parent | content | metadata |\n|---|---|---|---|---|\n'
LANES = 'ABCD'


def load_module(name, src, path):
    m = types.ModuleType(name)
    m.__file__ = path
    exec(compile(src, path, 'exec'), m.__dict__)
    return m


def bind(src):
    """Một cặp bản tham chiếu và dump.py gắn với nó. dump lấy bản tham chiếu qua
    import lúc nạp, nên phải đặt vào sys.modules trước."""
    ft = load_module('flowtable2drawio', src, REF)
    sys.modules['flowtable2drawio'] = ft
    d = load_module('dump_' + str(id(ft)), io.open(DUMP, encoding='utf-8').read(), DUMP)
    return ft, d


def stage_of(pair, md, stage):
    ft, d = pair
    try:
        t = ft.read_markdown_text(md)
        t.title = t.title or 'x'
        if any(i.level == 'error' for i in t.issues + ft.validate(t.rows)):
            return None
        lay = ft.Layout(t.rows, ft.Config(), ft.TextMeasure()).run()
        return d.stages(t, lay, (stage,))[stage]
    except Exception as ex:  # đột biến có thể làm bản tham chiếu gãy hẳn
        return 'LỖI ' + type(ex).__name__


def gen(rng):
    """Một bảng ngẫu nhiên đúng luật thứ tự dòng: node, rồi db và text của nó, rồi
    các cạnh ra. Có cả cạnh song song trùng đích và cạnh quay ngược."""
    nl, n = rng.randint(1, 4), rng.randint(3, 9)
    nodes = []
    for i in range(n):
        lane = LANES[rng.randrange(nl)]
        typ = 'start' if i == 0 else rng.choice(['task', 'task', 'task', 'condition', 'end', 'external'])
        nodes.append((f'{lane}-{i + 1}', typ, lane))
    rows = [f'| {LANES[i]} | lane | | Lane {LANES[i]} | |' for i in range(nl)]
    eid = 0
    for i, (nid, typ, lane) in enumerate(nodes):
        rows.append(f'| {nid} | {typ} | {lane} | {nid} x | |')
        for a in range(rng.choice([0, 0, 0, 0, 1, 1, 2, 3])):
            rows.append(f'| {nid}.{a} | {rng.choice(["db", "text"])} | {lane} | d{a} | attach={nid} |')
        if typ in ('end', 'external'):
            continue
        k = rng.choice([2, 2, 3]) if typ == 'condition' else rng.choice([1, 1, 1, 2, 2, 3])
        later = list(range(i + 1, n))
        targets = rng.sample(later, min(k, len(later)))
        if rng.random() < 0.08 and targets:
            targets.append(targets[0])
        if rng.random() < 0.15 and i > 0:
            targets.append(-rng.randrange(1, i + 1))
        for j in targets:
            eid += 1
            back = j <= 0
            dst = nodes[abs(j) - 1][0] if back else nodes[j][0]
            meta = f'from={nid}; to={dst}' + ('; back=true' if back else '')
            rows.append(f'| E{eid} | edge | | {"n" if typ == "condition" else ""} | {meta} |')
    return HEAD + '\n'.join(rows) + '\n'


def shrink(md, killed):
    rows = [r for r in md.split('\n')[2:] if r.strip()]
    changed = True
    while changed:
        changed = False
        for i in range(len(rows)):
            trial = HEAD + '\n'.join(rows[:i] + rows[i + 1:]) + '\n'
            if killed(trial):
                rows = rows[:i] + rows[i + 1:]
                changed = True
                break
    return HEAD + '\n'.join(rows) + '\n'


def main(argv):
    ap = argparse.ArgumentParser(description=__doc__.split('\n')[0])
    ap.add_argument('mutants')
    ap.add_argument('--stage', default='route')
    ap.add_argument('--seconds', type=float, default=120)
    ap.add_argument('--seed', type=int, default=1)
    ap.add_argument('--out', required=True)
    args = ap.parse_args(argv)

    src = io.open(REF, encoding='utf-8').read()
    table = runpy.run_path(args.mutants)['M']
    for name, (a, _) in table.items():
        if src.count(a) != 1:
            print(f'ERROR {name}: chuỗi gốc khớp {src.count(a)} chỗ, cần đúng 1')
            return 2
    base = bind(src)
    muts = {name: bind(src.replace(a, b)) for name, (a, b) in table.items()}

    rng = random.Random(args.seed)
    best, tried, valid, t0 = {}, 0, 0, time.time()
    while time.time() - t0 < args.seconds and len(best) < len(muts):
        md = gen(rng)
        tried += 1
        b = stage_of(base, md, args.stage)
        if b is None:
            continue
        valid += 1
        for name, pair in muts.items():
            if (name not in best or len(md) < len(best[name])) and stage_of(pair, md, args.stage) != b:
                best[name] = md
    print(f'thử {tried} bảng, {valid} hợp lệ, {time.time() - t0:.0f}s')

    os.makedirs(args.out, exist_ok=True)
    for name in table:
        if name not in best:
            print(f'SỐNG       {name}: ứng viên tương đương')
            continue

        def killed(md, pair=muts[name]):
            b = stage_of(base, md, args.stage)
            return b is not None and stage_of(pair, md, args.stage) != b

        md = shrink(best[name], killed)
        io.open(os.path.join(args.out, name + '.md'), 'w', encoding='utf-8').write(md)
        print(f'BỊ GIẾT    {name}: {md.count(chr(10)) - 2} dòng')
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
