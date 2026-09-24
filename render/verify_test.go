package render

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/luytbq/flowcast"
)

// TestVerifyKhopBanThamChieu so kiểm render với verify_svg của bản tham chiếu
// trên SVG thật do drawio xuất, nguyên vẹn và bị làm lệch từng kiểu.
func TestVerifyKhopBanThamChieu(t *testing.T) {
	data, err := os.ReadFile("../conformance/verify-vectors.json")
	if err != nil {
		t.Fatalf("%v; chạy tools/verify_vectors.py", err)
	}
	var vs []struct {
		Case, Variant, SVG string
		Problems           []string
	}
	if err := json.Unmarshal(data, &vs); err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		src, err := os.ReadFile("../conformance/cases/" + v.Case + ".md")
		if err != nil {
			t.Fatal(err)
		}
		r, err := flowcast.Build(flowcast.Source{Name: v.Case + ".md", Data: src}, flowcast.Options{})
		if err != nil || r.Layout == nil {
			t.Fatalf("%s: không dựng được: %v", v.Case, err)
		}
		got, err := Verify(*r.Layout, []byte(v.SVG))
		if err != nil {
			t.Fatalf("%s/%s: %v", v.Case, v.Variant, err)
		}
		if len(got) == 0 && len(v.Problems) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, v.Problems) {
			t.Errorf("%s/%s:\n  Go %q\n  Py %q", v.Case, v.Variant, got, v.Problems)
		}
	}
}
