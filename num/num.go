// Package num giữ hai quy tắc chuẩn hóa số dùng chung cho mọi tầng.
//
// Chúng quyết định từng byte trong file .drawio sinh ra, nên đổi chúng là đổi
// mọi golden.
package num

import (
	"math"
	"strconv"
	"strings"
)

// Fmt sinh dạng số dùng trong file .drawio: hai chữ số thập phân, rồi cắt số 0
// thừa và dấu chấm thừa. Số âm làm tròn về không ra "0", không ra "-0".
func Fmt(v float64) string {
	// Toạ độ đọc từ file .drawio cũ có thể là inf hay nan.
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
	s = strings.TrimRight(s, ".")
	if s == "-0" {
		return "0"
	}
	return s
}

// Rnd làm tròn lên tới số nguyên chẵn gần nhất. Mọi bề rộng và bề cao của phần
// tử đều đi qua đây, nên toạ độ luôn rơi vào lưới chẵn.
func Rnd(v float64) int {
	return int(math.Ceil(v/2.0) * 2)
}

// Round làm tròn tới d chữ số thập phân, dựa trên giá trị nhị phân thật của x.
//
// Không dùng math.Round(x*100)/100 vì nó sai ở hai chỗ. Phép nhân làm tròn thêm
// một lần, nên 2.675 (thật ra là 2.67499...) thành 2.68. Và math.Round đẩy
// đúng một nửa ra xa số không, nên 0.125 thành 0.13. Đi qua chuỗi thập phân
// được làm tròn đúng thì ra 2.67 và 0.12 (nửa chính xác về số chẵn), và Round
// cùng Fmt luôn thống nhất với nhau. Đã kiểm trên conformance/round-vectors.json.
func Round(x float64, d int) float64 {
	v, err := strconv.ParseFloat(strconv.FormatFloat(x, 'f', d, 64), 64)
	if err != nil {
		panic(err)
	}
	return v
}
