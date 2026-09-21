// Package text đo chữ và ngắt dòng.
//
// Không đọc file font. Số đo lấy từ bảng advance width đã trích sẵn, nên cùng
// một bảng thì mọi máy đo giống nhau, binary phân phối đi không mang theo font,
// và không phụ thuộc máy đích có cài Verdana hay không.
package text

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// Metrics là bảng advance width theo từng codepoint, đơn vị là font unit.
type Metrics struct {
	Family string
	UPEM   int
	Notdef int
	adv    map[rune]int
}

type metricsFile struct {
	Family   string         `json:"family"`
	UPEM     int            `json:"upem"`
	Notdef   int            `json:"notdef"`
	Advances map[string]int `json:"advances"`
}

// LoadMetrics đọc bảng do tools/extract_metrics.py sinh ra.
func LoadMetrics(path string) (*Metrics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f metricsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if f.UPEM == 0 {
		return nil, fmt.Errorf("%s: thiếu upem", path)
	}
	m := &Metrics{Family: f.Family, UPEM: f.UPEM, Notdef: f.Notdef, adv: make(map[rune]int, len(f.Advances))}
	for k, v := range f.Advances {
		cp, err := strconv.Atoi(k)
		if err != nil {
			return nil, fmt.Errorf("%s: codepoint %q không phải số", path, k)
		}
		m.adv[rune(cp)] = v
	}
	return m, nil
}

// Width trả về bề rộng chuỗi ở cỡ chữ size, tính theo điểm ảnh.
//
// Codepoint không có trong bảng dùng advance của glyph .notdef, đúng như bản
// tham chiếu: cmap tra trượt thì ra glyph 0.
func (m *Metrics) Width(s string, size float64) float64 {
	total := 0
	for _, r := range s {
		if w, ok := m.adv[r]; ok {
			total += w
		} else {
			total += m.Notdef
		}
	}
	return float64(total) * size / float64(m.UPEM)
}
