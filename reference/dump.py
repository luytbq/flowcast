"""Dump trạng thái trung gian của bản tham chiếu, làm cổng chặn cho bản port.

So từng byte file .drawio chỉ làm được ở cuối chuỗi. Port xong layout/place mà
chưa có writer thì không có cách nào biết nó đúng hay sai. Module này cắt trạng
thái ra thành từng chặng để mỗi module Go có điểm đối chiếu của riêng nó.

    python3 reference/dump.py <file> [--stage text|table|issues|place|route|geometry]

Không có --stage thì in tất cả.

Module này là giàn giáo cho bản port, không phải một phần của công cụ. Vì vậy nó
nằm tách khỏi flowtable2drawio.py và không được phép đổi hành vi của bản đó.

## Quy ước để hai ngôn ngữ so được với nhau

**Số.** Mọi số thực đi qua ft.fmt, tức hai chữ số thập phân rồi cắt số 0 thừa và
cắt dấu chấm thừa. Đây đúng là hàm sinh số trong file .drawio, nên dump và đầu
ra cuối cùng chuẩn hóa số giống hệt nhau. Bản Go phải tái tạo cả những góc kỳ
quặc của nó: fmt(-0.001) ra "-0", không phải "0" hay "-0.00".

**Thứ tự.** Phần tử và cạnh xếp theo order, tức thứ tự dòng trong bảng. Đoạn dây
xếp theo khóa chuẩn hóa, vì hai bản có thể sinh ra cùng tập đoạn dây theo thứ tự
append khác nhau. Khóa JSON luôn sắp xếp.

**Mã lỗi.** Chặng issues chốt level, loc, id và msg, không chốt mã máy. Mã máy
là thứ mới do docs/core-design.md quy định, không phải hành vi được port, nên
bản tham chiếu không có gì để đối chiếu ở đó.
"""
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)

import flowtable2drawio as ft  # noqa: E402

STAGES = ('text', 'table', 'issues', 'place', 'route', 'geometry')


def f(v):
    return ft.fmt(v)


def pair(t):
    return [f(t[0]), f(t[1])]


def stage_text(table, lay):
    """Ngắt dòng và đo chữ. Chốt module text."""
    return {
        'items': {it.id: {'kind': it.kind, 'lines': it.lines, 'w': f(it.w), 'h': f(it.h)}
                  for it in lay.items.values()},
        'edges': {e.id: {'lines': e.lines, 'lw': f(e.lw), 'lh': f(e.lh)}
                  for e in lay.edges},
    }


def load_split(path):
    """Như ft.load, nhưng giữ riêng issue của tầng parse và của tầng validate.

    ft.load gộp hai danh sách lại, nên không phân biệt được cái nào đến từ đâu.
    Bản port cần phân biệt: module source phải chốt được bằng issue của riêng
    nó, trước khi có module validate nào tồn tại.
    """
    ext = os.path.splitext(path)[1].lower()
    if ext in ('.md', '.markdown', '.txt'):
        table = ft.read_markdown(path)
    elif ext in ('.csv', '.tsv'):
        table = ft.read_csv(path)
    elif ext in ('.xlsx', '.xlsm'):
        table = ft.read_xlsx(path)
    else:
        raise ft.FlowTableError(f'đuôi file "{ext}" không hỗ trợ')
    if not table.title:
        table.title = os.path.splitext(os.path.basename(path))[0]
    parse_issues = list(table.issues)
    table.issues = parse_issues + ft.validate(table.rows)
    return table, parse_issues


def stage_table(table, lay, parse_issues=()):
    """Bảng sau khi parse, trước khi diễn giải theo ngữ nghĩa sơ đồ.

    Chốt module source và model. Không phụ thuộc layout, nên nó là chặng duy
    nhất dùng được cả với bảng có lỗi.

    issues ở đây chỉ là issue của tầng parse. Issue của tầng validate nằm ở
    chặng issues.
    """
    return {
        'title': table.title,
        'source': table.source,
        'issues': [{'level': i.level, 'loc': i.loc, 'id': i.id, 'msg': i.msg} for i in parse_issues],
        'rows': [{'idx': r.idx, 'loc': r.loc, 'id': r.id, 'type': r.type,
                  'parent': r.parent, 'lines': r.lines, 'meta': r.meta}
                 for r in table.rows],
    }


def stage_issues(table, lay):
    """Chốt module schema và validate."""
    return [{'level': i.level, 'loc': i.loc, 'id': i.id, 'msg': i.msg} for i in table.issues]


def stage_place(table, lay):
    """Lưới lane, row, col trước khi có pixel. Chốt module layout/place."""
    return {
        'lanes': [l.id for l in lay.lanes],
        'topo': list(lay.topo_order),
        'items': {it.id: {'lane': it.lane, 'row': it.row, 'col': it.col, 'attach': it.attach}
                  for it in lay.items.values()},
    }


def seg_key(s):
    return (str(s.res), s.lo, s.hi, str(s.key), s.track if s.track is not None else -1)


def stage_route(table, lay):
    """Kiểu đi dây, cổng ra vào, và các đoạn dây kèm track.

    Chốt module layout/route và layout/tracks.
    """
    return {
        'edges': {e.id: {'case': e.case, 'exit': e.exit_side, 'entry': e.entry_side,
                         'exit_frac': pair(e.exit_frac), 'entry_frac': pair(e.entry_frac),
                         'back': e.back}
                  for e in lay.edges},
        'segs': [{'res': [str(x) for x in s.res], 'lo': s.lo, 'hi': s.hi,
                  'key': str(s.key), 'track': s.track,
                  'stubs': [[p, side] for p, side in s.stubs]}
                 for s in sorted(lay.segs, key=seg_key)],
    }


def stage_geometry(table, lay):
    """Toạ độ cuối cùng. Chốt module layout/geometry và layout/labels."""
    return {
        'pool': {'w': f(lay.pool_w), 'h': f(lay.pool_h)},
        'origin': pair(lay.origin),
        'lanes': [{'id': l.id, 'x': f(lay.lane_x[i]), 'w': f(lay.lane_w[i])}
                  for i, l in enumerate(lay.lanes)],
        'items': {it.id: {'x': f(it.x), 'y': f(it.y), 'w': f(it.w), 'h': f(it.h)}
                  for it in lay.items.values()},
        'edges': {e.id: {'points': [pair(p) for p in (e.pts or [])],
                         'label_t': f(e.label_t), 'label_off': pair(e.label_off),
                         'label': pair(e.label) if e.label else None}
                  for e in lay.edges},
    }


FNS = {'text': stage_text, 'table': stage_table, 'issues': stage_issues,
       'place': stage_place, 'route': stage_route, 'geometry': stage_geometry}


def stages(table, lay, want=STAGES, parse_issues=()):
    """lay là None khi bảng có lỗi; khi đó chỉ chặng table và issues có nghĩa."""
    out = {}
    for name in want:
        if lay is None and name not in ('table', 'issues'):
            continue
        if name == 'table':
            out[name] = stage_table(table, lay, parse_issues)
        else:
            out[name] = FNS[name](table, lay)
    return out


def render(path, want=STAGES):
    table, parse_issues = load_split(path)
    lay = None
    if not any(i.level == 'error' for i in table.issues):
        tm = ft.TextMeasure()
        lay = ft.Layout(table.rows, ft.Config(), tm).run()
    return stages(table, lay, want, parse_issues)


def encode(data):
    return json.dumps(data, ensure_ascii=False, indent=1, sort_keys=True) + '\n'


def main(argv):
    if not argv:
        print(__doc__)
        return 2
    want = STAGES
    if '--stage' in argv:
        i = argv.index('--stage')
        want = (argv[i + 1],)
        if want[0] not in FNS:
            print(f'ERROR   chặng "{want[0]}" không có; chọn trong {", ".join(STAGES)}')
            return 2
        argv = argv[:i] + argv[i + 2:]
    sys.stdout.write(encode(render(argv[0], want)))
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
