package etree

import (
	"encoding/json"
	"os"
	"testing"
)

// TestKhopElementTree so việc đọc rồi ghi lại, và đọc rồi thụt lề rồi ghi lại,
// với ElementTree của Python trên tài liệu XML sinh ngẫu nhiên.
func TestKhopElementTree(t *testing.T) {
	data, err := os.ReadFile("../../conformance/etree-vectors.json")
	if err != nil {
		t.Fatalf("%v; chạy tools/etree_vectors.py", err)
	}
	var vs []struct{ Doc, Plain, Indented string }
	if err := json.Unmarshal(data, &vs); err != nil {
		t.Fatal(err)
	}
	bad := 0
	for _, v := range vs {
		e, err := Parse([]byte(v.Doc))
		if err != nil {
			t.Fatalf("lỗi đọc %q: %v", v.Doc, err)
		}
		got := e.String()
		Indent(e)
		ind := e.String()
		if got != v.Plain || ind != v.Indented {
			bad++
			if bad <= 3 {
				t.Errorf("doc %q\n  ghi ngay Go %q\n           Py %q\n  thụt lề  Go %q\n           Py %q",
					v.Doc, got, v.Plain, ind, v.Indented)
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d trên %d vector lệch", bad, len(vs))
	}
	t.Logf("đã so %d tài liệu", len(vs))
}
