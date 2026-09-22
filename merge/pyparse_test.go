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
}) {
	data, err := os.ReadFile("../conformance/merge-vectors.json")
	if err != nil {
		t.Fatalf("%v; chạy tools/merge_vectors.py", err)
	}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestPyFloatKhopFloatCuaPython(t *testing.T) {
	for _, c := range loadVectors(t).Float {
		got, ok := pyFloat(*c[0])
		switch {
		case c[1] == nil:
			if ok {
				t.Errorf("float(%q): Python báo lỗi, Go ra %v", *c[0], got)
			}
		case !ok:
			t.Errorf("float(%q): Go báo lỗi, Python ra %s", *c[0], *c[1])
		case *c[1] == "nan":
			if !math.IsNaN(got) {
				t.Errorf("float(%q): Go ra %v, Python ra nan", *c[0], got)
			}
		default:
			// float.hex() của Python là cú pháp số mười sáu mà strconv đọc được.
			want, err := strconv.ParseFloat(*c[1], 64)
			if err != nil {
				t.Fatal(err)
			}
			if math.Float64bits(got) != math.Float64bits(want) {
				t.Errorf("float(%q): Go ra %v, Python ra %v", *c[0], got, want)
			}
		}
	}
}

func TestB64decodeKhopBase64CuaPython(t *testing.T) {
	for _, c := range loadVectors(t).B64 {
		got, err := b64decode(*c[0])
		if c[1] == nil {
			if err == nil || err.Error() != *c[2] {
				t.Errorf("b64decode(%q): Go ra %x, %v; Python báo %q", *c[0], got, err, *c[2])
			}
			continue
		}
		if err != nil || hex.EncodeToString(got) != *c[1] {
			t.Errorf("b64decode(%q): Go ra %x, %v; Python ra %s", *c[0], got, err, *c[1])
		}
	}
}

func TestUnquoteKhopUrllibCuaPython(t *testing.T) {
	for _, c := range loadVectors(t).Unquote {
		if got := unquote(c[0]); got != c[1] {
			t.Errorf("unquote(%q): Go ra %q, Python ra %q", c[0], got, c[1])
		}
	}
}
