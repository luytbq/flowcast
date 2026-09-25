package source

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

// TestDecodeMatchesVectors compares decode against vectors over random bytes,
// including BOMs, bytes 0x80 to 0x9F and malformed utf-8 sequences.
func TestDecodeMatchesVectors(t *testing.T) {
	data, err := os.ReadFile("../conformance/decode-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vs []map[string]*string
	if err := json.Unmarshal(data, &vs); err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		raw, _ := hex.DecodeString(*v["hex"])
		for _, enc := range []string{"utf-8-sig", "utf-8", "cp1252", "latin-1"} {
			got, ok := decode(raw, enc)
			want := v[enc]
			if ok != (want != nil) || (ok && got != *want) {
				t.Errorf("%s as %s: got %q ok=%v, vector %v", *v["hex"], enc, got, ok, want)
			}
		}
	}
	t.Logf("compared %d byte sequences across four encodings", len(vs))
}
