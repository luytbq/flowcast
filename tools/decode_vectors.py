"""Sinh vector đối chiếu cho việc giải mã bảng mã của file csv.

    python3 tools/decode_vectors.py conformance/decode-vectors.json

Bytes ngẫu nhiên, nghiêng về những chỗ dễ lệch: BOM ở đầu, byte 0x80 tới 0x9F nơi
cp1252 khác Latin-1 và có năm mã không được định nghĩa, và dãy utf-8 hỏng. Mỗi
chuỗi bytes được giải mã theo bốn bảng mã bằng bytes.decode(errors="strict").
"""
import json
import random
import sys


def main(argv):
    if len(argv) != 1:
        print(__doc__)
        return 2
    rng = random.Random(4)
    pool = [0x41, 0xC3, 0xA9, 0xEF, 0xBB, 0xBF, 0xE1, 0xBB, 0x87]
    out = []
    for _ in range(3000):
        raw = bytes(rng.choice([rng.randrange(256), rng.randrange(0x80, 0xA0), rng.choice(pool)])
                    for _ in range(rng.randint(0, 12)))
        if rng.random() < 0.2:
            raw = b'\xef\xbb\xbf' + raw
        row = {'hex': raw.hex()}
        for enc in ('utf-8-sig', 'utf-8', 'cp1252', 'latin-1'):
            try:
                row[enc] = raw.decode(enc)
            except UnicodeDecodeError:
                row[enc] = None
        out.append(row)
    with open(argv[0], 'w', encoding='utf-8') as f:
        json.dump(out, f, ensure_ascii=False, separators=(',', ':'))
        f.write('\n')
    print(f'{argv[0]}: {len(out)} vector')
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
