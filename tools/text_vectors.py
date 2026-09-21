"""Sinh vector đối chiếu cho riêng việc đo chữ và ngắt dòng.

    python3 tools/text_vectors.py conformance/text-vectors.json

Bộ case sơ đồ chốt module text quá lỏng. Wrap thu hẹp về bề rộng nhỏ nhất vẫn
giữ nguyên số dòng, nên lệch vài pixel ở ngân sách ngắt dòng thường không đổi
đầu ra, và một bản port sai lệch nhỏ vẫn qua cổng. Đã kiểm bằng mutation test:
đổi CondWrap thêm 10 và TaskMaxW thêm 2 đều không làm case nào đỏ.

Vector ở đây quét bề rộng từng pixel một, nên mọi sai lệch dù một pixel đều lật
được ít nhất một dòng trong bảng.
"""
import json
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'reference'))

import flowtable2drawio as ft  # noqa: E402

SAMPLES = [
    'Xử lý',
    'Đối soát giao dịch với cổng thanh toán rồi ghi nhận kết quả vào sổ cái nội bộ',
    'https://api.example.com/v1/transactions/reconcile?from=2026-01-01',
    'DBTHANHTOANDOISOATGIAODICHNGOAITE',
    'Dòng một\nDòng hai | có gạch | đứng',
    'a::b::c::d_e.f,g(h)i{j}k=l&m?n-o/p',
    'Kiểm tra dữ liệu đầu vào',
    'AAAA BBBB CCCC DDDD EEEE FFFF GGGG HHHH',
]

LO, HI = 20, 300


def main(argv):
    if len(argv) != 1:
        print(__doc__)
        return 2
    tm = ft.TextMeasure()
    if not isinstance(tm.font, ft.TableMetrics):
        raise SystemExit('ERROR không đọc được data/verdana.json')
    out = []
    for s in SAMPLES:
        lines = s.split('\n')
        widths = []
        for maxw in range(LO, HI + 1):
            wrapped = tm.wrap(lines, maxw)
            w, h = tm.box(wrapped)
            widths.append({'maxw': maxw, 'lines': wrapped, 'hard': tm.hard,
                           'w': ft.fmt(w), 'h': ft.fmt(h)})
        out.append({'text': s, 'widths': widths})
    doc = {'size': tm.size, 'line_h': tm.line_h, 'samples': out}
    with open(argv[0], 'w', encoding='utf-8') as f:
        json.dump(doc, f, ensure_ascii=False, separators=(',', ':'), sort_keys=True)
        f.write('\n')
    n = sum(len(s['widths']) for s in out)
    print(f'{argv[0]}: {len(out)} chuỗi x {HI - LO + 1} bề rộng = {n} vector')
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
