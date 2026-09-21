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

# schema và validate
mutate "đảo thứ tự kiểm from và to" schema/schema.go '{Key: "from", Required: true, Note: edgeNote},
			{Key: "to", Required: true, Note: edgeNote},' '{Key: "to", Required: true, Note: edgeNote},
			{Key: "from", Required: true, Note: edgeNote},'
mutate "attach của text thành bắt buộc" schema/schema.go '{Key: "attach", Note: attachNote, SameLane: true}' '{Key: "attach", Required: true, Note: attachNote, SameLane: true}'
mutate "edge không nhận style dashed" schema/schema.go 'StyleValues: []string{"highlight", "dashed"}' 'StyleValues: []string{"highlight"}'
mutate "id trùng thì dòng sau đè dòng trước" validate/validate.go 'if j, dup := v.byID[r.ID]; dup {' 'if j, dup := v.byID[r.ID]; false {'
mutate "dòng đánh dấu phần còn lại cũng cần parent" validate/validate.go 'case !isMarker(r):' 'case true:'
mutate "styleList không bỏ giá trị trùng" validate/validate.go 'if !dup {' 'if true {'
mutate "vị trí lấy theo dòng đầu thay vì dòng cuối" validate/validate.go 'if r.ID != "" {
			pos[r.ID] = i
		}' 'if _, seen := pos[r.ID]; r.ID != "" && !seen {
			pos[r.ID] = i
		}'
mutate "không bỏ qua db và text khi xét cạnh liền sau" validate/validate.go 'for p < len(v.rows) && schema.AttachTypes[v.rows[p].Type] && v.rows[p].Meta["attach"] == r.ID {' 'for false {'
mutate "condition chỉ cần một cạnh ra" validate/validate.go 'r.Type == "condition" && len(es) < 2' 'r.Type == "condition" && len(es) < 1'
mutate "không cảnh báo cạnh condition thiếu nhãn" validate/validate.go 'if e.Text() == "" {' 'if false {'
mutate "quên bỏ qua dòng trùng id khi xét đồ thị" validate/validate.go 'if j, ok := v.byID[r.ID]; !ok || j != i {' 'if j, ok := v.byID[r.ID]; false || j == -1 && !ok {'
mutate "thông điệp dùng %q thay vì nháy thẳng" validate/validate.go 'fmt.Sprintf("metadata \"%s\" không dùng cho type %s, bị bỏ qua", k, r.Type)' 'fmt.Sprintf("metadata %q không dùng cho type %s, bị bỏ qua", k, r.Type)'

echo
echo "bắt được $caught, bỏ lọt $missed"
[ "$missed" -eq 0 ]
