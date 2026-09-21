#!/bin/sh
# Toàn bộ kiểm tra của repo, dùng cho CI và trước khi commit.
set -e
cd "$(dirname "$0")/.."
echo '--- test bản tham chiếu Python'
( cd reference && python3 -m unittest discover -s tests )
echo '--- độ phủ bộ đối chiếu'
python3 tools/coverage.py
echo '--- test bản Go'
gofmt -l . | grep . && { echo 'ERROR file chưa gofmt ở trên'; exit 1; } || true
go vet ./...
go test ./...
