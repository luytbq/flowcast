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
n = s.count(a)
# Đúng một chỗ, không hơn không kém. Thay chỗ đầu khi có nhiều chỗ sẽ để lại
# bản sao còn nguyên và đột biến thành vô hại, khiến cổng trông như bỏ lọt.
if n != 1:
    sys.exit(f'ERROR {p}: tìm thấy {n} chỗ khớp {a!r}, cần đúng 1')
io.open(p, 'w', encoding='utf-8').write(s.replace(a, b))
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
mutate "ngân sách ngắt dòng của task lệch 2px" layout/size.go 'cfg.TaskMaxW - 32' 'cfg.TaskMaxW - 30'
mutate "ngân sách của loại lạ dùng DBWrap" layout/size.go 'return float64(cfg.TextWrap)' 'return float64(cfg.DBWrap)'
mutate "ngân sách ngắt dòng của hình thoi lệch 10px" layout/size.go 'return float64(cfg.CondWrap)' 'return float64(cfg.CondWrap) + 10'
mutate "Fmt làm tròn 3 chữ số thay vì 2" num/num.go "'f', 2, 64" "'f', 3, 64"
mutate "Rnd làm tròn xuống thay vì lên" num/num.go 'math.Ceil(v/2.0)' 'math.Floor(v/2.0)'

# source/markdown
mutate "bỏ chuẩn hóa NFC" source/text.go 'return norm.NFC.String(b.String())' 'return b.String()'
mutate "gạch đứng có escape vẫn ngăn cột" source/text.go "if c == '|' && !prevEscape {" "if c == '|' {"
mutate "không gỡ escape markdown" source/text.go 'if r[i] == 0x5c && i+1 < len(r) && strings.ContainsRune(mdEscapable, r[i+1]) {' 'if false {'
mutate "chỉ cắt dòng theo \\n, bỏ CRLF" source/text.go "case 0x0d:" "case 0x2400:"
mutate "không hạ chữ thường cột type" source/markdown.go 'Type:   strings.ToLower(cells[1]),' 'Type:   cells[1],'
mutate "Idx dùng số dòng thay vì thứ tự đọc được" source/markdown.go 'Idx:    len(rows),' 'Idx:    i,'
mutate "khóa metadata cũng bị gỡ escape" source/markdown.go 'meta[strings.TrimSpace(k)] = unesc(strings.TrimSpace(v))' 'meta[unesc(strings.TrimSpace(k))] = unesc(strings.TrimSpace(v))'
mutate "thẻ br phân biệt hoa thường" source/text.go 'low := strings.ToLower(s)' 'low := s'
mutate "heading cấp hai cũng tính là tiêu đề" source/markdown.go 'if len(trimmed) == len(rest) {' 'if false {'

echo
echo "bắt được $caught, bỏ lọt $missed"
[ "$missed" -eq 0 ]
