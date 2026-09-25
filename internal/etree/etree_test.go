package etree

import (
	"encoding/json"
	"os"
	"testing"
)

// TestReadWriteMatchesVectors compares reading then writing back, and reading
// then indenting then writing back, against vectors over randomly generated XML
// documents.
func TestReadWriteMatchesVectors(t *testing.T) {
	data, err := os.ReadFile("../../conformance/etree-vectors.json")
	if err != nil {
		t.Fatalf("%v", err)
	}
	var vs []struct{ Doc, Plain, Indented string }
	if err := json.Unmarshal(data, &vs); err != nil {
		t.Fatal(err)
	}
	bad := 0
	for _, v := range vs {
		e, err := Parse([]byte(v.Doc))
		if err != nil {
			t.Fatalf("read error %q: %v", v.Doc, err)
		}
		got := e.String()
		Indent(e)
		ind := e.String()
		if got != v.Plain || ind != v.Indented {
			bad++
			if bad <= 3 {
				t.Errorf("doc %q\n  plain    Go %q\n           Py %q\n  indented Go %q\n           Py %q",
					v.Doc, got, v.Plain, ind, v.Indented)
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d of %d vectors mismatched", bad, len(vs))
	}
	t.Logf("compared %d documents", len(vs))
}
