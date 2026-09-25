package num

import (
	"encoding/json"
	"math"
	"os"
	"strconv"
	"testing"
)

// TestRoundMatchesVectorsBitForBit compares Round against vectors over 60,000
// values, including exact halfway numbers and random bits, and compares bit by
// bit rather than formatted strings.
func TestRoundMatchesVectorsBitForBit(t *testing.T) {
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
					t.Errorf("Round(%v, %d) = %v, vector gives %v", x, d, got, want)
				}
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d of %d roundings mismatched", bad, 2*len(vs))
	}
	t.Logf("compared %d roundings", 2*len(vs))
}

// TestMathRoundIsNotUsable keeps the reason math.Round is not used, so nobody
// "simplifies" Round back to it.
func TestMathRoundIsNotUsable(t *testing.T) {
	naive := math.Round(2.675*100) / 100
	if Round(2.675, 2) == naive {
		t.Fatalf("math.Round and Round give the same result at 2.675; Round's reason to exist is gone")
	}
}
