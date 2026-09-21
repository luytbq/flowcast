#!/bin/sh
# Toàn bộ kiểm tra của repo, dùng cho CI và trước khi commit.
set -e
cd "$(dirname "$0")/.."
echo '--- test bản tham chiếu'
( cd reference && python3 -m unittest discover -s tests )
echo '--- độ phủ bộ đối chiếu'
python3 tools/coverage.py
