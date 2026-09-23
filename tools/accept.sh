#!/bin/sh
# Nghiệm thu đầu cuối: dựng mọi case trong bộ đối chiếu rồi bắt drawio thật vẽ
# và so từng đường dây với toạ độ đã tính.
#
#     sh tools/accept.sh              # hướng TD và LR
#     DIRS="TD BT LR RL" sh tools/accept.sh
#
# Khác tools/check.sh ở chỗ nó chạy drawio thật trên toàn bộ case, nên mất vài
# phút; check.sh chỉ chạy drawio trên vài case. Đây là phép kiểm duy nhất trả
# lời được câu hỏi cuối cùng: draw.io mở file này ra có vẽ đúng thứ tool tính
# hay không.
set -e
cd "$(dirname "$0")/.."
command -v drawio >/dev/null 2>&1 || command -v draw.io >/dev/null 2>&1 || {
	echo 'không có drawio CLI'
	exit 1
}
go build -o "$(mktemp -d)/flowcast" ./cmd/flowcast 2>/dev/null || true
bin=$(mktemp -d)/flowcast
go build -o "$bin" ./cmd/flowcast
out=$(mktemp -d)
bad=0
n=0
for f in conformance/cases/*.md conformance/cases/*.csv conformance/cases/*.xlsx \
	conformance/flowchart/*.md conformance/mermaid/*.mmd; do
	name=$(basename "$f")
	# Case sai đầu vào dừng ở tầng kiểm tra, không có gì để vẽ.
	case "$name" in 9*|csv-9*|xlsx-9*) continue ;; esac
	for d in ${DIRS:-TD LR}; do
		n=$((n + 1))
		log=$("$bin" build "$PWD/$f" -o "$out/$(basename "$(dirname "$f")")-$name-$d.drawio" \
			--direction "$d" --verify 2>&1) || true
		echo "$log" | grep -q '^render: 0 điểm lệch' || {
			echo "LỆCH $f $d"
			echo "$log" | tail -3
			bad=$((bad + 1))
		}
	done
done
rm -rf "$out"
echo "$n lần dựng, $bad lần dây vẽ khác toạ độ tính"
[ "$bad" -eq 0 ]
