// Package num giữ hai quy tắc chuẩn hóa số dùng chung cho mọi tầng.
//
// Cả hai đều phải khớp tuyệt đối với bản tham chiếu Python, vì chúng quyết định
// từng byte trong file .drawio sinh ra. Xem conformance/README.md.
package num

import (
	"math"
	"strconv"
	"strings"
)

// Fmt sinh dạng số dùng trong file .drawio và trong dump: hai chữ số thập phân,
// rồi cắt số 0 thừa và dấu chấm thừa.
//
// Giữ nguyên cả những góc kỳ quặc của bản tham chiếu: Fmt(-0.001) ra "-0", vì
// làm tròn cho "-0.00" rồi cắt hết số 0 còn lại dấu trừ và số 0 đầu.
func Fmt(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}

// Rnd làm tròn lên tới số nguyên chẵn gần nhất. Mọi bề rộng và bề cao của phần
// tử đều đi qua đây, nên toạ độ luôn rơi vào lưới chẵn.
func Rnd(v float64) int {
	return int(math.Ceil(v/2.0) * 2)
}
