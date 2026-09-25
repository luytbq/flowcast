package merge

import (
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"strconv"
	"testing"
)

func loadVectors(t *testing.T) (v struct {
	Float   [][2]*string
	B64     [][3]*string
	Unquote [][2]string
	Sum     []struct {
		Xs   []string
		Want string
	}
}) {
	data, err := os.ReadFile("../conformance/merge-vectors.json")
	if err != nil {
		t.Fatalf("%v", err)
	}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestParseFloatMatchesVectors(t *testing.T) {
	for _, c := range loadVectors(t).Float {
		got, ok := parseFloat(*c[0])
		switch {
		case c[1] == nil:
			if ok {
				t.Errorf("parseFloat(%q): vector expects an error, got %v", *c[0], got)
			}
		case !ok:
			t.Errorf("parseFloat(%q): error, vector expects %s", *c[0], *c[1])
		case *c[1] == "nan":
			if !math.IsNaN(got) {
				t.Errorf("parseFloat(%q): got %v, vector expects nan", *c[0], got)
			}
		default:
			// The expected value is written in hexadecimal float syntax to keep every bit.
			want, err := strconv.ParseFloat(*c[1], 64)
			if err != nil {
				t.Fatal(err)
			}
			if math.Float64bits(got) != math.Float64bits(want) {
				t.Errorf("parseFloat(%q): got %v, vector expects %v", *c[0], got, want)
			}
		}
	}
}

func TestB64decodeMatchesVectors(t *testing.T) {
	for _, c := range loadVectors(t).B64 {
		got, err := b64decode(*c[0])
		if c[1] == nil {
			if err == nil || err.Error() != *c[2] {
				t.Errorf("b64decode(%q): got %x, %v; vector expects error %q", *c[0], got, err, *c[2])
			}
			continue
		}
		if err != nil || hex.EncodeToString(got) != *c[1] {
			t.Errorf("b64decode(%q): got %x, %v; vector expects %s", *c[0], got, err, *c[1])
		}
	}
}

func TestUnquoteMatchesVectors(t *testing.T) {
	for _, c := range loadVectors(t).Unquote {
		if got := unquote(c[0]); got != c[1] {
			t.Errorf("unquote(%q): got %q, vector expects %q", c[0], got, c[1])
		}
	}
}

func TestFsumMatchesVectors(t *testing.T) {
	for _, c := range loadVectors(t).Sum {
		var xs []float64
		for _, h := range c.Xs {
			x, err := strconv.ParseFloat(h, 64)
			if err != nil {
				t.Fatal(err)
			}
			xs = append(xs, x)
		}
		want, err := strconv.ParseFloat(c.Want, 64)
		if err != nil {
			t.Fatal(err)
		}
		if got := fsum(xs); math.Float64bits(got) != math.Float64bits(want) {
			t.Errorf("fsum(%v): got %v, vector expects %v", xs, got, want)
		}
	}
}
