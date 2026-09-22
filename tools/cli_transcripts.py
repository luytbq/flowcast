"""Sinh bản ghi phiên làm việc của CLI bản tham chiếu, làm đáp án cho CLI Go.

    python3 tools/cli_transcripts.py conformance/cli

Subagent flowtable-drawio đọc mã thoát và chép nguyên văn các dòng tool in ra,
nên CLI Go phải in ra đúng những dòng đó thì mới thay được bản Python.

Mỗi kịch bản chạy trong một thư mục tạm mới, với file đầu vào chép vào đó. Bản ghi
gồm từng lệnh, những gì lệnh in ra, mã thoát, và cuối cùng là danh sách file trong
thư mục kèm sha256 nội dung. Đường dẫn thư mục tạm được thay bằng $T.

Không có ở đây, vì thuộc các bước sau của lộ trình port hoặc cố ý khác: merge,
--png, --verify, --font, csv và xlsx, và --layout-json, file mà CLI Go ghi số theo
cách của Go.
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
]


def run(cmd, tmp):
    argv = [a.replace('$T', tmp) for a in cmd]
    p = subprocess.run([sys.executable, TOOL] + argv, capture_output=True, text=True,
                       stdin=subprocess.DEVNULL, cwd=tmp)
    if p.stderr.strip():
        raise SystemExit(f'ERROR bản tham chiếu ghi ra stderr với {cmd}: {p.stderr}')
    return p.stdout.replace(tmp, '$T'), p.returncode


def transcript(case, fname, cmds):
    tmp = tempfile.mkdtemp()
    try:
        tmp = os.path.realpath(tmp)
        shutil.copy(os.path.join(CASES, case + '.md'), os.path.join(tmp, fname))
        # Lệnh ghi bằng mảng JSON, vì đối số có thể chứa khoảng trắng.
        out = [f'# input: {case} -> {fname}\n']
        for cmd in cmds:
            text, code = run(cmd, tmp)
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
        if not f.endswith('.md'):
            continue
        case = f[:-3]
        text = transcript(case, 'in.md', [['check', '$T/in.md'], ['build', '$T/in.md']])
        io.open(os.path.join(dst, f'case-{case}.txt'), 'w', encoding='utf-8').write(text)
        n += 1
    for name, case, fname, cmds in SPECIAL:
        io.open(os.path.join(dst, f'{name}.txt'), 'w', encoding='utf-8').write(transcript(case, fname, cmds))
        n += 1
    print(f'{dst}: {n} bản ghi')
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
