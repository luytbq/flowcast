"""Sinh bản ghi phiên làm việc của CLI bản tham chiếu, làm đáp án cho CLI Go.

    python3 tools/cli_transcripts.py conformance/cli

Subagent flowtable-drawio đọc mã thoát và chép nguyên văn các dòng tool in ra,
nên CLI Go phải in ra đúng những dòng đó thì mới thay được bản Python.

Mỗi kịch bản chạy trong một thư mục tạm mới, với file đầu vào chép vào đó. Bản ghi
gồm từng lệnh, những gì lệnh in ra, mã thoát, và cuối cùng là danh sách file trong
thư mục kèm sha256 nội dung. Đường dẫn thư mục tạm được thay bằng $T.

Đầu vào "merge/<kịch bản>" là cả thư mục conformance/merge/<kịch bản>: bảng đã
sửa cùng file .drawio cũ đã sửa tay.

Bản ghi có --png hoặc --verify chạy với một drawio giả trong
conformance/drawio-fakes, ghi ở dòng "# drawio: <tên>", vì drawio thật cho ra
ảnh khác nhau giữa các phiên bản và không có trên máy CI.

Không có ở đây vì cố ý khác: --font, và --layout-json, file mà CLI Go ghi số
theo cách của Go.
"""
import hashlib
import io
import json
import os
import shutil
import subprocess
import sys
import tempfile

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
TOOL = os.path.join(ROOT, 'reference', 'flowtable2drawio.py')
CASES = os.path.join(ROOT, 'conformance', 'cases')
MERGE = os.path.join(ROOT, 'conformance', 'merge')
FAKES = os.path.join(ROOT, 'conformance', 'drawio-fakes')

# Kịch bản đặc biệt: (tên, case đầu vào, tên file đầu vào, các lệnh).
SPECIAL = [
    ('exists-no-mode', '03-condition-two', 'in.md', [['build', '$T/in.md'], ['build', '$T/in.md']]),
    ('force-backup', '03-condition-two', 'in.md', [['build', '$T/in.md'], ['build', '$T/in.md', '--mode', 'force']]),
    ('force-no-backup', '03-condition-two', 'in.md',
     [['build', '$T/in.md'], ['build', '$T/in.md', '--mode', 'force', '--no-backup']]),
    ('output-and-title', '05-merge-node', 'in.md',
     [['build', '$T/in.md', '-o', '$T/ra.drawio', '--title', 'Tiêu đề khác']]),
    ('equals-syntax', '28-tracks-fan-in', 'in.md',
     [['build', '$T/in.md', '--output=$T/ra.drawio', '--task-max-w=200', '--title=Có dấu bằng']]),
    ('flags-before-file', '12-text-attach', 'in.md', [['build', '--attach-gap', '60', '$T/in.md']]),
    ('config-flags', '19-wrap-long', 'in.md',
     [['build', '$T/in.md', '--task-max-w', '160', '--text-wrap', '120', '--db-wrap', '90',
       '--track-gap', '20', '--min-gutter', '40', '--min-channel', '50']]),
    ('config-tracks', '28-tracks-fan-in', 'in.md',
     [['build', '$T/in.md', '--track-gap', '30', '--gutter-margin', '5', '--channel-margin', '20']]),
    ('missing-file', '03-condition-two', 'in.md', [['check', '$T/khong-co.md'], ['build', '$T/khong-co.md']]),
    ('unknown-extension', '03-condition-two', 'in.dat', [['check', '$T/in.dat']]),
    ('txt-is-markdown', '03-condition-two', 'in.txt', [['build', '$T/in.txt']]),
    ('default-output-strips-extension', '03-condition-two', 'so.do.md', [['build', '$T/so.do.md']]),
    # Dấu chấm ở đầu tên file không phải dấu tách đuôi, nên ".md" không có đuôi.
    ('dotfile-has-no-extension', '03-condition-two', '.md', [['check', '$T/.md']]),
    # Cấu hình người dùng gõ được mà làm các track trùng nhau: nhánh duy nhất đi
    # tới mã thoát 2, vì với cấu hình mặc định không bảng nào có lỗi hình học.
    ('track-gap-zero', '28-tracks-fan-in', 'in.md', [['build', '$T/in.md', '--track-gap', '0']]),
    ('csv-delimiter-explicit', 'csv-02-semicolon', 'in.csv', [['build', '$T/in.csv', '--delimiter', ';']]),
    ('csv-delimiter-wrong', 'csv-01-comma', 'in.csv', [['check', '$T/in.csv', '--delimiter', ';']]),
    ('csv-encoding-alias', 'csv-01-comma', 'in.csv', [['check', '$T/in.csv', '--encoding', 'UTF8']]),
    # Buộc đọc một file utf-8 theo cp1252: chữ hỏng nhưng vẫn dựng được.
    ('csv-encoding-forced-cp1252', 'csv-01-comma', 'in.csv', [['build', '$T/in.csv', '--encoding', 'cp1252']]),
    # Tên viết hoa đọc được mà không cảnh báo, vì bản tham chiếu so nguyên văn.
    ('csv-encoding-uppercase', 'csv-05-cp1252', 'in.csv', [['check', '$T/in.csv', '--encoding', 'CP1252']]),
    ('csv-encoding-unknown', 'csv-01-comma', 'in.csv', [['check', '$T/in.csv', '--encoding', 'klingon']]),
    ('xlsx-sheet-explicit', 'xlsx-08-two-flows', 'in.xlsx', [['build', '$T/in.xlsx', '--sheet', 'Hai']]),
    ('xlsx-sheet-missing', 'xlsx-08-two-flows', 'in.xlsx', [['check', '$T/in.xlsx', '--sheet', 'Không có']]),
    # --mode merge khi chưa có file đích thì chỉ là tạo mới.
    ('merge-without-old-file', '03-condition-two', 'in.md', [['build', '$T/in.md', '--mode', 'merge']]),
    ('merge-twice', '03-condition-two', 'in.md',
     [['build', '$T/in.md'], ['build', '$T/in.md', '--mode', 'merge'], ['build', '$T/in.md', '--mode', 'merge']]),
    ('merge-no-backup', 'merge/moved-node', '*', [['build', '$T/flow.md', '--mode', 'merge', '--no-backup']]),
    ('merge-output-title', 'merge/moved-node', '*',
     [['build', '$T/flow.md', '--mode', 'merge', '-o', '$T/flow.drawio', '--title', 'Tiêu đề mới']]),
    ('merge-config-flags', 'merge/new-node', '*',
     [['build', '$T/flow.md', '--mode', 'merge', '--min-channel', '50', '--task-max-w', '160']]),
    ('merge-then-force', 'merge/freehand-note', '*',
     [['build', '$T/flow.md', '--mode', 'merge'], ['build', '$T/flow.md', '--mode', 'force']]),
]


# Kịch bản xuất ảnh: (tên, case đầu vào, tên file đầu vào, các lệnh, drawio giả).
RENDER = [
    ('render-no-drawio', '03-condition-two', 'in.md', [['build', '$T/in.md', '--png', '--verify']], 'none'),
    ('render-png-default', '03-condition-two', 'in.md', [['build', '$T/in.md', '--png']], 'ok'),
    ('render-png-path', '03-condition-two', 'in.md', [['build', '$T/in.md', '--png', '$T/anh.png']], 'ok'),
    ('render-png-equals', '03-condition-two', 'so.do.md', [['build', '$T/so.do.md', '--png=$T/b.png']], 'ok'),
    ('render-verify', '05-merge-node', 'in.md', [['build', '$T/in.md', '--verify']], 'ok'),
    ('render-png-and-verify', '05-merge-node', 'in.md', [['build', '$T/in.md', '--verify', '--png']], 'ok'),
    ('render-fail', '03-condition-two', 'in.md', [['build', '$T/in.md', '--png']], 'fail'),
    ('render-verify-fail', '03-condition-two', 'in.md', [['build', '$T/in.md', '--verify']], 'fail'),
    ('render-silent', '03-condition-two', 'in.md', [['build', '$T/in.md', '--png', '--verify']], 'silent'),
    # Mã 2 của tự kiểm được giữ, không bị kiểm render đổi thành 3.
    ('render-keeps-layout-code', '28-tracks-fan-in', 'in.md',
     [['build', '$T/in.md', '--track-gap', '0', '--verify']], 'ok'),
    ('render-fail-keeps-layout-code', '28-tracks-fan-in', 'in.md',
     [['build', '$T/in.md', '--track-gap', '0', '--png']], 'fail'),
    ('render-merge-skips-verify', 'merge/moved-node', '*',
     [['build', '$T/flow.md', '--mode', 'merge', '--verify', '--png']], 'ok'),
    ('render-merge-no-drawio', 'merge/moved-node', '*', [['build', '$T/flow.md', '--mode', 'merge', '--verify']],
     'none'),
    ('render-check-ignores-flags', '03-condition-two', 'in.md', [['check', '$T/in.md']], 'fail'),
]


def run(cmd, tmp, fake=None):
    argv = [a.replace('$T', tmp) for a in cmd]
    env = None
    if fake:
        env = dict(os.environ, PATH=os.path.join(FAKES, fake) + ':/usr/bin:/bin')
    p = subprocess.run([sys.executable, TOOL] + argv, capture_output=True, text=True,
                       stdin=subprocess.DEVNULL, cwd=tmp, env=env)
    if p.stderr.strip():
        raise SystemExit(f'ERROR bản tham chiếu ghi ra stderr với {cmd}: {p.stderr}')
    return p.stdout.replace(tmp, '$T'), p.returncode


def case_file(case):
    for ext in ('.md', '.csv', '.xlsx'):
        p = os.path.join(CASES, case + ext)
        if os.path.exists(p):
            return p
    raise SystemExit(f'ERROR không có case {case}')


def transcript(case, fname, cmds, fake=None):
    tmp = tempfile.mkdtemp()
    try:
        tmp = os.path.realpath(tmp)
        if case.startswith('merge/'):
            src = os.path.join(MERGE, case[len('merge/'):])
            for n in sorted(os.listdir(src)):
                shutil.copy(os.path.join(src, n), os.path.join(tmp, n))
        else:
            shutil.copy(case_file(case), os.path.join(tmp, fname))
        # Lệnh ghi bằng mảng JSON, vì đối số có thể chứa khoảng trắng.
        out = [f'# input: {case} -> {fname}\n']
        if fake:
            out.append(f'# drawio: {fake}\n')
        for cmd in cmds:
            text, code = run(cmd, tmp, fake)
            out.append('$ ' + json.dumps(cmd, ensure_ascii=False) + '\n' + text + f'[exit {code}]\n')
        out.append('--- files\n')
        for n in sorted(os.listdir(tmp)):
            data = open(os.path.join(tmp, n), 'rb').read()
            out.append(f'{hashlib.sha256(data).hexdigest()[:16]}  {n}\n')
        return ''.join(out)
    finally:
        shutil.rmtree(tmp)


def main(argv):
    if len(argv) != 1:
        print(__doc__)
        return 2
    dst = argv[0]
    if os.path.isdir(dst):
        shutil.rmtree(dst)
    os.makedirs(dst)
    n = 0
    for f in sorted(os.listdir(CASES)):
        case, ext = os.path.splitext(f)
        if ext not in ('.md', '.csv', '.xlsx'):
            continue
        fn = 'in' + ext
        text = transcript(case, fn, [['check', '$T/' + fn], ['build', '$T/' + fn]])
        io.open(os.path.join(dst, f'case-{case}.txt'), 'w', encoding='utf-8').write(text)
        n += 1
    for sc in sorted(os.listdir(MERGE)):
        text = transcript('merge/' + sc, '*', [['build', '$T/flow.md', '--mode', 'merge']])
        io.open(os.path.join(dst, f'merge-{sc}.txt'), 'w', encoding='utf-8').write(text)
        n += 1
    for name, case, fname, cmds in SPECIAL:
        io.open(os.path.join(dst, f'{name}.txt'), 'w', encoding='utf-8').write(transcript(case, fname, cmds))
        n += 1
    for name, case, fname, cmds, fake in RENDER:
        io.open(os.path.join(dst, f'{name}.txt'), 'w', encoding='utf-8').write(transcript(case, fname, cmds, fake))
        n += 1
    print(f'{dst}: {n} bản ghi')
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
