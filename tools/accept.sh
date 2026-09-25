#!/bin/sh
# End-to-end acceptance check: build every case in the conformance suite, then
# have the real drawio render it and compare each wire with the computed coordinates.
#
#     sh tools/accept.sh              # directions TD and LR
#     DIRS="TD BT LR RL" sh tools/accept.sh
#
# Unlike tools/check.sh, it runs the real drawio on every case, so it takes a few
# minutes; check.sh runs drawio on only a few cases. This is the only check that
# answers the ultimate question: when draw.io opens this file, does it draw what
# the tool computed?
set -e
cd "$(dirname "$0")/.."
command -v drawio >/dev/null 2>&1 || command -v draw.io >/dev/null 2>&1 || {
	echo 'drawio CLI not found'
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
	# Cases with invalid input stop at the check stage; there is nothing to draw.
	case "$name" in 9*|csv-9*|xlsx-9*) continue ;; esac
	for d in ${DIRS:-TD LR}; do
		n=$((n + 1))
		log=$("$bin" build "$PWD/$f" -o "$out/$(basename "$(dirname "$f")")-$name-$d.drawio" \
			--direction "$d" --verify 2>&1) || true
		echo "$log" | grep -q '^render: 0 mismatches' || {
			echo "MISMATCH $f $d"
			echo "$log" | tail -3
			bad=$((bad + 1))
		}
	done
done
rm -rf "$out"
echo "$n builds, $bad with wires drawn differently from the computed coordinates"
[ "$bad" -eq 0 ]
