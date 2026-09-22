package source

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

// TestDecodeKhopBytesDecodeCuaPython so decode với bytes.decode(errors="strict")
// của Python trên bytes ngẫu nhiên, gồm cả BOM, byte 0x80 tới 0x9F và dãy utf-8
// hỏng.
func TestDecodeKhopBytesDecodeCuaPython(t *testing.T) {
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
				t.Errorf("%s theo %s: Go %q ok=%v, Python %v", *v["hex"], enc, got, ok, want)
			}
		}
	}
	t.Logf("đã so %d chuỗi byte trên bốn bảng mã", len(vs))
}
