"""Áp một đột biến lên bản tham chiếu Python rồi xem các case có đổi kết quả không.

    python3 tools/py_mutant.py --from 'chuỗi gốc' --to 'chuỗi thay' [--stage place] case.md ...

Dùng để thiết kế case cho một đột biến mà tools/mutate.sh báo là bỏ lọt. Thay vì
đoán một bảng rồi chạy vòng qua bản Go, áp cùng đột biến lên chính bản Python:
case nào làm bản Python đổi đầu ra ở chặng đó thì chắc chắn làm bản Go đỏ, vì
bản Go phải khớp bản Python.

Chuỗi gốc phải xuất hiện đúng một lần trong reference/flowtable2drawio.py, cùng
luật với mutate.sh.
"""
import argparse
import io
import os
import sys
import types

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
REF = os.path.join(ROOT, 'reference', 'flowtable2drawio.py')
DUMP = os.path.join(ROOT, 'reference', 'dump.py')


def load_module(name, src, path):
    m = types.ModuleType(name)
    m.__file__ = path
    exec(compile(src, path, 'exec'), m.__dict__)
    return m


def bind_dump(ft):
    """Một bản dump.py riêng, gắn với đúng bản tham chiếu truyền vào.

    dump.py lấy bản tham chiếu qua import lúc nạp, nên phải đặt nó vào
    sys.modules trước rồi mới nạp dump.
    """
    sys.modules['flowtable2drawio'] = ft
    return load_module('dump_' + str(id(ft)), io.open(DUMP, encoding='utf-8').read(), DUMP)


def render(dump, path, stage):
    try:
        return dump.render(path, (stage,)).get(stage)
    except Exception as ex:  # đột biến có thể làm bản tham chiếu gãy hẳn
        return f'LỖI: {type(ex).__name__}: {ex}'


def main(argv):
    ap = argparse.ArgumentParser(description=__doc__.split('\n')[0])
    ap.add_argument('--from', dest='a', required=True)
    ap.add_argument('--to', dest='b', required=True)
    ap.add_argument('--stage', default='place')
    ap.add_argument('cases', nargs='+')
    args = ap.parse_args(argv)

    src = io.open(REF, encoding='utf-8').read()
    n = src.count(args.a)
    if n != 1:
        print(f'ERROR tìm thấy {n} chỗ khớp trong bản tham chiếu, cần đúng 1')
        return 2
    base = bind_dump(load_module('flowtable2drawio', src, REF))
    mut = bind_dump(load_module('flowtable2drawio', src.replace(args.a, args.b), REF))

    killed = 0
    for path in args.cases:
        x, y = render(base, path, args.stage), render(mut, path, args.stage)
        name = os.path.basename(path)
        if x is None:
            print(f'--         {name}: không có chặng {args.stage}, bảng có lỗi')
        elif x != y:
            killed += 1
            print(f'BỊ GIẾT    {name}')
        else:
            print(f'CÒN SỐNG   {name}')
    print(f'\n{killed} trên {len(args.cases)} case giết được đột biến')
    return 0 if killed else 1


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
