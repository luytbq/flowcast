package source

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// TestReadCSVMatchesVectors compares readCSV against vectors over thousands of
// random strings and hand-built strings at exactly the corners where Go's
// encoding/csv behaves differently.
func TestReadCSVMatchesVectors(t *testing.T) {
	data, err := os.ReadFile("../conformance/csv-vectors.json")
	if err != nil {
		t.Fatalf("%v", err)
	}
	var vs []struct {
		Text  string     `json:"text"`
		Delim string     `json:"delim"`
		Rows  [][]string `json:"rows"`
		Error string     `json:"error"`
	}
	if err := json.Unmarshal(data, &vs); err != nil {
		t.Fatal(err)
	}
	bad := 0
	for _, v := range vs {
		got, err := readCSV(v.Text, []rune(v.Delim)[0])
		gotErr := ""
		if err != nil {
			gotErr = err.Error()
		}
		if gotErr != v.Error || fmt.Sprintf("%q", got) != fmt.Sprintf("%q", v.Rows) && v.Error == "" {
			bad++
			if bad <= 5 {
				t.Errorf("%q with delimiter %q:\n  Go %q %s\n  Py %q %s", v.Text, v.Delim, got, gotErr, v.Rows, v.Error)
			}
		}
	}
	if bad > 5 {
		t.Errorf("... and %d more mismatched vectors", bad-5)
	}
	t.Logf("compared %d vectors", len(vs))
}
