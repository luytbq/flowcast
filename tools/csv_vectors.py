"""Sinh vector đối chiếu cho bộ đọc csv, từ csv.reader của chính Python.

    python3 tools/csv_vectors.py conformance/csv-vectors.json

encoding/csv của Go hiểu dấu nháy khác csv.reader của Python ở nhiều góc, chẳng
hạn "a"b ra ab trong Python nhưng ra a"b trong Go. Nên bản Go port nguyên máy
trạng thái của csv.reader, và vector ở đây chốt nó: chuỗi ngẫu nhiên trộn dấu
nháy, dấu phân cách, CR, LF, khoảng trắng và ký tự NUL, đọc với từng dấu phân
cách mà bản tham chiếu dùng, cộng các chuỗi dựng tay ở đúng những góc đó.
"""
import csv
import io
import json
import random
import sys

ALPHABET = ['a', 'b', 'é', ' ', ',', ';', '\t', '"', '"', '\n', '\r', '\r\n']
HAND = ['', '\n', '\r\n', '\r', 'a', 'a,b', 'a,b\n', 'a,b\r\nc,d', '"a"b,c', '"a""b",c', '"a\nb",c',
        '"a\r\nb",c', 'a"b,c', '"a', '"a"', '""', '"",""', ',,', 'a,\n,b', ' "a",b', '"a" ,b',
        'a\n\nb', 'a\r\rb', '﻿id,type', 'x,\x00,y', '"a""', 'a""b', '"\n"']


def read(text, d):
    try:
        return {'rows': list(csv.reader(io.StringIO(text, newline=''), delimiter=d))}
    except csv.Error as ex:
        return {'error': str(ex)}


def main(argv):
    if len(argv) != 1:
        print(__doc__)
        return 2
    rng = random.Random(9)
    texts = list(HAND) + [''.join(rng.choice(ALPHABET) for _ in range(rng.randint(0, 24))) for _ in range(3000)]
    out = [{'text': t, 'delim': d, **read(t, d)} for t in texts for d in (',', ';', '\t')]
    with open(argv[0], 'w', encoding='utf-8') as f:
        json.dump(out, f, ensure_ascii=False, separators=(',', ':'))
        f.write('\n')
    errs = sorted({v['error'] for v in out if 'error' in v})
    print(f'{argv[0]}: {len(out)} vector, {sum("error" in v for v in out)} ra lỗi: {errs}')
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
