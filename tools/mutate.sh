#!/bin/sh
# Kiểm xem bộ đối chiếu có thật sự bắt được lỗi hay không, bằng cách cố tình
# làm sai từng chỗ rồi xem test có đỏ không.
#
#     sh tools/mutate.sh
#
# Một bộ đối chiếu trông đồ sộ mà không bắt được đột biến nào thì không chốt gì,
# và điều đó im lặng cho tới lúc bản port sai thật. Chạy lại sau mỗi lần thêm
# module Go mới.
#
# Khôi phục bằng git chứ không bằng sed ngược: sed ngược hỏng khi chuỗi thay thế
# chứa chính chuỗi bị thay, và khi chuỗi đích xuất hiện ở nhiều chỗ.
set -e
cd "$(dirname "$0")/.."

if ! git diff --quiet || ! git diff --cached --quiet; then
	echo 'ERROR cây làm việc đang bẩn; commit hoặc stash trước khi chạy'
	exit 1
fi

caught=0
missed=0

mutate() {
	label="$1" file="$2" from="$3" to="$4"
	python3 - "$file" "$from" "$to" <<'PY'
import io, sys
p, a, b = sys.argv[1:4]
s = io.open(p, encoding='utf-8').read()
if a not in s:
    sys.exit(f'ERROR không tìm thấy trong {p}: {a!r}')
io.open(p, 'w', encoding='utf-8').write(s.replace(a, b, 1))
PY
	if go test -count=1 ./... >/dev/null 2>&1; then
		echo "BỎ LỌT     $label"
		missed=$((missed + 1))
	else
		echo "BẮT ĐƯỢC   $label"
		caught=$((caught + 1))
	fi
	git checkout -- "$file"
}

mutate "bỏ ký tự - khỏi chỗ được ngắt" text/measure.go '=&?-"' '=&?"'
mutate "bỏ nhánh :: khỏi chỗ được ngắt" text/measure.go "return i >= 2 && r[i-1] == ':' && r[i-2] == ':'" 'return false'
mutate "codepoint lạ đo bằng 0 thay vì notdef" text/metrics.go 'total += m.Notdef' 'total += 0'
mutate "bỏ thu hẹp nhị phân trong Wrap" text/measure.go 'out = append(out, t.wrapLine(line, float64(hi))...)' 'out = append(out, first...)'
mutate "không bật cờ hard khi cắt cứng" text/measure.go 't.hard = true' '_ = 0'
mutate "ngân sách ngắt dòng của task lệch 2px" layout/size.go 'cfg.TaskMaxW-32' 'cfg.TaskMaxW-30'
mutate "ngân sách ngắt dòng của hình thoi lệch 10px" layout/size.go 'float64(cfg.CondWrap))' 'float64(cfg.CondWrap)+10)'
mutate "Fmt làm tròn 3 chữ số thay vì 2" num/num.go "'f', 2, 64" "'f', 3, 64"
mutate "Rnd làm tròn xuống thay vì lên" num/num.go 'math.Ceil(v/2.0)' 'math.Floor(v/2.0)'

echo
echo "bắt được $caught, bỏ lọt $missed"
[ "$missed" -eq 0 ]
