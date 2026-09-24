package flowcast

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/luytbq/flowcast/model"
)

func table(rows ...string) Source {
	s := "# t\n\n| id | type | parent | content | metadata |\n|---|---|---|---|---|\n" + strings.Join(rows, "\n") + "\n"
	return Source{Name: "t.md", Data: []byte(s)}
}

func TestLoiNoiBoCuaEngineThanhLoiCoMaKhongPanic(t *testing.T) {
	rows := []string{"| A | lane | | A | |", "| A-1 | task | A | Việc | |"}
	for i := 0; i < 60; i++ {
		rows = append(rows, fmt.Sprintf("| A-1.%d | text | A | ghi chú %d | attach=A-1 |", i, i))
	}
	_, err := Build(table(rows...), Options{})
	var me *model.Error
	if !errors.As(err, &me) || me.Code != "layout.internal" {
		t.Fatalf("muốn lỗi layout.internal, được %v", err)
	}
}

func TestCanhBaoVongLapChuaDanhDauCoMa(t *testing.T) {
	r, err := Build(table(
		"| A | lane | | A | |",
		"| A-1 | start | A | Vào | |",
		"| E1 | edge | | | from=A-1; to=A-2 |",
		"| A-2 | task | A | Hai | |",
		"| E2 | edge | | | from=A-2; to=A-3 |",
		"| A-3 | task | A | Ba | |",
		"| E3 | edge | | | from=A-3; to=A-2 |",
	), Options{})
	if err != nil || r.HasErrors() {
		t.Fatalf("bảng phải dựng được: %v %v", err, r.Issues)
	}
	for _, w := range r.Warnings {
		if w.Code == "layout.unmarked-cycle" && strings.Contains(w.Msg, "A-2") {
			return
		}
	}
	t.Fatalf("thiếu cảnh báo layout.unmarked-cycle: %+v", r.Warnings)
}
