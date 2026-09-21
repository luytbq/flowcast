"""Trích bảng độ rộng glyph từ một file TTF ra JSON.

Bản Go không đọc file font: nó đọc bảng này. Nhờ vậy cùng một bảng đầu vào cho
ra cùng một bố cục trên mọi máy, kể cả máy không cài Verdana, và binary phân
phối đi không mang theo file font.

    python3 tools/extract_metrics.py /System/Library/Fonts/Supplemental/Verdana.ttf data/verdana.json
"""
import json
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'reference'))

import flowtable2drawio as ft  # noqa: E402


def flatten(path):
    m = ft.TTFMetrics(path)
    adv = {}
    for cp, gid in m.cmap.items():
        adv[cp] = m.adv[gid] if gid < len(m.adv) else m.adv[-1]
    return {
        'family': 'Verdana',
        'upem': m.upem,
        'notdef': m.adv[0],
        'advances': {str(cp): w for cp, w in sorted(adv.items())},
    }


def main(argv):
    if len(argv) != 3:
        print(__doc__)
        return 2
    table = flatten(argv[1])
    with open(argv[2], 'w', encoding='utf-8') as f:
        json.dump(table, f, ensure_ascii=False, indent=0, sort_keys=False)
        f.write('\n')
    print(f'{argv[2]}: {len(table["advances"])} codepoint, upem {table["upem"]}')
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv))
