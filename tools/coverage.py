"""Đo bộ đối chiếu có chạm tới những nhánh nào của thuật toán xếp hình.

    python3 tools/coverage.py

Bộ đáp án chỉ có giá trị bằng đúng phần thuật toán nó chạm tới. Một bộ trông
đồ sộ mà không case nào ép chia track thì không chốt được gì về việc chia track.
Công cụ này in ra con số thật cho từng nhánh, để lỗ hổng nhìn thấy được thay vì
được giả định là không có.

Ngưỡng ở THRESHOLDS là mức tối thiểu để một nhánh coi như đã được chốt. Thiếu
thì thoát với mã 1.
"""
import collections
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
sys.path.insert(0, os.path.join(ROOT, 'reference'))

import flowtable2drawio as ft  # noqa: E402

CASES = os.path.join(ROOT, 'conformance', 'cases')

THRESHOLDS = {
    'đi dây A (thẳng đứng)': 1,
    'đi dây B (thẳng ngang)': 1,
    'đi dây C (chữ L)': 1,
    'đi dây D (qua kênh và máng)': 1,
    'nhánh dạt trái (drift -1)': 1,
    'nhánh dạt phải (drift 1)': 1,
    'nhánh không rõ hướng (drift 0)': 1,
    'track thứ hai trở lên': 1,
    'db hoặc text bám node': 1,
    'cạnh back': 1,
    'cạnh có nhãn': 1,
    'phần tử tô nhấn': 1,
    'ngắt dòng cứng giữa từ': 1,
}


def measure():
    hits = collections.Counter()
    for name in sorted(f for f in os.listdir(CASES) if f.endswith('.md')):
        table = ft.load(os.path.join(CASES, name))
        if any(i.level == 'error' for i in table.issues):
            continue
        tm = ft.TextMeasure()
        lay = ft.Layout(table.rows, ft.Config(), tm).run()
        for e in lay.edges:
            hits[f'đi dây {e.case} ({ {"A": "thẳng đứng", "B": "thẳng ngang", "C": "chữ L",
                                        "D": "qua kênh và máng"}[e.case] })'] += 1
            if e.back:
                hits['cạnh back'] += 1
            if e.lines:
                hits['cạnh có nhãn'] += 1
            if e.highlight:
                hits['phần tử tô nhấn'] += 1
            if not e.back and lay.items[e.dst].lane == lay.items[e.src].lane:
                d = lay.branch_drift(e)
                hits[{-1: 'nhánh dạt trái (drift -1)', 1: 'nhánh dạt phải (drift 1)',
                      0: 'nhánh không rõ hướng (drift 0)'}[d]] += 1
        for it in lay.items.values():
            if it.attach:
                hits['db hoặc text bám node'] += 1
            if it.highlight:
                hits['phần tử tô nhấn'] += 1
        hits['track thứ hai trở lên'] += sum(1 for s in lay.segs if s.track > 0)
        # TextMeasure.hard chỉ nói về dòng vừa ngắt xong, nên phải đọc ngay sau
        # từng lần đo chứ không đọc một lần ở cuối.
        for r in table.rows:
            if r.id not in lay.items:
                continue
            lay.size_item(r.type, r.lines)
            if tm.hard:
                hits['ngắt dòng cứng giữa từ'] += 1
    return hits


def main():
    hits = measure()
    width = max(len(k) for k in THRESHOLDS)
    missing = []
    for name, need in THRESHOLDS.items():
        got = hits.get(name, 0)
        mark = 'OK ' if got >= need else 'THIẾU'
        if got < need:
            missing.append(name)
        print(f'{mark:6} {name.ljust(width)}  {got}')
    print(f'\nnhánh chưa chạm tới: {len(missing)}')
    return 1 if missing else 0


if __name__ == '__main__':
    sys.exit(main())
