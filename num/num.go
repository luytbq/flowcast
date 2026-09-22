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
	// Toạ độ đọc từ file .drawio cũ có thể là inf hay nan; Python in chúng bằng
	// chữ thường và không có dấu cộng.
	switch {
	case math.IsNaN(v):
		return "nan"
	case math.IsInf(v, 1):
		return "inf"
	case math.IsInf(v, -1):
		return "-inf"
	}
	s := strconv.FormatFloat(v, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}

// Rnd làm tròn lên tới số nguyên chẵn gần nhất. Mọi bề rộng và bề cao của phần
// tử đều đi qua đây, nên toạ độ luôn rơi vào lưới chẵn.
func Rnd(v float64) int {
	return int(math.Ceil(v/2.0) * 2)
}

// Round làm tròn tới d chữ số thập phân, đúng như round(x, d) của Python.
//
// Không dùng math.Round: nó làm tròn nửa ra xa số không trên giá trị đã nhân
// lên, còn Python làm tròn giá trị nhị phân thật về số chẵn gần nhất, nên hai
// cách cho kết quả khác nhau ở đúng những số như 2.675. Đi qua chuỗi thập phân
// thì khớp, vì cả hai phía đều dùng cùng một cách chuyển số thực sang chuỗi được
// làm tròn đúng. Đã kiểm trên conformance/round-vectors.json.
func Round(x float64, d int) float64 {
	v, err := strconv.ParseFloat(strconv.FormatFloat(x, 'f', d, 64), 64)
	if err != nil {
		panic(err)
	}
	return v
}
