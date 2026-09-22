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
# Có drawio thì kiểm luôn rằng draw.io thật vẽ dây đúng toạ độ đã tính.
if command -v drawio >/dev/null 2>&1 || command -v draw.io >/dev/null 2>&1; then
	FLOWCAST_DRAWIO_TEST=1 go test ./...
else
	echo 'không có drawio CLI: bỏ qua test render với drawio thật'
	go test ./...
fi
