#!/bin/sh
# Kiểm xem bộ đối chiếu có thật sự bắt được lỗi hay không, bằng cách cố tình
# làm sai từng chỗ rồi xem test có đỏ không.
#
#     sh tools/mutate.sh
#     ONLY=writer/ sh tools/mutate.sh    # chỉ đột biến trên file bắt đầu bằng writer/
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

replace=$(mktemp -d)/replaceonce
go build -o "$replace" ./tools/replaceonce

caught=0
missed=0

# Bị ngắt giữa lúc một file đang mang đột biến thì khôi phục nó, nếu không cây
# làm việc sẽ còn lại code hỏng mà test vẫn có thể qua.
current=''
trap '[ -n "$current" ] && git checkout -- "$current"' EXIT INT TERM

mutate() {
	label="$1" file="$2" from="$3" to="$4"
	# Lọc trước khi chạm vào file. Đột biến bị lọc ra mà vẫn được áp thì hàm
	# thoát sớm để lại file hỏng, và trap không biết để khôi phục.
	if [ -z "$PREFLIGHT" ]; then
		case "$file" in "${ONLY:-}"*) ;; *) return 0 ;; esac
		current="$file"
	fi
	if [ -n "$PREFLIGHT" ]; then
		"$replace" -check "$file" "$from" "$to"
		return 0
	fi
	"$replace" "$file" "$from" "$to"
	if go test -count=1 ./... >/dev/null 2>&1; then
		echo "BỎ LỌT     $label"
		missed=$((missed + 1))
	else
		echo "BẮT ĐƯỢC   $label"
		caught=$((caught + 1))
	fi
	git checkout -- "$file"
	current=''
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
# - Tâm của một cột không phải cột đích trong resolvePaths. route chỉ sinh tham
#   chiếu tới cột của chính đích, nên nhánh đó không chạy. Giữ lại vì nó làm rx
#   đúng với mọi tham chiếu cột, không chỉ với những gì route đang sinh.
# - Dừng ở ứng viên nhãn chi phí không. Ứng viên được chọn bằng so sánh < nghiêm
#   ngặt, nên một ứng viên chi phí không đứng sau không bao giờ thay được ứng
#   viên chi phí không đứng trước; break chỉ là tối ưu.
# - fmax trả số sau khi hai số bằng nhau. Chỉ khác khi đó là âm không và dương
#   không, mà âm không không sinh ra được trong pha hình học: mọi toạ độ là tổng
#   bề rộng dương bắt đầu từ 0, x - x luôn ra dương không, và số âm duy nhất là
#   d = -1 nhân với một lượng dương khác không.
#
# - Dòng rỗng hoàn toàn trong readCSV không ra hàng rỗng. splitKeepEnds không
#   bao giờ sinh dòng rỗng: mỗi dòng mang ít nhất ký tự xuống dòng của nó, nên
#   máy trạng thái luôn thấy ký tự xuống dòng trước khi thấy hết dòng.
# - .text của phần tử xlsx gom cả chữ nằm sau phần tử con. Bộ đọc chỉ đọc text
#   của t và v, hai phần tử không bao giờ có con trong SpreadsheetML hợp lệ.
#
# Mỗi cái sống sót qua hàng chục nghìn bảng sinh ngẫu nhiên, hoặc có lập luận
# dựa trên cấu trúc đầu vào, trước khi được chứng minh. Suy luận mà không có số
# liệu thì không đáng tin: hai nhánh khác từng bị nghi là thừa rồi bị bảng ngẫu
# nhiên giết.
#
# Đột biến sống sót mà CHƯA chứng minh được là tương đương. Khác danh sách trên:
# đây là chỗ bộ đối chiếu có thể đang hở, chỉ là chưa ai tìm ra case.
#
# - Dây chỉ chạm mép trái hoặc phải của hộp nhãn cũng tính là cắt. Sống qua
#   133.141 bảng hợp lệ. Theo cấu trúc thì không xảy ra: hộp nhãn dọc cách dây
#   của nó đúng 5 điểm ảnh, hộp nhãn ngang cách đầu đoạn 6, còn dây dọc song
#   song cách nhau 12 và cách mép cột ít nhất 15. Ngoài ra chỉ còn trùng hợp
#   ngẫu nhiên, mà bề rộng chữ là số lẻ. Chiều trên dưới thì có case chốt.

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
mutate "không hạ chữ thường cột type" source/markdown.go 'unistr.Lower(cells[1])' 'cells[1]'
mutate "Idx dùng số dòng thay vì thứ tự đọc được" source/markdown.go 'len(rows),' 'i,'
mutate "khóa metadata cũng bị gỡ escape" source/markdown.go 'key := unistr.Strip(k)' 'key := unesc(unistr.Strip(k))'
mutate "thẻ br phân biệt hoa thường" source/text.go "(r[1] != 'b' && r[1] != 'B')" "r[1] != 'b'"
mutate "heading cấp hai cũng tính là tiêu đề" source/markdown.go 'if len(rest) < 2 || !unistr.IsSpace(rest[0]) {' 'if len(rest) < 2 {'
mutate "thứ tự key metadata theo map thay vì theo ô" source/markdown.go 'if _, seen := meta[key]; !seen {' 'if _, seen := meta[key]; seen {'

# schema và validate
mutate "đảo thứ tự kiểm from và to" schema/schema.go '{Key: "from", Required: true, Note: edgeNote},
			{Key: "to", Required: true, Note: edgeNote},' '{Key: "to", Required: true, Note: edgeNote},
			{Key: "from", Required: true, Note: edgeNote},'
mutate "attach của text thành bắt buộc" schema/schema.go '{Key: "attach", Note: attachNote, SameLane: true}' '{Key: "attach", Required: true, Note: attachNote, SameLane: true}'
mutate "edge không nhận style dashed" schema/schema.go 'StyleValues: []string{"highlight", "dashed", "bold", "noarrow"}' 'StyleValues: []string{"highlight", "bold", "noarrow"}'
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
mutate "tắt mergeCol" layout/place.go 'if mc, ok := l.mergeCol(same, v); ok && (' 'if mc, ok := l.mergeCol(same, v); false && ok && ('
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
mutate "bỏ bộ cổng né giữa mặt" layout/tracks.go '} else if (len(fixed) > 0 || entries) && n <= 6 {' '} else if false {'
mutate "dây ra không tách khỏi dây vào cùng mặt" layout/tracks.go 'entries := len(l.sideIn[key]) > 0' 'entries := false'
mutate "cổng tách không trên đường viền hình thoi" layout/shape.go '		inset = math.Abs(t - 0.5)' '		inset = 0'
mutate "condition nhận dây ngang vào mặt bên" layout/shape.go '"condition": {Shape: ShapeDiamond, EntryTopOnly: true},' '"condition": {Shape: ShapeDiamond},'
mutate "mặt bên sắp cổng theo cột thay vì theo hàng" layout/tracks.go "if side == 'L' || side == 'R' {" 'if false {'
mutate "cổng mặt trái đặt ở mép phải" layout/shape.go '	return [2]float64{inset, t}' '	return [2]float64{far, t}'
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
mutate "kênh không có chiều cao tối thiểu" layout/geometry.go 'h := fmax(float64(cfg.MinChannel), lzC[k]+inner)' 'h := lzC[k] + inner'
mutate "db bám bên phải không cách node" layout/geometry.go 'a.X = h.u.X + h.u.W + float64(cfg.AttachGap)' 'a.X = h.u.X + h.u.W'
mutate "trackY bỏ chỗ chừa nhãn" layout/geometry.go 'return g.chanY[s.Res.A] + g.lzC[s.Res.A] +' 'return g.chanY[s.Res.A] +'
mutate "không bỏ điểm thẳng hàng" layout/geometry.go 'if vertical || horizontal {' 'if false {'
mutate "không bỏ điểm trùng" layout/geometry.go 'if math.Abs(p[0]-last[0]) < 0.01 && math.Abs(p[1]-last[1]) < 0.01 {' 'if false {'
mutate "làm tròn điểm gấp tới một chữ số" layout/geometry.go 'num.Round(out[i][0], 2)' 'num.Round(out[i][0], 1)'
mutate "bỏ lượt hình học thứ hai" layout/labels.go 'lzG, lzC = l.labelReservations(l.horizontalSpans())' 'lzG, lzC = l.labelReservations(nil)'

# layout/labels
mutate "chi phí chồng node nhẹ hơn" layout/labels.go 'cost += float64(area(c.b, nb.b) * 4)' 'cost += float64(area(c.b, nb.b) * 2)'
mutate "không tránh nhãn đã đặt" layout/labels.go 'cost += float64(area(c.b, pb) * 4)' 'cost += 0'
mutate "tính cả dây của chính cạnh" layout/labels.go 'if w.id != e.ID && segHits(w.a, w.b, c.b) {' 'if segHits(w.a, w.b, c.b) {'
mutate "hòa chi phí thì chọn ứng viên sau" layout/labels.go 'if best == nil || cost < bestCost {' 'if best == nil || cost <= bestCost {'
mutate "đoạn ngắn hơn 20 không đặt nhãn" layout/labels.go 'if L >= 12 {' 'if L >= 20 {'
mutate "nhãn ngang cách đầu đoạn 4 thay vì 6" layout/labels.go 'a[0] + float64(d*(6+w/2))' 'a[0] + float64(d*(4+w/2))'
mutate "nhãn dọc thử bên trái trước" layout/labels.go '[]float64{a[0] + 5 + w/2, a[0] - 5 - w/2}' '[]float64{a[0] - 5 - w/2, a[0] + 5 + w/2}'
mutate "label_t làm tròn hai chữ số" layout/labels.go 'num.Round(float64(2*best.dist)/total-1, 4)' 'num.Round(float64(2*best.dist)/total-1, 2)'
mutate "bỏ hộp header" layout/labels.go 'box{-1e6, -1e6, 1e6,' 'box{-1e6, -1e6, -1e6,'
mutate "bỏ đường phân cách lane" layout/labels.go 'for _, x := range l.LaneX[1:] {' 'for _, x := range l.LaneX[:0] {'
mutate "chạm mép trên dưới của hộp cũng tính là cắt" layout/labels.go 'return x1 < bx[2] && x2 > bx[0] && y1 < bx[3] && y2 > bx[1]' 'return x1 < bx[2] && x2 > bx[0] && y1 <= bx[3] && y2 >= bx[1]'

# layout/check
mutate "không kiểm node chồng node" layout/check.go 'if area(boxes[a], boxes[b]) > 0 {' 'if false {'
mutate "chỉ kiểm tràn lane ở mép trái" layout/check.go 'if it.X < lx || it.X+it.W > lx+lw {' 'if it.X < lx {'
mutate "đoạn xiên chỉ cần lệch một trục" layout/check.go 'if math.Abs(a[0]-b[0]) > 0.01 && math.Abs(a[1]-b[1]) > 0.01 {' 'if math.Abs(a[0]-b[0]) > 0.01 || math.Abs(a[1]-b[1]) > 0.01 {'
mutate "miễn mọi đoạn chạm node nguồn và đích" layout/check.go '(it.ID == e.Src || it.ID == e.Dst) && (k == 0 || k == len(pts)-2)' '(it.ID == e.Src || it.ID == e.Dst)'
mutate "không thu hộp node trước khi kiểm cắt" layout/check.go 'segHits(a, b, shrink(boxes[it.ID], 1))' 'segHits(a, b, shrink(boxes[it.ID], 0))'
mutate "dây cùng đích cũng bị tính chồng" layout/check.go 'if w1.edge == w2.edge || e1.Dst == e2.Dst || e1.Src == e2.Src {' 'if w1.edge == w2.edge || e1.Src == e2.Src {'
mutate "dây cùng nguồn cũng bị tính chồng" layout/check.go 'if w1.edge == w2.edge || e1.Dst == e2.Dst || e1.Src == e2.Src {' 'if w1.edge == w2.edge || e1.Dst == e2.Dst {'
mutate "chồng dây cần dài hơn 3 điểm ảnh" layout/check.go 'if collinearOverlap(w1.a, w1.b, w2.a, w2.b) > 1 {' 'if collinearOverlap(w1.a, w1.b, w2.a, w2.b) > 3 {'
mutate "hai dây ngang cách dưới 0.1 mới coi là cùng đường" layout/check.go 'math.Abs(a1[1]-a2[1]) < 0.5' 'math.Abs(a1[1]-a2[1]) < 0.1'
mutate "không kiểm dây dọc chồng nhau" layout/check.go 'if math.Abs(a1[0]-b1[0]) < 0.01 && math.Abs(a2[0]-b2[0]) < 0.01 && math.Abs(a1[0]-a2[0]) < 0.5 {' 'if false {'
mutate "không kiểm nhãn đè node" layout/check.go 'if area(lb.b, boxes[it.ID]) > 0 {' 'if false {'
mutate "không kiểm nhãn đè nhãn" layout/check.go 'if area(lb.b, lb2.b) > 0 {' 'if false {'
mutate "nhãn đè dây của chính cạnh cũng báo" layout/check.go 'if w.edge != lb.edge && segHits(w.a, w.b, lb.b) {' 'if segHits(w.a, w.b, lb.b) {'
mutate "không bỏ phát hiện trùng" layout/check.go 'if !seen[f] {' 'if true {'
mutate "cặp node chồng không sắp theo id" layout/check.go 'sort.Strings(ids)' 'sort.Sort(sort.Reverse(sort.StringSlice(ids)))'
mutate "số thứ tự đoạn xiên đếm từ 1" layout/check.go '"%s: đoạn %d không vuông góc", e.ID, k)' '"%s: đoạn %d không vuông góc", e.ID, k+1)'

# writer/drawio
mutate "không thoát & trong nội dung html" writer/drawio/write.go 'r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")' 'r := strings.NewReplacer("<", "&lt;", ">", "&gt;")'
mutate "nối dòng bằng xuống dòng thay vì br" writer/drawio/write.go 'return strings.Join(out, "<br>")' 'return strings.Join(out, "\n")'
mutate "tên trang cắt 40 ký tự" writer/drawio/write.go 'if len(name) > 80 {' 'if len(name) > 40 {'
mutate "tên trang cắt theo byte thay vì rune" writer/drawio/write.go 'name := []rune(title)' 'name := []rune(string([]byte(title)[:min(len(title), 80)]))'
mutate "ghi chú tô nhấn dùng style của node" writer/drawio/write.go 'if it.Shape == layout.ShapeNote {' 'if false {'
mutate "bỏ nét đứt" writer/drawio/write.go 'style += "dashed=1;"' 'style += ""'
mutate "bỏ thuộc tính x của nhãn" writer/drawio/write.go 'g.Set("x", f(e.LabelT))' '_ = e.LabelT'
mutate "ghi cả điểm đầu và điểm cuối vào points" writer/drawio/write.go 'pts = e.Pts[1 : len(e.Pts)-1]' 'pts = e.Pts'
mutate "dây giữ từ file cũ vẫn ghi điểm neo tính được" writer/drawio/write.go '		case e.Kept:
			for _, kv := range e.Constraints {' '		case false:
			for _, kv := range e.Constraints {'
mutate "dây tự đi vẫn ghi điểm neo" writer/drawio/write.go '		switch {
		case e.Auto:
		case e.Kept:
			for' '		switch {
		case false:
		case e.Kept:
			for'
mutate "dây giữ từ file cũ vẫn ghi điểm gấp tính được" writer/drawio/write.go '			pts = e.Waypoints' '			pts = e.Pts'
mutate "nhãn về giữa đường vẫn ghi vị trí" writer/drawio/write.go 'withLabel := e.Label != nil && !e.NoLabelPos && !e.Auto' 'withLabel := e.Label != nil && !e.Auto'
mutate "dây tự đi vẫn ghi vị trí nhãn" writer/drawio/write.go 'withLabel := e.Label != nil && !e.NoLabelPos && !e.Auto' 'withLabel := e.Label != nil && !e.NoLabelPos'
mutate "bỏ các cell tự vẽ" writer/drawio/write.go '	root.Children = append(root.Children, extras...)' '	_ = extras'
mutate "bỏ các trang khác" writer/drawio/write.go '	mxfile.Children = append(mxfile.Children, pages...)' '	_ = pages'

# internal/etree
mutate "không thoát dấu nháy kép trong thuộc tính" internal/etree/etree.go '"\"", "&quot;",' ''
mutate "không thoát tab trong thuộc tính" internal/etree/etree.go '"\t", "&#09;")' '"\t", "\t")'
mutate "không thoát xuống dòng trong thuộc tính" internal/etree/etree.go '"\n", "&#10;",' '"\n", "\n",'
mutate "thoát dấu nháy kép trong chữ" internal/etree/etree.go 'textEsc = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")' 'textEsc = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")'
mutate "phần tử rỗng không có khoảng trắng trước gạch chéo" internal/etree/etree.go 'b.WriteString(" />")' 'b.WriteString("/>")'
mutate "phần tử chỉ có chữ ghi thành thẻ rỗng" internal/etree/etree.go 'if e.Text != "" || len(e.Children) > 0 {' 'if len(e.Children) > 0 {'
mutate "bỏ tail khi ghi" internal/etree/etree.go '	b.WriteString(textEsc.Replace(e.Tail))' '	_ = e.Tail'
mutate "thụt lề bốn khoảng trắng" internal/etree/etree.go 'levels = append(levels, levels[level]+"  ")' 'levels = append(levels, levels[level]+"    ")'
mutate "thụt lề ghi đè cả chữ thật" internal/etree/etree.go '		if unistr.Strip(e.Text) == "" {
			e.Text = child' '		if true {
			e.Text = child'
mutate "thụt lề ghi đè cả tail thật" internal/etree/etree.go '			if unistr.Strip(c.Tail) == "" {' '			if true {'
mutate "tail của con cuối không lùi về cấp cha" internal/etree/etree.go '			last.Tail = levels[level]' '			last.Tail = child'
mutate "chữ sau phần tử con dồn vào text của cha" internal/etree/etree.go '			if len(cur.Children) == 0 {' '			if true {'
mutate "không chuẩn hóa xuống dòng trong thuộc tính" internal/etree/etree.go '			case '"'"'\n'"'"', '"'"'\t'"'"':
				out.WriteByte('"'"' '"'"')' '			case '"'"'\t'"'"':
				out.WriteByte('"'"' '"'"')'
mutate "CRLF trong thuộc tính thành hai khoảng trắng" internal/etree/etree.go '				if i < n && data[i] == '"'"'\n'"'"' {
					i++
				}' ''
mutate "chuẩn hóa cả trong chú thích" internal/etree/etree.go '		if j, ok := skip(i, "<!--", "-->"); ok {' '		if j, ok := skip(i, "<!-- ", "-->"); ok {'
mutate "Set đổi thuộc tính có sẵn ra cuối" internal/etree/etree.go '		if e.Attrs[i][0] == k {
			e.Attrs[i][1] = v
			return
		}' ''
mutate "Copy dùng chung thuộc tính" internal/etree/etree.go 'Attrs: append([][2]string(nil), e.Attrs...)}' 'Attrs: e.Attrs}'

mutate "lane không trừ header của pool" writer/drawio/write.go 'geo(c, r.LaneX[i], float64(r.PoolHeader), r.LaneW[i], r.PoolH-float64(r.PoolHeader))' 'geo(c, r.LaneX[i], float64(r.PoolHeader), r.LaneW[i], r.PoolH)'

# cmd/flowcast
mutate "không đưa lỗi lên trước cảnh báo" cmd/flowcast/main.go 'return sorted[i].Level == model.LevelError && sorted[j].Level != model.LevelError' 'return false'
mutate "dòng build viết kích thước có khoảng trắng" cmd/flowcast/main.go '(%d lane, %d phần tử, %d cạnh, %sx%spx)' '(%d lane, %d phần tử, %d cạnh, %s x %spx)'
mutate "đường ra mặc định giữ đuôi nguồn" cmd/flowcast/main.go 'out = strings.TrimSuffix(c.a.file, source.Ext(c.a.file)) + ".drawio"' 'out = c.a.file + ".drawio"'
mutate "dấu chấm đầu tên file tính là dấu tách đuôi" source/source.go 'trimmed := strings.TrimLeft(base, ".")' 'trimmed := base'
mutate "thông điệp lỗi hệ thống không viết hoa chữ đầu" cmd/flowcast/main.go 'r[0] = unicode.ToUpper(r[0])' '_ = r'
mutate "force không sao lưu file cũ" cmd/flowcast/main.go 'if mode != "new" && !c.a.noBackup {' 'if mode == "merge" && !c.a.noBackup {'
mutate "không có terminal vẫn hỏi chế độ" cmd/flowcast/main.go 'if !isTerminal(c.stdin) {' 'if false {'
mutate "lỗi hình học không đổi mã thoát" cmd/flowcast/main.go '		code = 2' '		code = 0'
mutate "bỏ tiền tố cảnh báo layout" cmd/flowcast/main.go 'c.println("WARNING layout: " + w.Msg)' 'c.println(w.Msg)'
mutate "không nhận .txt là markdown" cmd/flowcast/main.go '".txt": true,' ''
mutate "check không trả mã 1 khi bảng lỗi" cmd/flowcast/main.go '	if c.printIssues(r.Issues) > 0 {
		return 1
	}
	return 0
}' '	c.printIssues(r.Issues)
	return 0
}'
mutate "--no-backup không có tác dụng" cmd/flowcast/args.go 'a.noBackup = true' 'a.noBackup = false'
mutate "cờ cấu hình đứng trước tên file bị bỏ qua" cmd/flowcast/args.go '			*ints[name] = n' '			_ = n'
mutate "cú pháp --cờ=giá trị không tách giá trị" cmd/flowcast/args.go 'name, val, hasVal := strings.Cut(tok[2:], "=")' 'name, val, hasVal := tok[2:], "", false'

# source/csv
mutate "chữ sau dấu nháy đóng không được nối vào ô" source/csvread.go '				state = inField
				return add(c)
			}
		case eatCRNL:' '				state = inField
				return nil
			}
		case eatCRNL:'
mutate "hết dữ liệu trong dấu nháy thì bỏ ô dở" source/csvread.go 'if len(field) != 0 || state == inQuotedField {' 'if len(field) != 0 {'
mutate "CR đơn không là ranh giới dòng khi đọc csv" source/csvread.go 'i := strings.IndexAny(s, "\r\n")' 'i := strings.IndexAny(s, "\n")'
mutate "utf-8-sig không bỏ BOM" source/encoding.go 'return strings.TrimPrefix(string(raw), "\ufeff"), true' 'return string(raw), true'
mutate "cp1252 nhận cả byte không được định nghĩa" source/encoding.go '			if !ok {
				return "", false
			}
			b.WriteRune(r)' '			_ = ok
			b.WriteRune(r)'
mutate "thử cp1252 trước utf-8" source/encoding.go 'var csvEncodings = []string{"utf-8-sig", "utf-8", "cp1252"}' 'var csvEncodings = []string{"cp1252", "utf-8-sig", "utf-8"}'
mutate "đoán dấu chấm phẩy trước dấu phẩy" source/grid.go 'delims := []string{",", ";", "\t"}' 'delims := []string{";", ",", "\t"}'
mutate "header phải bắt đầu ở cột đầu" source/grid.go 'for c0 := 0; c0 < max(1, len(low)-4); c0++ {' 'for c0 := 0; c0 < 1; c0++ {'
mutate "tiêu đề không lấy từ phía trên header" source/grid.go 'for r := 0; r < hr && title == ""; r++ {' 'for r := 0; r < 0 && title == ""; r++ {'
mutate "bảng không dừng ở hàng trống" source/grid.go '		if empty {
			break
		}' '		if empty {
			continue
		}'
mutate "không cảnh báo escape markdown trong csv" source/grid.go 'if strings.Contains(cells[3], `\|`) || strings.Contains(cells[4], `\|`) {' 'if false {'
mutate "ô csv không tách dòng tại xuống dòng thật" source/grid.go 'parts = append(parts, strings.Split(chunk, "\n")...)' 'parts = append(parts, chunk)'
mutate "cảnh báo cp1252 so tên đã chuẩn hóa" source/grid.go 'if used == "cp1252" {' 'if c, _ := normalizeEncoding(used); c == "cp1252" {'
mutate "khoảng trắng bỏ U+001C tới U+001F" internal/unistr/unistr.go 'return unicode.IsSpace(r) || (r >= 0x1c && r <= 0x1f)' 'return unicode.IsSpace(r)'
mutate "Lower không tách İ thành hai ký tự" internal/unistr/unistr.go 'if r == 0x130 {' 'if false {'
mutate "thẻ br không nhận khoảng trắng Unicode" source/text.go 'for i < len(r) && unistr.IsSpace(r[i]) {
		i++
	}
	if i < len(r) && r[i] == '"'"'/'"'"' {' 'for i < len(r) && r[i] == '"'"' '"'"' {
		i++
	}
	if i < len(r) && r[i] == '"'"'/'"'"' {'
mutate "tiêu đề toàn khoảng trắng không khớp" source/markdown.go '		return unescape(string(rest[len(rest)-1:])), true' '		return "", false'
mutate "dòng phân cách cắt cả khoảng trắng cuối" source/markdown.go '	s := unistr.LStrip(ln)
	if !strings.HasPrefix(s, "|") || len(s) == 1 {' '	s := unistr.Strip(ln)
	if !strings.HasPrefix(s, "|") || len(s) == 1 {'

# source/xlsx
mutate "chuỗi dùng chung đọc thành rỗng" source/xlsx.go '						val = shared[i]' '						_ = i'
mutate "chuỗi nội tuyến đọc thành rỗng" source/xlsx.go '					val = is.textOf("t")' '					_ = is'
mutate "số nguyên không bỏ phần .0" source/xlsx.go "					val = val[:strings.IndexByte(val, '.')]" '					_ = val'
mutate "chỉ cảnh báo ô số ở cột id" source/xlsx.go '(ci == 0 || ci == 2)' '(ci == 0)'
mutate "không cảnh báo ô công thức rỗng" source/xlsx.go 'if c.find("f") != nil {' 'if false {'
mutate "không cảnh báo hàng ẩn" source/xlsx.go 'if h, _ := rowEl.get("hidden"); h == "1" {' 'if false {'
mutate "ô gộp không mang số hàng" source/xlsx.go '		r, _ := strconv.Atoi(digits)' '		r := 0 * len(digits)'
mutate "cảnh báo cả ngoài vùng bảng" source/xlsx.go 'if hr <= n.row && n.row <= hr+len(rows) {' 'if true {'
mutate "sheet không có header thì dừng thay vì thử sheet sau" source/xlsx.go 'if _, _, found := findHeader(grid); !found {' 'if false {'
mutate "đường dẫn sheet tuyệt đối không bỏ dấu gạch đầu" source/xlsx.go 'if strings.HasPrefix(target, "/xl/") {' 'if false {'
mutate "--sheet không lọc" source/xlsx.go '			if s.name == sheet {' '			if true {'
mutate "cột lấy theo thứ tự ô thay vì theo địa chỉ" source/xlsx.go '				ci = colIndex(ref)' '				ci = len(cells)'

# merge
mutate "số không nhận khoảng trắng hai đầu" merge/parse.go '	t := strings.Trim(b.String(), " \t\n\v\f\r")' '	t := b.String()'
mutate "số không nhận dấu gạch dưới" merge/parse.go "			if c == '_' && n > 0 && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {" '			if false {'
mutate "số không nhận inf" merge/parse.go '	case "inf", "infinity":' '	case "infinity":'
mutate "số nhận hai dấu" merge/parse.go '	if len(t)-len(body) > 1 {' '	if len(t)-len(body) > 2 {'
mutate "base64 dừng ở mọi dấu đệm" merge/parse.go '			if quad >= 2 {
				pads++' '			if true {
				pads++'
mutate "base64 báo lỗi ký tự lạ" merge/parse.go '		if v < 0 {
			continue
		}' '		if v < 0 {
			return nil, errors.New("Incorrect padding")
		}'
mutate "unquote giải cả %XX sai" merge/parse.go '			if ok1 && ok2 {' '			if ok1 || ok2 {'
mutate "bỏ giải %XX" merge/parse.go '		b.WriteString(decodeReplace(unquoteBytes(s[:n])))' '		b.WriteString(s[:n])'
mutate "UTF-8 hỏng thay từng byte" merge/parse.go '		if need > 0 && j-i == need+1 {
			b.Write(p[i:j])
		} else {
			b.WriteRune(utf8.RuneError)
		}
		i = j' '		if need > 0 && j-i == need+1 {
			b.Write(p[i:j])
			i = j
		} else {
			b.WriteRune(utf8.RuneError)
			i++
		}'
mutate "đọc trang đầu bỏ qua phần tử object" merge/read.go '			if cell = el.Find("mxCell"); cell == nil {' '			if cell = nil; cell == nil {'
mutate "id trùng giữ cell đầu" merge/read.go '		old.cells[cid] = &oldCell{cid, el, cell, i}' '		if old.cells[cid] == nil {
			old.cells[cid] = &oldCell{cid, el, cell, i}
		}'
mutate "không giữ các trang khác" merge/read.go '		old.Pages = pages[1:]' '		_ = pages'
mutate "style không ghi nhận khóa không có dấu bằng" merge/read.go '		st.vals[k] = styleVal{v, found}' '		st.vals[k] = styleVal{v, true}'
mutate "số hỏng đọc thành một" merge/read.go '	v, ok := parseFloat(s)
	if !ok {
		return 0
	}' '	v, ok := parseFloat(s)
	if !ok {
		return 1
	}'
mutate "id theo quy ước không nhận chữ số con" merge/merge.go '	if strings.HasPrefix(s, "E") && digitsDots(s[1:]) {' '	if strings.HasPrefix(s, "E") && digitsDots(strings.Split(s[1:], ".")[0]) && !strings.Contains(s, ".") {'
mutate "id theo quy ước không nhận dấu gạch dưới" merge/merge.go "s[i] >= '0' && s[i] <= '9' || s[i] == '_') {" "s[i] >= '0' && s[i] <= '9') {"
mutate "không có dấu thì mọi cell đều là của tool" merge/merge.go '		return shaped && c.style().has("flowtable")' '		return true'
mutate "lane không nhận theo style swimlane" merge/merge.go '	laneLike := c.parent() == "pool" && strings.HasPrefix(c.styleRaw(), "swimlane")' '	laneLike := false'
mutate "không giữ vị trí pool cũ" merge/merge.go '		r.Origin = [2]float64{b[0], b[1]}' '		_ = b'
mutate "gốc toạ độ không cộng dồn qua cha" merge/merge.go '		cid = c.parent()
	}
	return x, y' '		break
	}
	return x, y'
mutate "lane giữ bề rộng mới tính" merge/merge.go '				w = ow' '				_ = ow'
mutate "lane chưa sửa lấy bề rộng đã làm tròn" merge/merge.go 'if ow := attrf(oc.geo(), "width"); num.Fmt(ow) != num.Fmt(w) {' 'if ow := attrf(oc.geo(), "width"); true {'
mutate "lane đã xoá theo lane cuối thay vì lane kế tiếp" merge/merge.go '					target, base = t, newX[t]
					break' '					_ = t
					break'
mutate "mapX trước lane đầu lấy lane cuối" merge/merge.go '		return m.spans[0].dx, m.spans[0].target' '		return m.spans[len(m.spans)-1].dx, m.spans[len(m.spans)-1].target'
mutate "mapX nhận cả mép phải" merge/merge.go '		if s.x0 <= x && x < s.x1 {' '		if s.x0 <= x && x <= s.x1 {'
mutate "node đổi lane vẫn giữ vị trí" merge/merge.go '		if c.parent() != laneID {' '		if false {'
mutate "node giữ vị trí không theo lane dịch" merge/merge.go '			dx = r.LaneX[it.Lane] - attrf(oc.geo(), "x")
		}
		it.X, it.Y' '			_ = oc
		}
		it.X, it.Y'
mutate "neo tìm cạnh ra trước cạnh vào" merge/merge.go '	for _, e := range byBackOrder(m.ins[v.ID]) {
		if m.inFinal[e.Src] {' '	for _, e := range byBackOrder(nil) {
		if m.inFinal[e.Src] {'
mutate "neo không xét cạnh lặp sau cùng" merge/merge.go '			return !out[i].Back' '			return out[i].Back'
mutate "không lan rộng tìm neo" merge/merge.go '		frontier = nxt
	}' '		frontier = nil
	}'
mutate "hàng xóm không xếp cạnh lặp sau cùng" merge/merge.go '	for _, back := range []bool{false, true} {' '	for _, back := range []bool{true, false} {'
mutate "vật cản bỏ cell tự vẽ" merge/merge.go '	return append(out, m.fhBoxes...)' '	return out'
mutate "node mới không dịch khi chồng" merge/merge.go '		it.Y += step' '		break'
mutate "chồng không tính lề" merge/merge.go '		b := box{it.X - pad, it.Y - pad, it.X + it.W + pad, it.Y + it.H + pad}' '		b := box{it.X, it.Y, it.X + it.W, it.Y + it.H}'
mutate "không kéo node mới vào trong lane" merge/merge.go '	if lw >= it.W+2*pad {' '	if false {'
mutate "node mới xếp theo thứ tự bảng thay vì topo" merge/merge.go '		if a != b {
			return a < b
		}
		return flow[i].Order < flow[j].Order' '		return flow[i].Order < flow[j].Order'
mutate "node không có neo đặt ở đầu lane" merge/merge.go '					bottom = fmax(bottom, b[3])' '					bottom = fmin(bottom, b[3])'
mutate "node khác lane vẫn giữ khoảng lệch x của neo" merge/merge.go '	if an.Lane == it.Lane {
		cx = ax + fv[0] - fa[0]' '	if true {
		cx = ax + fv[0] - fa[0]'
mutate "dây giữ dù đầu nối đã đổi" merge/merge.go '			keep = sok && dok && src == e.Src && dst == e.Dst && m.pinned[e.Src] && m.pinned[e.Dst]' '			keep = sok && dok && m.pinned[e.Src] && m.pinned[e.Dst]'
mutate "dây giữ dù node đầu nối đã được đặt lại" merge/merge.go '			keep = sok && dok && src == e.Src && dst == e.Dst && m.pinned[e.Src] && m.pinned[e.Dst]' '			keep = sok && dok && src == e.Src && dst == e.Dst'
mutate "điểm gấp không theo lane dịch" merge/merge.go '			e.Waypoints = append(e.Waypoints, [2]float64{ax + dx, ay})' '			e.Waypoints = append(e.Waypoints, [2]float64{ax, ay})'
mutate "giữ cả khóa neo không có giá trị" merge/merge.go '			if v, ok := st.vals[k]; ok && v.set {' '			if v, ok := st.vals[k]; ok {'
mutate "điểm gấp lệch bao nhiêu vẫn coi là chưa sửa" merge/merge.go '		if math.Abs(a[0]-b[0]) > 2 || math.Abs(a[1]-b[1]) > 2 {' '		if false {'
mutate "điểm neo sửa vẫn coi là chưa sửa" merge/merge.go '		if !ok || num.Fmt(v) != num.Fmt(w.v) {' '		if !ok {'
mutate "điểm neo so không làm tròn" merge/merge.go '		if !ok || num.Fmt(v) != num.Fmt(w.v) {' '		if !ok || v != w.v {'
mutate "dây sửa tay không có nhãn vẫn báo nhãn về giữa" merge/merge.go '			if len(e.Lines) > 0 {
				m.rep.LabelsCentered' '			if true {
				m.rep.LabelsCentered'
mutate "điểm gấp không theo lane nới" merge/merge.go '				e.Waypoints[k][0] += dx' '				_ = k'
mutate "cell tự vẽ trong pool không theo lane dịch" merge/merge.go '			dx, _ := m.mapX(x + b[2]/2)
			x += dx' '			_, _ = m.mapX(x + b[2]/2)'
mutate "giữ cell tự vẽ có cha đã xoá" merge/merge.go '			ok = m.gen[p] || isLayer(p) || kept[p]' '			ok = true'
mutate "bỏ cell tự vẽ trong lane đã xoá" merge/merge.go '			ok = m.oldLane(p) != nil' '			ok = false'
mutate "cell tự vẽ trong lane đã xoá không đổi sang pool" merge/merge.go '				cell.Set("parent", "pool")' '				_ = cell'
mutate "cell tự vẽ trong lane đã xoá không cộng gốc lane" merge/merge.go '					g.Set("x", num.Fmt(attrf(g, "x")+ax-m.ox))' '					_ = ax'
mutate "không tháo đầu dây tự vẽ" merge/merge.go '		if c.edge() {
			m.detach(c, cell, g, alive, kept)
		}' ''
mutate "tháo cả đầu nối vào cell tự vẽ còn giữ" merge/merge.go '		if ref == "" || kept[ref] || (alive[ref] && m.gen[ref]) {' '		if ref == "" || (alive[ref] && m.gen[ref]) {'
mutate "điểm tự do không bỏ điểm cũ cùng tên" merge/merge.go '				g.Remove(pt)' '				_ = pt'
mutate "dời cả điểm offset" merge/merge.go '		if as, _ := pt.Get("as"); as != "offset" {' '		if true {'
mutate "geometry tương đối dời x và y" merge/merge.go '	if v, _ := g.Get("relative"); v == "1" {' '	if false {'
mutate "cell tự vẽ theo lane không theo lane nới" merge/merge.go '		m.addMovable(lane, func(dx, dy float64) { shiftGeo(g, dx, dy) }, true)' '		_ = lane'
mutate "cell tự vẽ tuyệt đối không trừ gốc pool" merge/merge.go '	if fr == "abs" {
		ox = m.ox
	}' ''
mutate "dây tự vẽ không theo lane" merge/merge.go '		pt.Set("x", num.Fmt(attrf(pt, "x")+dx))
		m.addMovable' '		m.addMovable'
mutate "lane nới không dịch lane sau" merge/merge.go '			r.LaneX[j] += gl + gr' '			_ = j'
mutate "lane nới dịch cả thứ trong lane bằng tổng" merge/merge.go '			if mv.lane == i && gl != 0 {
				mv.shift(gl, 0)' '			if mv.lane == i && gl != 0 {
				mv.shift(gl+gr, 0)'
mutate "lane nới dịch cả thứ tương đối ở lane sau" merge/merge.go '			} else if mv.lane > i && !mv.rel {' '			} else if mv.lane > i {'
mutate "lane nới không có lề" merge/merge.go '		gr := fmax(0.0, right-(lw-pad))' '		gr := fmax(0.0, right-lw)'
mutate "không đẩy sơ đồ xuống khi bị kéo lên đầu" merge/merge.go '	if low < top {' '	if false {'
mutate "đầu lane không tính lề" merge/merge.go '	top := float64(r.PoolHeader + r.LaneHeader + pad)' '	top := float64(r.PoolHeader + r.LaneHeader)'
mutate "chiều cao pool bỏ điểm gấp" merge/merge.go '			bottoms = append(bottoms, p[1])
		}
	}
	for _, b := range m.freehandBoxesNow() {' '		}
	}
	for _, b := range m.freehandBoxesNow() {'
mutate "chiều cao pool bỏ cell tự vẽ" merge/merge.go '		bottoms = append(bottoms, b[3])' '		_ = b'
mutate "chiều cao pool không chừa kênh cuối" merge/merge.go '		h = fmax(h, b+float64(r.MinChannel))' '		h = fmax(h, b)'
mutate "cell tự vẽ theo lane đo không cộng header" merge/merge.go '			x, y = x+m.r.LaneX[f.lane], y+float64(m.r.PoolHeader)' '			x = x + m.r.LaneX[f.lane]'
mutate "không báo element chồng nhau" merge/merge.go '				m.rep.Overlaps = append(m.rep.Overlaps, a.ID+"/"+b.ID)' '				_ = b'
mutate "báo cả cell pool là bị xoá" merge/merge.go '		if !isLayer(id) && id != "pool" && !m.tableIDs[id] {' '		if !isLayer(id) && !m.tableIDs[id] {'
mutate "fsum không bù sai số" merge/merge.go '	if lo != 0 && !math.IsInf(lo, 0) && !math.IsNaN(lo) {' '	if false {'
mutate "merge không sao lưu file cũ" cmd/flowcast/main.go 'if mode != "new" && !c.a.noBackup {' 'if mode == "force" && !c.a.noBackup {'
mutate "merge in báo cáo nhưng vẫn in tự kiểm" cmd/flowcast/main.go '		c.println("layout: merge không tự kiểm dây và nhãn; xem danh sách ở trên và ảnh PNG")
		return c.exportAndVerify(out, r, 0)' '		c.println("layout: merge không tự kiểm dây và nhãn; xem danh sách ở trên và ảnh PNG")
		for _, f := range r.Findings {
			c.println(f.Msg)
		}
		return c.exportAndVerify(out, r, 0)'
# render
mutate "kiểm render không bỏ qua khi SVG không có id" render/render.go '	if len(cells) == 0 {' '	if false {'
mutate "hình neo lấy cả rect không có width" render/render.go 'case n.tag == "rect" && n.attrs["width"] != "":' 'case n.tag == "rect":'
mutate "hình neo ellipse không trừ bán kính" render/render.go 'rect = &[2]float64{attrFloat(n, "cx") - attrFloat(n, "rx"), attrFloat(n, "cy") - attrFloat(n, "ry")}' 'rect = &[2]float64{attrFloat(n, "cx"), attrFloat(n, "cy")}'
mutate "hình neo path lấy điểm đầu thay vì góc" render/render.go '				rect = &[2]float64{mx, my}' '				rect = &[2]float64{nums[0], nums[1]}'
mutate "neo là phần tử cuối" render/render.go '		if it.Order < anchor.Order {' '		if it.Order > anchor.Order {'
mutate "path của dây nhận cả path tô màu" render/render.go 'if n.tag == "path" && n.attrs["d"] != "" && n.attrs["fill"] == "none" {' 'if n.tag == "path" && n.attrs["d"] != "" {'
mutate "sai lệch cho phép 5 điểm ảnh" render/render.go '	const tol = 2.0' '	const tol = 5.0'
mutate "so cả điểm cuối như điểm thường" render/render.go '		for k := 0; k < len(got)-1; k++ {' '		for k := 0; k < len(got); k++ {'
mutate "báo hết điểm lệch của một dây" render/render.go '					e.ID, k, a[0], a[1], b[0], b[1]))
				break' '					e.ID, k, a[0], a[1], b[0], b[1]))'
mutate "đoạn cuối so sai trục" render/render.go '			bad = math.Abs(lastG[0]-lastW[0]) > tol
		} else {' '			bad = math.Abs(lastG[1]-lastW[1]) > tol
		} else {'
mutate "id trùng giữ g đầu" render/render.go '		if id := n.attrs["data-cell-id"]; n.tag == "g" && id != "" {' '		if id := n.attrs["data-cell-id"]; n.tag == "g" && id != "" && cells[id] == nil {'
mutate "đọc cả phần tử ngoài namespace svg" render/render.go '			if t.Name.Space == svgNS {' '			if true {'
mutate "export không đặt tỉ lệ" render/render.go '	if scale != 0 {' '	if false {'
mutate "export lỗi không lấy stdout khi stderr rỗng" render/render.go '				msg = strings.TrimSpace(stdout.String())' '				_ = stdout'
mutate "export coi thiếu file ra là thành công" render/render.go 'if _, statErr := os.Stat(out); err != nil || statErr != nil {' 'if err != nil {'
mutate "lỗi xuất ảnh ghi đè mã tự kiểm" cmd/flowcast/main.go '		if code != 0 {
			return code
		}
		return 3' '		return 3'
mutate "kiểm render ghi đè mã tự kiểm" cmd/flowcast/main.go '		if len(probs) > 0 && code == 0 {' '		if len(probs) > 0 {'
mutate "merge vẫn kiểm render" cmd/flowcast/main.go '	if r.Merge != nil && verify {' '	if false {'
mutate "png mặc định giữ đuôi .drawio" cmd/flowcast/main.go 'png = strings.TrimSuffix(out, source.Ext(out)) + ".png"' 'png = out + ".png"'

# giới hạn tài nguyên
mutate "không chặn kích thước đầu vào" limits.go '	if b.lim.MaxBytes > 0 && len(src.Data) > b.lim.MaxBytes {' '	if false {'
mutate "chặn số dòng lệch một" limits.go '	if b.lim.MaxRows > 0 && len(t.Rows) > b.lim.MaxRows {' '	if b.lim.MaxRows > 0 && len(t.Rows) >= b.lim.MaxRows {'
mutate "không đếm cạnh" limits.go '			edges++' '			_ = r'
mutate "không truyền giới hạn giải nén" limits.go '		src.MaxUnzipped = b.lim.MaxUnzipped' '		_ = src'
mutate "không kiểm thời gian" limits.go '	if b.lim.Timeout > 0 && time.Since(b.start) > b.lim.Timeout {' '	if false {'
mutate "giải nén không đếm dồn" source/xlsx.go '		unzipped += int64(len(b))' '		_ = b'
mutate "Check bỏ qua giới hạn" build.go '	t, err := parseWithin(src, newBudget(opt.Limits))' '	t, err := parseWithin(src, newBudget(nil))'

# cmd/flowcastd
mutate "web không giới hạn thân yêu cầu" cmd/flowcastd/server.go '	r.Body = http.MaxBytesReader(w, r.Body, int64(s.lim.MaxBytes)+multipartSlack)' '	_ = multipartSlack'
mutate "web giữ cả đường dẫn trong tên file" cmd/flowcastd/server.go '	name := path.Base(strings.ReplaceAll(hdr.Filename, "\\", "/"))' '	name := hdr.Filename'
mutate "web không bao giờ báo bận" cmd/flowcastd/server.go '	case <-t.C:
	case <-ctx.Done():
	}
	return false' '	case <-t.C:
	case <-ctx.Done():
	}
	return true'
mutate "web bảng lỗi vẫn trả 200" cmd/flowcastd/server.go '		status = http.StatusUnprocessableEntity
	}
	if build && resp.OK {' '	}
	if build && resp.OK {'
mutate "web vượt giới hạn trả 422" cmd/flowcastd/server.go '	case strings.HasPrefix(code, "limit."):
		return http.StatusRequestEntityTooLarge' '	case false:
		return http.StatusRequestEntityTooLarge'
mutate "web bỏ header nosniff" cmd/flowcastd/server.go '		w.Header().Set("X-Content-Type-Options", "nosniff")' ''
mutate "web không truyền tùy chọn đọc" cmd/flowcastd/server.go '			src.Options[k] = v' '			_ = v'
mutate "web không dùng giới hạn" cmd/flowcastd/server.go 'Config: &cfg, Limits: &s.lim,' 'Config: &cfg,'

# flowchart: bảng không có lane
mutate "bảng không có lane vẫn đòi parent" validate/validate.go '			if r.Parent == "" && len(v.lanes) == 0 {' '			if false {'
mutate "flowchart giữ header của pool và lane" layout/model.go '		l.Cfg.PoolHeader, l.Cfg.LaneHeader = 0, 0' '		_ = l.Cfg'
mutate "flowchart vẫn vẽ pool" writer/drawio/write.go '	if !r.NoLanes {
		writePool' '	if true {
		writePool'
mutate "flowchart không cộng gốc vào node" writer/drawio/write.go '			geo(c, it.X+ox, it.Y+oy, it.W, it.H)' '			geo(c, it.X, it.Y, it.W, it.H)'
mutate "flowchart không cộng gốc vào điểm gấp" writer/drawio/write.go '		edgeParent, dx, dy = "1", ox, oy' '		edgeParent = "1"'
mutate "merge flowchart so parent với lane ẩn" merge/merge.go '	if m.r.NoLanes {
		return "1"
	}' ''

# source/mermaid
mutate "mermaid không nhận hình start/end" source/mermaid.go '	{"([", "])", "stadium"},' ''
mutate "mermaid đọc (( thành (" source/mermaid.go '	{"((", "))", "circle"},' ''
mutate "mermaid bỏ nét đứt" source/mermaid.go '		lk.edge.dashed = true' '		_ = lk'
mutate "mermaid bỏ nét đậm" source/mermaid.go "		lk.edge.bold = ch == '='" '		_ = ch'
mutate "mermaid mũi tên mở vẫn có đầu" source/mermaid.go '		lk.edge.noarrow = true' '		_ = lk'
mutate "mermaid bỏ nhãn |...|" source/mermaid.go '	lk.edge.text = strings.TrimSpace(c.rest()[1 : end+1])' '	_ = end'
mutate "mermaid không nhận nhãn giữa mũi tên" source/mermaid.go '		if n >= 3 || h != "" {' '		if true {'
mutate "mermaid id không nhận gạch giữa và dấu chấm" source/mermaid.go "		if (r == '-' || r == '.') && c.i > start && c.i+1 < len(c.s) {" '		if false {'
mutate "mermaid id không nhận dấu chấm" source/mermaid.go "if (r == '-' || r == '.') &&" "if (r == '-') &&"
mutate "mermaid không đọc &" source/mermaid.go "		if c.i < len(c.s) && c.s[c.i] == '&' {" '		if false {'
mutate "mermaid không đánh back" source/mermaid.go '			if pos[e.to] <= pos[e.from] {' '			if false {'
mutate "mermaid hợp nhánh không chờ nguồn" source/mermaid.go '			if !back[e] && e.from != id && !written[e.from] {' '			if false {'
mutate "mermaid db không đứng cạnh node" source/mermaid.go '		attach[n.id] = peer' '		_ = peer'
mutate "mermaid condition một nhánh vẫn là condition" source/mermaid.go '		if types[id] == "condition" && len(outs[id]) < 2 {' '		if false {'
mutate "mermaid node ngoài subgraph không có lane" source/mermaid.go '		out[id] = loose' '		_ = id'
mutate "mermaid subgraph lồng thành lane riêng" source/mermaid.go '	if len(m.stack) > 0 {
		m.warn(ln, "mermaid.nested_subgraph",' '	if false {
		m.warn(ln, "mermaid.nested_subgraph",'
mutate "mermaid không đổi id dành riêng" source/mermaid.go '		if id == "0" || id == "1" || id == "pool" {' '		if false {'
mutate "mermaid cạnh trùng không có hậu tố" source/mermaid.go '		if seen[base] > 1 {' '		if false {'
mutate "mermaid không giải mã #quot;" source/mermaid.go '				if v, ok := entities[code]; ok {' '				if v, ok := entities[""]; ok {'
mutate "mermaid không tách dòng br" source/mermaid.go '	parts := splitBR(text)' '	parts := []string{text}'
mutate "mermaid bỏ màu nhấn" source/mermaid.go '	if props["fill"] == highlightFill {' '	if false {'
mutate "mermaid không đọc tiêu đề" source/mermaid.go '			m.title = normalize(unquote(strings.TrimSpace(v)))' '			_ = v'
mutate "mermaid không đọc khối trong markdown" source/source.go '		if block, ok := mermaidBlock(s.Data); ok && isCode(err, "source.no_header") {' '		if block, ok := mermaidBlock(s.Data); false && ok {'
mutate "mermaid cảnh báo không theo số dòng" source/mermaid.go '	sort.SliceStable(m.issues, func(i, j int) bool { return m.issues[i].Loc.Line < m.issues[j].Loc.Line })' ''
mutate "cạnh không nhận nét đậm" layout/model.go '			Bold: hasStyle(r, "bold"), NoArrow: hasStyle(r, "noarrow"),' '			NoArrow: hasStyle(r, "noarrow"),'
mutate "nét đậm viết trước màu nhấn" writer/drawio/write.go '			style += "strokeWidth=3;"' '			style += ""'

# heuristic nhánh chính cho sơ đồ không có lane
mutate "nhánh chính luôn là cạnh sau cùng" layout/branch.go '	last := len(same) - 1
	if !l.NoLanes {' '	last := len(same) - 1
	if true {'
mutate "hòa độ sâu lấy cạnh viết trước" layout/branch.go '		if d := l.branchDepth(same[i].Dst); d > bd {' '		if d := l.branchDepth(same[i].Dst); d >= bd {'
mutate "độ sâu tính cả luồng chung sau hợp nhánh" layout/branch.go '	if l.nonBackIn(id) <= 1 {' '	if true {'
mutate "độ sâu đi cả cạnh vòng lặp" layout/branch.go '			if !x.Back {
				if k := 1 + l.branchDepth(x.Dst); k > d {' '			if true {
				if k := 1 + l.branchDepth(x.Dst); k > d {'
mutate "hợp nhánh ngoài xương sống vẫn về cột chính" layout/place.go 'ok && (!l.NoLanes || spine[vid]) {' 'ok {'
mutate "xương sống chỉ đi một bước" layout/branch.go '			id = same[l.mainEdge(same)].Dst
		}' '			_ = same
			break
		}'

# hướng
mutate "hướng LR không hoán đổi kích thước node" layout/axis.go '		it.W, it.H = it.H, it.W' '		_ = it'
mutate "hướng LR không hoán đổi kích thước nhãn" layout/axis.go '		e.LW, e.LH = e.LH, e.LW' '		_ = e'
mutate "lật BT lật cả phần header" layout/axis.go '		return [2]float64{x, o.hdr + o.end - y}' '		return [2]float64{x, o.end - y}'
mutate "RL không lật sau khi đổi trục" layout/axis.go '		return [2]float64{o.hdr + o.end - y, x}' '		return [2]float64{y, x}'
mutate "đổi trục không đổi điểm neo" layout/axis.go '	case DirLR:
		return [2]float64{f[1], f[0]}' '	case DirLR:
		return f'
mutate "lật BT không lật điểm neo" layout/axis.go '		return [2]float64{f[0], 1 - f[1]}' '		return f'
mutate "đổi trục không đổi độ lệch nhãn" layout/axis.go '	case DirLR:
		return [2]float64{v[1], v[0]}' '	case DirLR:
		return v'
mutate "đổi trục không đổi kích thước pool" layout/axis.go '		out.PoolW, out.PoolH = r.PoolH, r.PoolW' '		_ = out'
mutate "đổi trục quên hộp nhãn" layout/axis.go '			b := o.box(*e.Label)
			e.Label = &b' ''
mutate "lane ngang vẽ như lane dọc" writer/drawio/write.go 'func horizontalLanes(r layout.Result) bool { return r.Dir == layout.DirLR || r.Dir == layout.DirRL }' 'func horizontalLanes(r layout.Result) bool { return false }'
mutate "phần tử trong lane ngang lấy toạ độ của lane dọc" writer/drawio/write.go '			geo(c, it.X-float64(r.PoolHeader), it.Y-r.LaneX[it.Lane], it.W, it.H)' '			geo(c, it.X-r.LaneX[it.Lane], it.Y-float64(r.PoolHeader), it.W, it.H)'
mutate "merge nhận cả hướng khác TD" build.go '	if opt.Previous != nil && dir != layout.DirTD {' '	if false {'
mutate "hướng lạ không bị từ chối" build.go '	if !layout.ValidDirection(dir) {' '	if false {'
mutate "hướng của nguồn bị bỏ qua" build.go '		dir = t.Direction' '		_ = t.Direction'

# khai báo trường cấu hình
mutate "mặc định không lấy từ khai báo trường" layout/config.go '		*f.Get(&c) = f.Default' '		_ = f'
mutate "không kiểm miền giá trị" layout/config.go '		if v := *f.Get(&c); v < f.Lo || v > f.Hi {' '		if false {'
mutate "miền giá trị bỏ chặn trên" layout/config.go '		if v := *f.Get(&c); v < f.Lo || v > f.Hi {' '		if v := *f.Get(&c); v < f.Lo {'
mutate "Fields trả về chính slice bên trong" layout/config.go '{ return append([]Field(nil), fields...) }' '{ return fields }'
mutate "web bỏ qua tham số xếp hình" cmd/flowcastd/server.go '		*f.Get(&cfg) = n' '		_ = n'
mutate "web bỏ qua hướng" cmd/flowcastd/server.go '		Direction: strings.TrimSpace(r.FormValue("direction"))}' '	}'
mutate "web nhận tham số không phải số" cmd/flowcastd/server.go '		n, err := strconv.Atoi(v)
		if err != nil {' '		n, _ := strconv.Atoi(v)
		if false {'
mutate "web trả 422 cho cấu hình sai" cmd/flowcastd/server.go '	case code == "schema.out_of_range" || code == "config.direction" || code == "merge.direction":' '	case false:'
mutate "web không khai báo hướng" cmd/flowcastd/server.go '		"directions": []string{layout.DirTD, layout.DirBT, layout.DirLR, layout.DirRL},' '		"directions": []string{},'
mutate "web khai báo thiếu miền giá trị" cmd/flowcastd/server.go '		out = append(out, apiField{f.Name, f.Help, f.Default, f.Lo, f.Hi})' '		out = append(out, apiField{f.Name, f.Help, f.Default, 0, 0})'

mutate "help không liệt kê tham số xếp hình" cmd/flowcast/args.go '	for _, f := range layout.Fields() {
		fmt.Fprintf(&b,' '	for _, f := range nil {
		fmt.Fprintf(&b,'
mutate "help chỉ nhận ở vị trí đầu" cmd/flowcast/args.go '		case name == "help":
			return nil, errHelp' '		case false:
			return nil, errHelp'

[ -n "$PREFLIGHT" ] && exit 0
# Mọi đột biến phải được khôi phục. Cây còn bẩn nghĩa là chính công cụ này đang
# hỏng, và mọi kết quả phía trên đều không đáng tin.
if ! git diff --quiet; then
	echo 'ERROR cây làm việc còn bẩn sau khi chạy:'
	git status --short
	exit 1
fi
echo
echo "bắt được $caught, bỏ lọt $missed"
[ "$missed" -eq 0 ]
