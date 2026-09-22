package conformance

import (
	"testing"

	"github.com/luytbq/flowcast/model"
	"github.com/luytbq/flowcast/validate"
)

// TestChangIssues chốt gói schema và validate.
//
// So cả thứ tự lẫn nội dung: thứ tự phát hiện là một phần của giao diện, vì
// caller in chúng ra theo đúng thứ tự này.
func TestChangIssues(t *testing.T) {
	dumps, err := Load(Dir())
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, d := range dumps {
		t.Run(d.Name, func(t *testing.T) {
			if d.Fatal != "" {
				return
			}
			tbl, err := ParseCase(Dir(), d.Name)
			if err != nil {
				t.Fatalf("parse lỗi: %v", err)
			}
			got := append(append([]model.Issue{}, tbl.Issues...), validate.Validate(tbl.Rows)...)
			assertIssues(t, got, d.Issues)
		})
		total += len(d.Issues)
	}
	if total == 0 {
		t.Fatal("không so được phát hiện nào")
	}
	t.Logf("đã so %d phát hiện trên %d case", total, len(dumps))
}
