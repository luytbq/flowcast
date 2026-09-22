"""Sinh vector đối chiếu cho bộ đọc và ghi XML kiểu ElementTree.

    python3 tools/etree_vectors.py conformance/etree-vectors.json


Mỗi vector là một tài liệu XML ngẫu nhiên cùng hai đầu ra của Python: ghi lại ngay
sau khi đọc, và ghi lại sau ET.indent(space='  '). Merge chép nguyên cell người dùng
tự vẽ rồi ghi lại qua đúng con đường này, nên bản Go phải khớp cả hai.
"""
import json
import random
import sys
import xml.etree.ElementTree as ET

TAGS = ['mxCell', 'mxGeometry', 'mxPoint', 'Array', 'object', 'diagram']
KEYS = ['id', 'value', 'style', 'parent', 'x', 'y', 'as', 'label']
PIECES = ['a', 'b', 'Ghi chú', ' ', '  ', '\n', '\n  ', '\t', '&amp;', '&lt;', '&gt;', '&quot;', '&#10;',
          '&#13;', '&#x41;', '"', "'", '>', 'ử', '\r', '\r\n']
ATTR_PIECES = ['a', 'b', ' ', 'ử', '&amp;', '&lt;', '&gt;', '&quot;', '&#10;', '&#9;', '&#13;', '\n', '\t',
               "'", '>', '=', ';', '\r', '\r\n', '\r\r']


def text(rng, pieces, n=4):
    return ''.join(rng.choice(pieces) for _ in range(rng.randint(0, n)))


def attr_value(rng):
    s = text(rng, ATTR_PIECES)
    return s.replace('"', '')


def element(rng, depth):
    tag = rng.choice(TAGS)
    keys = rng.sample(KEYS, rng.randint(0, 3))
    attrs = ''.join(f' {k}="{attr_value(rng)}"' for k in keys)
    if depth > 2 or rng.random() < 0.3:
        return f'<{tag}{attrs}/>' if rng.random() < 0.5 else f'<{tag}{attrs}>{text(rng, PIECES)}</{tag}>'
    body = text(rng, PIECES)
    for _ in range(rng.randint(0, 3)):
        r = rng.random()
        # Dấu nháy lẻ trong chú thích, CDATA và chỉ thị xử lý: bộ chuẩn hóa thuộc
        # tính phải bỏ qua chúng, nếu không nó tưởng một chuỗi trong nháy đã mở.
        if r < 0.1:
            body += rng.choice(['<!-- chú thích -->', '<!--"-->', "<!--'\n-->"])
        elif r < 0.15:
            body += rng.choice(['<?pi dữ liệu?>', '<?pi "?>'])
        elif r < 0.22:
            body += rng.choice(['<![CDATA[a < b & c]]>', '<![CDATA["\n\t]]>'])
        body += element(rng, depth + 1) + text(rng, PIECES)
    return f'<{tag}{attrs}>{body}</{tag}>'


def main(argv):
    rng = random.Random(17)
    out = []
    while len(out) < 1500:
        doc = element(rng, 0)
        if rng.random() < 0.3:
            doc = '<?xml version="1.0" encoding="UTF-8"?>\n' + doc
        try:
            plain = ET.tostring(ET.fromstring(doc), encoding='unicode')
            root = ET.fromstring(doc)
            ET.indent(root, space='  ')
            indented = ET.tostring(root, encoding='unicode')
        except ET.ParseError:
            continue
        out.append({'doc': doc, 'plain': plain, 'indented': indented})
    with open(argv[0], 'w', encoding='utf-8') as f:
        json.dump(out, f, ensure_ascii=False, separators=(',', ':'))
    print(len(out), 'vector')


if __name__ == '__main__':
    main(sys.argv[1:])
