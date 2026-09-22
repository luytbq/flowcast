package num

import (
	"encoding/json"
	"math"
	"os"
	"strconv"
	"testing"
)

// TestRoundKhopPythonTungBit so Round với round(x, d) của Python trên 60.000 giá
// trị, gồm cả số đúng nửa chừng và bit ngẫu nhiên, và so từng bit chứ không so
// chuỗi đã định dạng.
func TestRoundKhopPythonTungBit(t *testing.T) {
	data, err := os.ReadFile("../conformance/round-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vs [][3]string
	if err := json.Unmarshal(data, &vs); err != nil {
		t.Fatal(err)
	}
	parse := func(h string) float64 {
		b, err := strconv.ParseUint(h, 16, 64)
		if err != nil {
			t.Fatal(err)
		}
		return math.Float64frombits(b)
	}
	bad := 0
	for _, v := range vs {
		x := parse(v[0])
		for i, d := range []int{2, 4} {
			want := parse(v[1+i])
			if got := Round(x, d); math.Float64bits(got) != math.Float64bits(want) {
				bad++
				if bad <= 5 {
					t.Errorf("Round(%v, %d) = %v, Python ra %v", x, d, got, want)
				}
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d trên %d phép làm tròn lệch", bad, 2*len(vs))
	}
	t.Logf("đã so %d phép làm tròn", 2*len(vs))
}

// TestMathRoundKhongDungDuoc giữ lại lý do không dùng math.Round, để không ai
// "đơn giản hóa" Round về nó.
func TestMathRoundKhongDungDuoc(t *testing.T) {
	naive := math.Round(2.675*100) / 100
	if Round(2.675, 2) == naive {
		t.Fatalf("math.Round và Round cho cùng kết quả ở 2.675; lý do tồn tại của Round đã mất")
	}
}
