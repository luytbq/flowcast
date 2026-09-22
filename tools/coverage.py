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
import re
import unicodedata
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
    'đầu vào ở dạng NFD': 1,
    'db hoặc text tràn ra ngoài bốn ô cạnh node': 1,
}

# Luật trong validate, nhận diện bằng mẫu trên thông điệp.
#
# Đo riêng vì 16 trên 25 luật từng không được case nào chạm tới, mà điều đó
# không nhìn thấy được qua số lượng case.
VALIDATE_RULES = [
    (r'^id trống', 'validate: id trống'),
    (r'trùng id dành riêng', 'validate: id dành riêng của draw.io'),
    (r'^id trùng với', 'validate: id trùng'),
    (r'^type không hợp lệ', 'validate: type lạ'),
    (r'^bảng không có lane', 'validate: bảng không có lane'),
    (r'phải để trống parent', 'validate: lane hoặc edge có parent'),
    (r'^thiếu parent', 'validate: thiếu parent'),
    (r'không phải id của một lane', 'validate: parent không phải lane'),
    (r'^metadata ".*" không dùng', 'validate: key metadata lạ'),
    (r'^style ".*" không dùng', 'validate: style lạ'),
    (r'^back=', 'validate: back sai giá trị'),
    (r'thiếu from|thiếu to', 'validate: edge thiếu from hoặc to'),
    (r'^(from|to)=.*trỏ tới id không tồn tại', 'validate: from hoặc to treo'),
    (r'^(from|to)=.*là .*; cạnh chỉ nối', 'validate: from hoặc to sai loại'),
    (r'^db thiếu attach', 'validate: db thiếu attach'),
    (r'^attach=.*trỏ tới id không tồn tại', 'validate: attach treo'),
    (r'^attach=.*là .*; chỉ gắn', 'validate: attach sai loại'),
    (r'nhưng gắn vào .* thuộc lane', 'validate: attach khác lane'),
    (r'cạnh ra, cần ít nhất 2', 'validate: condition thiếu nhánh'),
    (r'không có cạnh ra', 'validate: node không có cạnh ra'),
    (r'^start có cạnh đi vào', 'validate: start có cạnh vào'),
    (r'^end có cạnh đi ra', 'validate: end có cạnh ra'),
    (r'của condition .* không có nhãn', 'validate: nhánh condition không nhãn'),
    (r'^cạnh ra phải nằm liền sau', 'validate: cạnh ra lệch chỗ'),
    (r'đứng trước nguồn', 'validate: quay ngược mà thiếu back=true'),
]

for _pat, _name in VALIDATE_RULES:
    THRESHOLDS[_name] = 1


def measure():
    hits = collections.Counter()
    for name in sorted(f for f in os.listdir(CASES) if f.endswith(('.md', '.csv', '.xlsx'))):
        try:
            raw = open(os.path.join(CASES, name), 'rb').read().decode('utf-8')
            if unicodedata.normalize('NFC', raw) != raw:
                hits['đầu vào ở dạng NFD'] += 1
        except UnicodeDecodeError:
            pass
        try:
            table = ft.load(os.path.join(CASES, name))
        except ft.FlowTableError:
            continue
        for issue in table.issues:
            for pat, label in VALIDATE_RULES:
                if re.search(pat, issue.msg):
                    hits[label] += 1
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
        hits['db hoặc text tràn ra ngoài bốn ô cạnh node'] += sum(
            1 for w in lay.warnings if 'không còn ô trống cạnh' in w)
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
