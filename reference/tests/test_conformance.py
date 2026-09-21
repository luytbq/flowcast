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
