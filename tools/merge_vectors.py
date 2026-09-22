"""Sinh vector đối chiếu cho các bộ đọc kiểu Python mà merge dùng.

    python3 tools/merge_vectors.py conformance/merge-vectors.json

Merge đọc số trong file .drawio bằng float(), giải trang nén bằng
base64.b64decode và urllib.parse.unquote, và cộng bề rộng lane bằng sum(), thứ
từ Python 3.12 cộng có bù sai số. Cả ba chấp nhận nhiều thứ hơn hàm
tương ứng của Go, nên bản Go port lại chúng, và vector ở đây chốt bản port trên
chuỗi sinh ngẫu nhiên.
"""
import base64
import binascii
import json
import random
import sys
import urllib.parse

FLOAT_PIECES = ['0', '1', '7', '12', '.', '5', '_', 'e', 'E', '-', '+', ' ', '\t', '\n', 'inf', 'Infinity', 'nan',
                'NaN', 'x', '0x1', '١', '٣', ' ', '\x1c', '1e400', '-0', 'ử']
B64_PIECES = ['Q', 'U', 'J', 'D', 'QUJD', '=', '==', ' ', '\n', '*', 'YWJj', 'w', '+', '/', 'ZA', 'ZA==']
UQ_PIECES = ['a', '%', '%4', '%41', '%4g', '%e1%bb%ad', '%E1%BB', '%ff', '%C3', '%28', 'ử', ' ', '%%', '%c3%a9']


def pieces(rng, ps, n=6):
    return ''.join(rng.choice(ps) for _ in range(rng.randint(0, n)))


def main(argv):
    rng = random.Random(29)
    floats, b64s, uqs, sums = [], [], [], []
    for _ in range(4000):
        s = pieces(rng, FLOAT_PIECES, 4)
        try:
            v = float(s)
            floats.append([s, 'nan' if v != v else v.hex()])
        except ValueError:
            floats.append([s, None])
    for _ in range(3000):
        s = pieces(rng, B64_PIECES)
        try:
            b64s.append([s, base64.b64decode(s).hex(), None])
        except binascii.Error as ex:
            b64s.append([s, None, str(ex)])
    for _ in range(3000):
        s = pieces(rng, UQ_PIECES)
        uqs.append([s, urllib.parse.unquote(s)])
    for _ in range(2000):
        xs = [rng.choice([rng.uniform(0, 400), round(rng.uniform(0, 400), 2), 1e16, -1e16, 0.1, 1e-9])
              for _ in range(rng.randint(0, 8))]
        sums.append({'xs': [x.hex() for x in xs], 'want': float(sum(xs)).hex()})
    with open(argv[0], 'w', encoding='utf-8') as f:
        json.dump({'float': floats, 'b64': b64s, 'unquote': uqs, 'sum': sums}, f, ensure_ascii=False, separators=(',', ':'))
    print(len(floats) + len(b64s) + len(uqs) + len(sums), 'vector')


if __name__ == '__main__':
    main(sys.argv[1:])
