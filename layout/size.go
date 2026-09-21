package layout

import (
	"strings"

	"github.com/luytbq/flowcast/num"
	"github.com/luytbq/flowcast/text"
)

// WrapBudget là bề rộng ngắt dòng của một loại phần tử.
//
// Tách ra thành hàm có tên vì hành vi không chốt được nó: Wrap thu hẹp về bề
// rộng nhỏ nhất vẫn giữ nguyên số dòng, nên ngân sách sai vài pixel chỉ lộ ra
// khi có ranh giới ký tự rơi đúng vào khoảng lệch đó. Chốt thẳng con số thì
// không phụ thuộc may rủi.
//
// Loại không nhận ra dùng ngân sách của ghi chú, đúng như bản tham chiếu.
func WrapBudget(cfg Config, kind string) float64 {
	switch kind {
	case "task":
		// Trừ đi lề trái phải của hộp task.
		return float64(cfg.TaskMaxW - 32)
	case "condition":
		return float64(cfg.CondWrap)
	case "start", "end", "external":
		return float64(cfg.TermWrap)
	case "db":
		return float64(cfg.DBWrap)
	}
	return float64(cfg.TextWrap)
}

// SizeItem ngắt dòng nội dung một phần tử rồi trả về kích thước hộp của nó.
//
// Phụ thuộc duy nhất vào loại phần tử, nội dung và cấu hình, không phụ thuộc
// chỗ đứng của phần tử trong sơ đồ. Vì vậy nó chạy trước mọi bước xếp chỗ.
func SizeItem(tm *text.Measure, cfg Config, kind string, lines []string) (wrapped []string, w, h float64) {
	switch kind {
	case "task":
		wl := tm.Wrap(lines, WrapBudget(cfg, kind))
		tw, th := tm.Box(wl)
		return wl, float64(num.Rnd(max(float64(cfg.TaskMinW), tw+32))), float64(num.Rnd(max(50, th+20)))

	case "condition":
		wl := tm.Wrap(lines, WrapBudget(cfg, kind))
		tw, th := tm.Box(wl)
		// Chữ nằm lọt hình thoi khi tw/W + th/H <= 1.
		return wl, float64(num.Rnd(max(100, tw/0.6+10))), float64(num.Rnd(max(60, th/0.4+10)))

	case "start", "end", "external":
		wl := tm.Wrap(lines, WrapBudget(cfg, kind))
		tw, th := tm.Box(wl)
		// end có thêm viền ngoài nên cần rộng hơn cùng một nội dung.
		pad := 0.0
		if kind == "end" {
			pad = 12
		}
		return wl, float64(num.Rnd(max(80, tw*1.42+24+pad))), float64(num.Rnd(max(44, th*1.42+16+pad)))

	case "db":
		wl := tm.Wrap(lines, WrapBudget(cfg, kind))
		tw, th := tm.Box(wl)
		return wl, float64(num.Rnd(max(100, tw+24))), float64(num.Rnd(max(60, th+34)))
	}

	wl := tm.Wrap(lines, WrapBudget(cfg, kind))
	tw, th := tm.Box(wl)
	return wl, float64(num.Rnd(tw + 20)), float64(num.Rnd(th + 12))
}

// SizeLabel đo nhãn của một cạnh.
//
// Cạnh không có nhãn vẫn nhận kích thước, không phải số không: bề rộng bằng lề
// hai bên và bề cao bằng 2. Bản tham chiếu cộng lề ngoài nhánh kiểm chuỗi rỗng,
// nên cạnh trống ra 8 x 2 chứ không ra 0 x 0.
func SizeLabel(tm *text.Measure, cfg Config, lines []string) (wrapped []string, w, h float64) {
	var wl []string
	if HasText(lines) {
		wl = tm.Wrap(lines, float64(cfg.LabelWrap))
	}
	var lw, lh float64
	if len(wl) > 0 {
		lw, lh = tm.Box(wl)
	}
	return wl, lw + 2*float64(cfg.LabelPad), lh + 2
}

// HasText cho biết nội dung một dòng bảng có chữ thật hay không, theo đúng cách
// bản tham chiếu tính: nối các dòng bằng xuống dòng rồi cắt khoảng trắng.
func HasText(lines []string) bool {
	return strings.TrimSpace(strings.Join(lines, "\n")) != ""
}
