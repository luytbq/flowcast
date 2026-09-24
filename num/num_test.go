package num

import "testing"

// Các giá trị dưới đây chọn đúng chỗ làm tròn nửa chừng, nơi cách làm tròn ngây
// thơ sẽ lệch:
// 0.125 và 2.675 rơi xuống, 0.135 và 0.005 rơi lên, tuỳ giá trị nhị phân thật.
func TestFmtLamTronHaiChuSo(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{100, "100"},
		{100.5, "100.5"},
		{0.125, "0.12"},
		{0.135, "0.14"},
		{0.005, "0.01"},
		{0.015, "0.01"},
		{611.4, "611.4"},
		{63.99, "63.99"},
		{-11.5, "-11.5"},
		{-0.001, "0"},
		{1e-09, "0"},
		{2.675, "2.67"},
		{1.005, "1"},
	}
	for _, c := range cases {
		if got := Fmt(c.in); got != c.want {
			t.Errorf("Fmt(%v) = %q, cần %q", c.in, got, c.want)
		}
	}
}

func TestRndLenSoChanGanNhat(t *testing.T) {
	cases := []struct {
		in   float64
		want int
	}{{0, 0}, {1, 2}, {2, 2}, {2.1, 4}, {49.2, 50}, {50, 50}, {-1, 0}}
	for _, c := range cases {
		if got := Rnd(c.in); got != c.want {
			t.Errorf("Rnd(%v) = %d, cần %d", c.in, got, c.want)
		}
	}
}
