package source

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// TestReadCSVKhopVector so readCSV với vector trên hàng nghìn chuỗi ngẫu nhiên
// và các chuỗi dựng tay ở đúng những góc mà encoding/csv của Go hiểu khác.
func TestReadCSVKhopVector(t *testing.T) {
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
				t.Errorf("%q với dấu %q:\n  Go %q %s\n  Py %q %s", v.Text, v.Delim, got, gotErr, v.Rows, v.Error)
			}
		}
	}
	if bad > 5 {
		t.Errorf("... và %d vector lệch nữa", bad-5)
	}
	t.Logf("đã so %d vector", len(vs))
}
