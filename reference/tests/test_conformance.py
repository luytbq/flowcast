import os
import subprocess
import sys
import unittest

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))


def run(*args):
    return subprocess.run([sys.executable, *args], cwd=ROOT, capture_output=True, text=True)


class ConformanceTest(unittest.TestCase):
    def test_golden_khop_voi_ban_tham_chieu(self):
        r = run('conformance/generate.py', '--check')
        self.assertEqual(r.returncode, 0, r.stdout + r.stderr)

    def test_moi_nhanh_thuat_toan_deu_duoc_cham(self):
        r = run('tools/coverage.py')
        self.assertEqual(r.returncode, 0, r.stdout + r.stderr)


sys.path.insert(0, os.path.join(ROOT, 'reference'))
import flowtable2drawio as ft  # noqa: E402
import dump as dmp  # noqa: E402

CASES = os.path.join(ROOT, 'conformance', 'cases')


def stages_of(case, **cfg_kw):
    table = ft.load(os.path.join(CASES, case))
    cfg = ft.Config()
    for k, v in cfg_kw.items():
        setattr(cfg, k, v)
    lay = None
    if not any(i.level == 'error' for i in table.issues):
        lay = ft.Layout(table.rows, cfg, ft.TextMeasure()).run()
    return dmp.stages(table, lay)


class DumpTest(unittest.TestCase):
    """Một chặng dump chỉ là cổng chặn nếu nó thật sự phản ứng với thay đổi.

    Chặng luôn ra cùng một giá trị thì không chốt được gì, và điều đó im lặng
    cho tới lúc bản port sai ở đúng module ấy mà vẫn qua cổng.
    """

    def test_moi_chang_deu_khac_nhau_giua_cac_case(self):
        seen = {s: set() for s in dmp.STAGES}
        for name in sorted(f for f in os.listdir(CASES) if f.endswith('.md')):
            for stage, data in stages_of(name).items():
                seen[stage].add(dmp.encode(data))
        for stage, values in seen.items():
            self.assertGreater(len(values), 1, f'chặng {stage} ra cùng một giá trị cho mọi case')

    def test_chang_text_va_geometry_phan_ung_voi_cau_hinh(self):
        base = stages_of('19-wrap-long.md')
        narrow = stages_of('19-wrap-long.md', task_max_w=140)
        for stage in ('text', 'geometry'):
            self.assertNotEqual(dmp.encode(base[stage]), dmp.encode(narrow[stage]),
                                f'chặng {stage} không đổi khi bề rộng hộp đổi')

    def test_chang_luoi_khong_phu_thuoc_pixel(self):
        """place và route sống trên lưới, nên tham số pixel không được chạm tới chúng."""
        base = stages_of('28-tracks-fan-in.md')
        wide = stages_of('28-tracks-fan-in.md', track_gap=30, min_gutter=60, min_channel=80)
        for stage in ('place', 'route'):
            self.assertEqual(dmp.encode(base[stage]), dmp.encode(wide[stage]),
                             f'chặng {stage} đổi theo tham số pixel, tức lưới đã lẫn pixel')
