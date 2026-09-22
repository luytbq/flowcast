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
# Chạy hai lượt. Lượt đầu chỉ kiểm mọi chuỗi đích còn khớp đúng một chỗ, để một
# đột biến cũ sau khi sửa code làm cả bộ dừng ngay từ đầu, thay vì dừng giữa
# chừng sau khi đã tốn thời gian chạy test.
#
# Chuỗi đích nên tránh khoảng trắng canh lề: gofmt canh lại cột mỗi khi một
# struct có thêm trường dài hơn, và đột biến sẽ không còn khớp.
#
# Khôi phục bằng git chứ không bằng sed ngược: sed ngược hỏng khi chuỗi thay thế
# chứa chính chuỗi bị thay, và khi chuỗi đích xuất hiện ở nhiều chỗ.
set -e
cd "$(dirname "$0")/.."

if [ -z "$PREFLIGHT" ]; then
	if ! git diff --quiet || ! git diff --cached --quiet; then
		echo 'ERROR cây làm việc đang bẩn; commit hoặc stash trước khi chạy'
		exit 1
	fi
	PREFLIGHT=1 sh "$0" || exit 1
	echo 'mọi chuỗi đích còn khớp; bắt đầu đột biến'
	echo
fi

caught=0
missed=0

mutate() {
	label="$1" file="$2" from="$3" to="$4"
	python3 - "$file" "$from" "$to" "${PREFLIGHT:-}" <<'EOPY'
import io, sys
p, a, b, preflight = sys.argv[1:5]
s = io.open(p, encoding='utf-8').read()
n = s.count(a)
# Đúng một chỗ, không hơn không kém. Thay chỗ đầu khi có nhiều chỗ sẽ để lại
# bản sao còn nguyên và đột biến thành vô hại, khiến cổng trông như bỏ lọt.
if n != 1:
    sys.exit(f'ERROR {p}: tìm thấy {n} chỗ khớp {a!r}, cần đúng 1')
if not preflight:
    io.open(p, 'w', encoding='utf-8').write(s.replace(a, b))
EOPY
	[ -n "$PREFLIGHT" ] && return 0
	if go test -count=1 ./... >/dev/null 2>&1; then
		echo "BỎ LỌT     $label"
		missed=$((missed + 1))
	else
		echo "BẮT ĐƯỢC   $label"
		caught=$((caught + 1))
	fi
	git checkout -- "$file"
}

# Đột biến tương đương đã chứng minh, không thêm lại vì chúng luôn bỏ lọt:
#
# - Đảo hướng branchOffset. Nó chỉ dùng để đếm cạnh có offset dương, mà offset
#   là hoán vị của 0 tới n-1, nên số đếm luôn là n-1. Đã bỏ hàm đó.
# - Bỏ phép kiểm mặt đón của đích trong kiểu B. Lúc pha B chạy, mặt trái hoặc
#   phải của đích chỉ có thể bị chiếm bởi một cạnh B trước đó nối đích với một
#   node x. x cùng phía với nguồn trên cùng hàng thì hoặc x nằm giữa, chặn
#   đường của nguồn, hoặc nguồn nằm giữa, khiến cạnh của x không thể là B. x
#   trùng nguồn thì phép kiểm phía nguồn đã bắt trước.
# - Bỏ lối tắt một cạnh mỗi mặt trong assignPorts. Với đúng một cạnh, mọi nhánh
#   đều cho ra giữa mặt.
#
# Hai cái sau sống sót qua 78.120 bảng sinh ngẫu nhiên trước khi được chứng
# minh. Suy luận mà không có số liệu thì không đáng tin: hai nhánh khác cũng
# từng bị nghi là thừa rồi bị bảng ngẫu nhiên giết.

# num, text, layout/size
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
mutate "không coi CR là ranh giới dòng" source/text.go "case 0x0d:" "case 0x2400:"
mutate "không hạ chữ thường cột type" source/markdown.go 'strings.ToLower(cells[1])' 'cells[1]'
mutate "Idx dùng số dòng thay vì thứ tự đọc được" source/markdown.go 'len(rows),' 'i,'
mutate "khóa metadata cũng bị gỡ escape" source/markdown.go 'key := strings.TrimSpace(k)' 'key := unesc(strings.TrimSpace(k))'
mutate "thẻ br phân biệt hoa thường" source/text.go 'low := strings.ToLower(s)' 'low := s'
mutate "heading cấp hai cũng tính là tiêu đề" source/markdown.go 'if len(trimmed) == len(rest) {' 'if false {'
mutate "thứ tự key metadata theo map thay vì theo ô" source/markdown.go 'if _, seen := meta[key]; !seen {' 'if _, seen := meta[key]; seen {'

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

# Phép kiểm tĩnh FMA
mutate "gỡ bọc float64() khỏi phép nhân trước phép cộng" layout/size.go 'float64(tw*1.42)+24+pad' 'tw*1.42+24+pad'

# layout/place
mutate "topo ưu tiên dòng đứng sau thay vì đứng trước" layout/topo.go 'return h[i].Order < h[j].Order' 'return h[i].Order > h[j].Order'
mutate "topo không bỏ qua cạnh back" layout/topo.go 'if _, ok := indeg[e.Dst]; ok && !e.Back {' 'if _, ok := indeg[e.Dst]; ok {'
mutate "branchDrift đảo hướng" layout/branch.go 'if w.Lane < v.Lane {' 'if w.Lane > v.Lane {'
mutate "branchDrift không dừng ở node hợp nhánh" layout/branch.go 'if l.nonBackIn(w.ID) > 1 {' 'if false {'
mutate "nhánh không rõ hướng ưu tiên trái" layout/branch.go 'if next[1] <= next[-1] {' 'if next[1] < next[-1] {'
mutate "bỏ qua mặt đã có mũi tên ngang" layout/branch.go "onlyLeft := busy['R'] && !busy['L']" 'onlyLeft := false'
mutate "mergeCol chọn node rẽ xa nhất" layout/branch.go 'if best == nil || it.Row > best.Row' 'if best == nil || it.Row < best.Row'
mutate "tắt mergeCol" layout/place.go 'if mc, ok := l.mergeCol(same, v); ok {' 'if mc, ok := l.mergeCol(same, v); false && ok {'
mutate "attachSide mặc định sang trái" layout/branch.go 'if !right {' 'if right {'
mutate "condition cần ba nhánh phụ mới giữ hai mặt" layout/place.go 'l.sideBranches(v) >= 2' 'l.sideBranches(v) >= 3'
mutate "tắt mũi tên ngang" layout/place.go 'if len(preds) == 1 && l.nonBackIn(vid) == 1' 'if false && len(preds) == 1 && l.nonBackIn(vid) == 1'
mutate "nhánh phụ chỉ dạt một cột" layout/place.go 'tries < 3' 'tries < 1'
mutate "start không về hàng 0" layout/place.go 'case v.Kind == "start":' 'case false:'
mutate "db và text thử phía ngược trước" layout/place.go '[]int{side, -side, 2 * side, -2 * side}' '[]int{-side, side, 2 * side, -2 * side}'
mutate "mũi tên ngang không tránh mũi tên ngang khác" layout/place.go 'if s.row == r && s.a.less(b) && a.less(s.b) {' 'if false {'
mutate "mũi tên ngang không tránh ô đã chiếm ở giữa" layout/place.go 'if k.row == r && a.less(gk{k.lane, k.col}) && (gk{k.lane, k.col}).less(b) {' 'if false {'
mutate "nguồn tham chiếu là nguồn nông nhất" layout/place.go 'if a.Row > b.Row ||' 'if a.Row < b.Row ||'
mutate "đảo phía hside" layout/place.go 'toRight := (gk{u.Lane, u.Col}).less(gk{v.Lane, col})' 'toRight := !(gk{u.Lane, u.Col}).less(gk{v.Lane, col})'
mutate "đếm cả nhánh chính là nhánh phụ" layout/branch.go 'return n - 1' 'return n'
mutate "đường lùi ra xa bắt đầu từ cột thứ hai" layout/place.go 'for d := 3; d < 50; d++ {' 'for d := 2; d < 50; d++ {'

# layout/route
mutate "A không kiểm mặt đáy đã có cạnh ra" layout/route.go "if len(l.sideOut[sideKey{u.ID, 'B'}]) > 0 {" 'if false {'
mutate "ô dọc cho dây khác đích chồng lên" layout/route.go 'if id != dst {' 'if false {'
mutate "B không tránh mặt có db" layout/route.go 'if l.attachSides(u)[s] || l.attachSides(v)[opp(s)] {' 'if false {'
mutate "C không tránh mặt có db" layout/route.go 'if l.sideUsed(u, s) || l.attachSides(u)[s] {' 'if l.sideUsed(u, s) {'
mutate "C không kiểm ô rẽ góc" layout/route.go 'hcells := append(l.cellsBetween(gkOf(u), gkOf(v), u.Row), turn)' 'hcells := l.cellsBetween(gkOf(u), gkOf(v), u.Row)'
mutate "cạnh back được ra đáy" layout/route.go 'if e.Back || v.Row <= u.Row {' 'if false {'
mutate "D ưu tiên ra đáy trước mặt bên" layout/route.go "choices := []byte{s, 'B', opp(s)}" "choices := []byte{'B', s, opp(s)}"
mutate "tắt lối tắt hàng liền kề của D" layout/route.go 'if v.Row == u.Row+1 {' 'if false {'
mutate "tắt lối tắt cột trống của D" layout/route.go 'if all(vcells, func(c cell) bool { return vOK(c, v.ID) }) && gkOf(u) != gkOf(v) {' 'if false {'
mutate "đảo phía máng khi ra đáy" layout/route.go 'if !gkOf(u).less(gkOf(v)) {' 'if gkOf(u).less(gkOf(v)) {'
mutate "máng bên bắt đầu từ giữa hàng thay vì dưới node" layout/route.go 'l.seg(gs, 2*u.Row+1,' 'l.seg(gs, 2*u.Row,'
mutate "đảo phía chân nối của máng bên" layout/route.go 'dir = -1' 'dir = 1'
mutate "đoạn máng khi ra đáy thành đoạn riêng" layout/route.go 's2 := l.seg(gt, 2*(u.Row+1), 2*v.Row, v.ID)' 's2 := l.seg(gt, 2*(u.Row+1), 2*v.Row, "")'
mutate "hình thoi chia sẻ một mặt cho nhiều cạnh" layout/route.go 'return len(used) == 0' 'return true'
mutate "hộp chữ nhật chia sẻ mặt đã có cạnh B hoặc C" layout/route.go "if x.Case == 'B' || x.Case == 'C' {" 'if false {'
mutate "cùng cột thì hướng sang trái" layout/route.go 'if gkOf(u) == gkOf(v) || gkOf(u).less(gkOf(v)) {' 'if gkOf(u).less(gkOf(v)) {'

# gán cổng và track
mutate "bỏ bộ cổng né giữa mặt" layout/tracks.go 'if len(fixed) > 0 && n <= 6 {' 'if false {'
mutate "mặt bên sắp cổng theo cột thay vì theo hàng" layout/tracks.go "if side == 'L' || side == 'R' {" 'if false {'
mutate "cổng mặt trái đặt ở mép phải" layout/tracks.go 'e.ExitFrac = [2]float64{0.0, fr[i]}' 'e.ExitFrac = [2]float64{1.0, fr[i]}'
mutate "không ưu tiên track dùng chung cùng đích" layout/tracks.go 'if fits(s, ti, true) {' 'if false {'
mutate "đoạn cùng đích không được chồng" layout/tracks.go 'if !(s.Key != "" && t.Key == s.Key) && overlap(t, s) {' 'if overlap(t, s) {'
mutate "bỏ ràng buộc thứ tự chân nối" layout/tracks.go 'if pa.Pos == pb.Pos && pa.Dir < pb.Dir {' 'if false {'
mutate "đoạn cùng đích vẫn chịu ràng buộc thứ tự" layout/tracks.go 'if a.Key != "" && a.Key == b.Key {' 'if false {'
mutate "hai đoạn chạm đầu không tính là chồng" layout/tracks.go 'return a.Lo <= b.Hi && b.Lo <= a.Hi' 'return a.Lo < b.Hi && b.Lo < a.Hi'
mutate "track mới luôn chèn cuối" layout/tracks.go 'chosen = max(lo, min(hi, len(tracks)))' 'chosen = len(tracks)'
mutate "orderOK bỏ qua ràng buộc chiều xuôi" layout/tracks.go 'if mustPrecede(s, t) && !(ti < tj) {' 'if false {'

# layout/geometry
mutate "chừa ít chỗ cho nhãn của cạnh ra đáy" layout/geometry.go 'e.LH+8)' 'e.LH+4)'
mutate "không chừa chỗ cho nhãn của cạnh D" layout/geometry.go "case e.Case == 'D' || (crosses && spans != nil):" 'case (crosses && spans != nil):'
mutate "thiếu hụt nhãn tính với lề 8 thay vì 16" layout/geometry.go 'e.LW + 16 - spans[e.ID]' 'e.LW + 8 - spans[e.ID]'
mutate "chỗ chừa nhãn luôn vào nửa trái máng" layout/geometry.go "key := gzKey{u.Lane, l.gutter(u.Lane, u.Col, e.ExitSide), 'r'}" "key := gzKey{u.Lane, l.gutter(u.Lane, u.Col, e.ExitSide), 'l'}"
mutate "bám sát cả db cách hai cột" layout/geometry.go 'abs(a.Col-u.Col) == 1' 'abs(a.Col-u.Col) <= 2'
mutate "bám sát cả mặt đang có dây" layout/geometry.go 'if len(near) != 1 || l.sideUsed(u, side) {' 'if len(near) != 1 {'
mutate "db bám sát vẫn chiếm bề rộng cột" layout/geometry.go 'if _, hugged := g.hugs[it.ID]; !hugged {' 'if true {'
mutate "lề header lane 20 thay vì 30" layout/geometry.go '"\n"))) + 30' '"\n"))) + 20'
mutate "tên lane nối bằng khoảng trắng thay vì xuống dòng" layout/geometry.go 'strings.Join(lrow.Lines, "\n")' 'strings.Join(lrow.Lines, " ")'
mutate "phần bù bề rộng lane dồn hết vào máng đầu" layout/geometry.go 'widths[0] += need / 2' 'widths[0] += need'
mutate "máng tính thừa một track" layout/geometry.go 'inner = float64(2*cfg.GutterMargin + (n-1)*cfg.TrackGap)' 'inner = float64(2*cfg.GutterMargin + n*cfg.TrackGap)'
mutate "kênh không có chiều cao tối thiểu" layout/geometry.go 'h := pyMax(float64(cfg.MinChannel), lzC[k]+inner)' 'h := lzC[k] + inner'
mutate "db bám bên phải không cách node" layout/geometry.go 'a.X = h.u.X + h.u.W + float64(cfg.AttachGap)' 'a.X = h.u.X + h.u.W'
mutate "trackY bỏ chỗ chừa nhãn" layout/geometry.go 'return g.chanY[s.Res.A] + g.lzC[s.Res.A] +' 'return g.chanY[s.Res.A] +'
mutate "tâm cột khác lấy mép trái" layout/geometry.go 'return c[0] + c[1]/2' 'return c[0]'
mutate "không bỏ điểm thẳng hàng" layout/geometry.go 'if vertical || horizontal {' 'if false {'
mutate "không bỏ điểm trùng" layout/geometry.go 'if math.Abs(p[0]-last[0]) < 0.01 && math.Abs(p[1]-last[1]) < 0.01 {' 'if false {'
mutate "làm tròn điểm gấp tới một chữ số" layout/geometry.go 'num.Round(out[i][0], 2)' 'num.Round(out[i][0], 1)'
mutate "bỏ lượt hình học thứ hai" layout/labels.go 'lzG, lzC = l.labelReservations(l.horizontalSpans())' 'lzG, lzC = l.labelReservations(nil)'

# layout/labels
mutate "chi phí chồng node nhẹ hơn" layout/labels.go 'cost += float64(area(c.b, nb.b) * 4)' 'cost += float64(area(c.b, nb.b) * 2)'
mutate "không tránh nhãn đã đặt" layout/labels.go 'cost += float64(area(c.b, pb) * 4)' 'cost += 0'
mutate "tính cả dây của chính cạnh" layout/labels.go 'if w.id != e.ID && segHits(w.a, w.b, c.b) {' 'if segHits(w.a, w.b, c.b) {'
mutate "không dừng ở ứng viên chi phí không" layout/labels.go 'if cost == 0 {' 'if false {'
mutate "hòa chi phí thì chọn ứng viên sau" layout/labels.go 'if best == nil || cost < bestCost {' 'if best == nil || cost <= bestCost {'
mutate "đoạn ngắn hơn 20 không đặt nhãn" layout/labels.go 'if L >= 12 {' 'if L >= 20 {'
mutate "nhãn ngang cách đầu đoạn 4 thay vì 6" layout/labels.go 'a[0] + float64(d*(6+w/2))' 'a[0] + float64(d*(4+w/2))'
mutate "nhãn dọc thử bên trái trước" layout/labels.go '[]float64{a[0] + 5 + w/2, a[0] - 5 - w/2}' '[]float64{a[0] - 5 - w/2, a[0] + 5 + w/2}'
mutate "label_t làm tròn hai chữ số" layout/labels.go 'num.Round(float64(2*best.dist)/total-1, 4)' 'num.Round(float64(2*best.dist)/total-1, 2)'
mutate "bỏ hộp header" layout/labels.go 'box{-1e6, -1e6, 1e6,' 'box{-1e6, -1e6, -1e6,'
mutate "bỏ đường phân cách lane" layout/labels.go 'for _, x := range l.LaneX[1:] {' 'for _, x := range l.LaneX[:0] {'
mutate "chạm mép hộp cũng tính là cắt" layout/labels.go 'return x1 < bx[2] && x2 > bx[0]' 'return x1 <= bx[2] && x2 >= bx[0]'
mutate "pyMax trả số sau khi bằng nhau" layout/util.go 'if b > a {' 'if b >= a {'

[ -n "$PREFLIGHT" ] && exit 0
echo
echo "bắt được $caught, bỏ lọt $missed"
[ "$missed" -eq 0 ]
