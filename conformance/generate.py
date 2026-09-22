"""Sinh và kiểm bộ đáp án đối chiếu từ bản tham chiếu Python.

    python3 conformance/generate.py            # sinh lại golden
    python3 conformance/generate.py --check    # so golden hiện có, khác thì báo lỗi

Mỗi case trong cases/ cho ra hai file trong golden/:

    <case>.drawio      xml đầy đủ, so từng byte
    <case>.report.txt  issue, cảnh báo layout, phát hiện tự kiểm, số liệu

Case có lỗi bảng thì không có file .drawio, chỉ có report. Đó là cố ý: chúng
chốt hành vi kiểm tra đầu vào, thứ bản port cũng phải tái tạo đúng.

Bản port đọc đúng bộ này và phải khớp từng byte. Vì vậy mọi thứ ở đây phải tất
định: số đo chữ lấy từ data/verdana.json chứ không dò font trên máy, và report
không chứa đường dẫn, thời gian hay thứ tự phụ thuộc bảng băm.
"""
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
sys.path.insert(0, os.path.join(ROOT, 'reference'))

import flowtable2drawio as ft  # noqa: E402
import dump as dmp  # noqa: E402

CASES = os.path.join(HERE, 'cases')
GOLDEN = os.path.join(HERE, 'golden')
DUMPS = os.path.join(HERE, 'dumps')


EXTS = ('.md', '.csv', '.xlsx')


def render(path):
    """Trả về (xml hoặc None, report, dump). xml là None khi bảng có lỗi.

    Lỗi chặn ở tầng đọc, như không giải mã được hay không có header, không có
    bảng nào để dump; report chỉ có dòng lỗi, và dump chỉ ghi thông điệp. Đường
    dẫn trong thông điệp rút về tên file, để golden không phụ thuộc máy.
    """
    out = []
    try:
        table, parse_issues = dmp.load_split(path)
    except ft.FlowTableError as ex:
        msg = str(ex).replace(path, os.path.basename(path))
        return None, f'ERROR   {msg}\n', dmp.encode({'fatal': msg})
    for issue in sorted(table.issues, key=lambda i: i.level != 'error'):
        out.append(str(issue))
    nerr = sum(1 for i in table.issues if i.level == 'error')
    out.append(f'check: {nerr} lỗi, {len(table.issues) - nerr} cảnh báo')
    if nerr:
        out.append('build: dừng vì bảng có lỗi')
        return None, '\n'.join(out) + '\n', dmp.encode(dmp.stages(table, None, parse_issues=parse_issues))

    tm = ft.TextMeasure()
    if not isinstance(tm.font, ft.TableMetrics):
        raise SystemExit('ERROR bảng số đo data/verdana.json không đọc được; golden sẽ không tất định')
    lay = ft.Layout(table.rows, ft.Config(), tm).run()
    findings = lay.check()
    xml = ft.to_drawio(lay, table.title)

    out.append(f'build: {len(lay.lanes)} lane, {len(lay.items)} phần tử, {len(lay.edges)} cạnh, '
               f'{ft.fmt(lay.pool_w)}x{ft.fmt(lay.pool_h)}px')
    for w in lay.warnings:
        out.append(f'WARNING layout: {w}')
    for level, msg in findings:
        out.append(f'{level.upper():7} layout: {msg}')
    nerr = sum(1 for l, _ in findings if l == 'error')
    out.append(f'layout: {nerr} lỗi, {len(findings) - nerr} cảnh báo')
    return xml, '\n'.join(out) + '\n', dmp.encode(dmp.stages(table, lay, parse_issues=parse_issues))


def cases():
    return sorted(f for f in os.listdir(CASES) if f.endswith(EXTS))


def read(path):
    if not os.path.exists(path):
        return None
    with open(path, encoding='utf-8') as f:
        return f.read()


def write(path, text):
    if text is None:
        if os.path.exists(path):
            os.remove(path)
        return
    with open(path, 'w', encoding='utf-8') as f:
        f.write(text)


def main(argv):
    check = '--check' in argv
    os.makedirs(GOLDEN, exist_ok=True)
    os.makedirs(DUMPS, exist_ok=True)
    bad = []
    for name in cases():
        stem = os.path.splitext(name)[0]
        xml, report, dump_json = render(os.path.join(CASES, name))
        want = {
            os.path.join(GOLDEN, f'{stem}.drawio'): xml,
            os.path.join(GOLDEN, f'{stem}.report.txt'): report,
            os.path.join(DUMPS, f'{stem}.json'): dump_json,
        }
        for path, text in want.items():
            if check:
                if read(path) != text:
                    bad.append(os.path.relpath(path, HERE))
            else:
                write(path, text)
    if check:
        for f in bad:
            print(f'LỆCH  {f}')
        print(f'đối chiếu: {len(cases())} case, {len(bad)} lệch')
        return 1 if bad else 0
    print(f'đã sinh golden cho {len(cases())} case')
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
