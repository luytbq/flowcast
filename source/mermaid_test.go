package source

import (
	"sort"
	"strings"
	"testing"

	"github.com/luytbq/flowcast/model"
)

// rowsOf prints the table compactly as "id type parent text meta" lines for comparison.
func rowsOf(t model.Table) []string {
	var out []string
	for _, r := range t.Rows {
		var meta []string
		for _, k := range r.MetaKeys {
			meta = append(meta, k+"="+r.Meta[k])
		}
		out = append(out, strings.Join([]string{r.ID, r.Type, r.Parent, strings.Join(r.Lines, "/"),
			strings.Join(meta, ";")}, " | "))
	}
	return out
}

func codesOf(t model.Table) []string {
	var out []string
	for _, i := range t.Issues {
		out = append(out, i.Code)
	}
	sort.Strings(out)
	return out
}

func parseMM(t *testing.T, src string) model.Table {
	t.Helper()
	tb, err := ParseMermaid([]byte(src), "t.mmd")
	if err != nil {
		t.Fatal(err)
	}
	return tb
}

func expectRows(t *testing.T, got model.Table, want ...string) {
	t.Helper()
	rows := rowsOf(got)
	if strings.Join(rows, "\n") != strings.Join(want, "\n") {
		t.Errorf("rows:\n  %s\nwant:\n  %s", strings.Join(rows, "\n  "), strings.Join(want, "\n  "))
	}
}

func expectCodes(t *testing.T, got model.Table, want ...string) {
	t.Helper()
	sort.Strings(want)
	if c := codesOf(got); strings.Join(c, ",") != strings.Join(want, ",") {
		t.Errorf("warning codes %v, want %v", c, want)
	}
}

func TestMermaidNodeShapeBecomesType(t *testing.T) {
	tb := parseMM(t, `flowchart TD
  s([Bắt đầu]) --> a[Hộp] --> b(Bo góc) --> c{Rẽ?}
  c -->|x| d[[Con]]
  c -->|y| e((Luồng khác))
  d --> f(((Hết)))
  e --> g([Xong])`)
	expectRows(t, tb,
		"s | start |  | Bắt đầu | ",
		"s-->a | edge |  |  | from=s;to=a",
		"a | task |  | Hộp | ",
		"a-->b | edge |  |  | from=a;to=b",
		"b | task |  | Bo góc | ",
		"b-->c | edge |  |  | from=b;to=c",
		"c | condition |  | Rẽ? | ",
		"c-->d | edge |  | x | from=c;to=d",
		"c-->e | edge |  | y | from=c;to=e",
		"d | task |  | Con | ",
		"d-->f | edge |  |  | from=d;to=f",
		"f | end |  | Hết | ",
		"e | external |  | Luồng khác | ",
		"e-->g | edge |  |  | from=e;to=g",
		"g | end |  | Xong | ")
	expectCodes(t, tb)
}

func TestMermaidArrowStyles(t *testing.T) {
	tb := parseMM(t, `graph TD
  a --> b
  a -.-> c
  a ==> d
  a --- e
  a -. chấm .-> f
  a == đậm ==> g
  a -- chữ --> h
  a ---> i
  a -.- j
  a --o k
  a <--> l
  a ~~~ m`)
	want := []string{
		"a-->b | edge |  |  | from=a;to=b",
		"a-->c | edge |  |  | from=a;to=c;style=dashed",
		"a-->d | edge |  |  | from=a;to=d;style=bold",
		"a-->e | edge |  |  | from=a;to=e;style=noarrow",
		"a-->f | edge |  | chấm | from=a;to=f;style=dashed",
		"a-->g | edge |  | đậm | from=a;to=g;style=bold",
		"a-->h | edge |  | chữ | from=a;to=h",
		"a-->i | edge |  |  | from=a;to=i",
		"a-->j | edge |  |  | from=a;to=j;style=dashed,noarrow",
		"a-->k | edge |  |  | from=a;to=k",
		"a-->l | edge |  |  | from=a;to=l",
	}
	rows := rowsOf(tb)
	if len(rows) < len(want)+1 || strings.Join(rows[1:len(want)+1], "\n") != strings.Join(want, "\n") {
		t.Errorf("rows:\n  %s", strings.Join(rows, "\n  "))
	}
	expectCodes(t, tb, "mermaid.arrow_head", "mermaid.bidirectional", "mermaid.invisible_link")
}

// Flow Table ids look like LANE-1, and mermaid accepts such ids too, so a hyphen
// between two id characters must not be read as an arrow.
func TestMermaidIDWithHyphen(t *testing.T) {
	tb := parseMM(t, "flowchart TD\n  API-1[Nhận] --> API-2.x[Xử lý]\n  API-2.x --- SVC-3")
	expectRows(t, tb,
		"API-1 | task |  | Nhận | ",
		"API-1-->API-2.x | edge |  |  | from=API-1;to=API-2.x",
		"API-2.x | task |  | Xử lý | ",
		"API-2.x-->SVC-3 | edge |  |  | from=API-2.x;to=SVC-3;style=noarrow",
		"SVC-3 | task |  | SVC-3 | ")
}

// A diamond with only one outgoing edge is not a branch point, and the Flow Table
// requires a condition to have at least two branches, so it is drawn as a task.
func TestMermaidSingleBranchDiamondBecomesTask(t *testing.T) {
	tb := parseMM(t, "flowchart TD\n  a{Chỉ một?} --> b[Tiếp]\n  c{Hai?} -->|có| b\n  c -->|không| d[Khác]")
	rows := rowsOf(tb)
	if rows[0] != "a | task |  | Chỉ một? | " {
		t.Errorf("first row %q", rows[0])
	}
	found := false
	for _, r := range rows {
		if r == "c | condition |  | Hai? | " {
			found = true
		}
	}
	if !found {
		t.Errorf("a two-branch diamond must stay a condition:\n  %s", strings.Join(rows, "\n  "))
	}
	expectCodes(t, tb, "mermaid.condition_one_branch")
}

// Warnings must follow line numbers, because readers trace them in the source file.
func TestMermaidWarningsInLineOrder(t *testing.T) {
	tb := parseMM(t, `flowchart TD
  a --> b
  click a "x"
  c{Một} --> d
  e[(Kho)] --> f
  click f "y"`)
	var lines []int
	for _, i := range tb.Issues {
		lines = append(lines, i.Loc.Line)
	}
	for k := 1; k < len(lines); k++ {
		if lines[k] < lines[k-1] {
			t.Fatalf("warnings not in line order: %v", lines)
		}
	}
	if len(lines) < 4 {
		t.Errorf("only %d warnings: %v", len(lines), lines)
	}
}

func TestMermaidChainsAndAmpersandSources(t *testing.T) {
	tb := parseMM(t, `flowchart TD
  a & b --> c & d; c --> e`)
	expectRows(t, tb,
		"a | task |  | a | ",
		"a-->c | edge |  |  | from=a;to=c",
		"a-->d | edge |  |  | from=a;to=d",
		"b | task |  | b | ",
		"b-->c | edge |  |  | from=b;to=c",
		"b-->d | edge |  |  | from=b;to=d",
		"c | task |  | c | ",
		"c-->e | edge |  |  | from=c;to=e",
		"e | task |  | e | ",
		"d | task |  | d | ")
}

// A merge node is written only after all of its sources, and an edge returning
// to an already written node carries back=true, per the Flow Table order rule.
func TestMermaidMergeAndLoopOrder(t *testing.T) {
	tb := parseMM(t, `flowchart TD
  s([Vào]) --> c{Ok?}
  c -->|có| m[Hợp]
  c -->|không| r[Sửa]
  r --> c
  r --> m
  m --> x([Ra])`)
	expectRows(t, tb,
		"s | start |  | Vào | ",
		"s-->c | edge |  |  | from=s;to=c",
		"c | condition |  | Ok? | ",
		"c-->m | edge |  | có | from=c;to=m",
		"c-->r | edge |  | không | from=c;to=r",
		"r | task |  | Sửa | ",
		"r-->c | edge |  |  | from=r;to=c;back=true",
		"r-->m | edge |  |  | from=r;to=m",
		"m | task |  | Hợp | ",
		"m-->x | edge |  |  | from=m;to=x",
		"x | end |  | Ra | ")
}

func TestMermaidSubgraphBecomesLane(t *testing.T) {
	tb := parseMM(t, `flowchart TD
  subgraph U [Người dùng]
    a([Mở]) --> b[Gửi]
  end
  subgraph S["Máy chủ"]
    c[Nhận]
    subgraph S2
      d[Lưu]
    end
  end
  b --> c --> d --> e[Ngoài]`)
	expectRows(t, tb,
		"U | lane |  | Người dùng | ",
		"S | lane |  | Máy chủ | ",
		"_ | lane |  |  | ",
		"a | start | U | Mở | ",
		"a-->b | edge |  |  | from=a;to=b",
		"b | task | U | Gửi | ",
		"b-->c | edge |  |  | from=b;to=c",
		"c | task | S | Nhận | ",
		"c-->d | edge |  |  | from=c;to=d",
		"d | task | S | Lưu | ",
		"d-->e | edge |  |  | from=d;to=e",
		"e | task | _ | Ngoài | ")
	expectCodes(t, tb, "mermaid.nested_subgraph", "mermaid.outside_subgraph")
}

func TestMermaidDbPlacedBesideNode(t *testing.T) {
	tb := parseMM(t, `flowchart TD
  a[Ghi] -->|lưu| db[(Kho)]
  a --> b[Xong]
  x[Đọc] --> db2[(Chung)]
  y[Đọc nữa] --> db2
  y --> x`)
	rows := strings.Join(rowsOf(tb), "\n")
	for _, want := range []string{
		"a | task |  | Ghi | \ndb | db |  | Kho | attach=a\na-->b",
		"db2 | task |  | Chung | ",
	} {
		if !strings.Contains(rows, want) {
			t.Errorf("missing %q in:\n%s", want, rows)
		}
	}
	expectCodes(t, tb, "mermaid.db_attach", "mermaid.db_edge_label", "mermaid.db_shape")
}

func TestMermaidLabelText(t *testing.T) {
	tb := parseMM(t, `flowchart TD
  a["Dòng một<br/>dòng hai"] --> b["`+"`**đậm** thường`"+`"]
  b --> c[Nháy #quot;kép#quot; và #35;]
  c --> d[<b>thẻ</b> bỏ]`)
	expectRows(t, tb,
		"a | task |  | Dòng một/dòng hai | ",
		"a-->b | edge |  |  | from=a;to=b",
		"b | task |  | đậm thường | ",
		"b-->c | edge |  |  | from=b;to=c",
		`c | task |  | Nháy "kép" và # | `,
		"c-->d | edge |  |  | from=c;to=d",
		"d | task |  | thẻ bỏ | ")
}

func TestMermaidHighlightColorBecomesHighlight(t *testing.T) {
	tb := parseMM(t, `flowchart TD
  a:::hot --> b --> c
  classDef hot fill:#DAE8FC,stroke:#6c8ebf
  style c fill:#f00
  linkStyle 1 stroke:#6c8ebf`)
	expectRows(t, tb,
		"a | task |  | a | style=highlight",
		"a-->b | edge |  |  | from=a;to=b",
		"b | task |  | b | ",
		"b-->c | edge |  |  | from=b;to=c;style=highlight",
		"c | task |  | c | ")
	expectCodes(t, tb, "mermaid.style")
}

func TestMermaidRenamesReservedIDs(t *testing.T) {
	tb := parseMM(t, "flowchart TD\n  1 --> pool --> x")
	expectRows(t, tb,
		"n_1 | task |  | 1 | ",
		"n_1-->n_pool | edge |  |  | from=n_1;to=n_pool",
		"n_pool | task |  | pool | ",
		"n_pool-->x | edge |  |  | from=n_pool;to=x",
		"x | task |  | x | ")
	expectCodes(t, tb, "mermaid.renamed_id", "mermaid.renamed_id")
}

func TestMermaidDuplicateEdgeIDGetsSuffix(t *testing.T) {
	tb := parseMM(t, "flowchart TD\n  a -->|một| b\n  a -->|hai| b")
	if rows := rowsOf(tb); rows[2] != "a-->b#2 | edge |  | hai | from=a;to=b" {
		t.Errorf("rows:\n  %s", strings.Join(rows, "\n  "))
	}
}

func TestMermaidDirectionAndTitle(t *testing.T) {
	tb := parseMM(t, "---\ntitle: Tiêu đề\n---\nflowchart LR\n  a --> b")
	if tb.Title != "Tiêu đề" || tb.Direction != "LR" {
		t.Errorf("title %q, direction %q", tb.Title, tb.Direction)
	}
	expectCodes(t, tb)
	for src, want := range map[string]string{"graph TB\n a-->b": "TD", "graph\n a-->b": "", "flowchart BT\n a-->b": "BT"} {
		if d := parseMM(t, src).Direction; d != want {
			t.Errorf("%q: direction %q, want %q", src, d, want)
		}
	}
	bad := parseMM(t, "flowchart XY\n  a --> b")
	expectCodes(t, bad, "mermaid.direction")
	if bad.Issues[0].Loc.Line != 1 {
		t.Errorf("direction warning on line %d, want line 1", bad.Issues[0].Loc.Line)
	}
}

func TestMermaidNotFlowchart(t *testing.T) {
	_, err := ParseMermaid([]byte("sequenceDiagram\n  A->>B: hi"), "t.mmd")
	if !isCode(err, "mermaid.not_flowchart") {
		t.Errorf("error %v", err)
	}
}

func TestMermaidInMarkdown(t *testing.T) {
	src := "# Tài liệu\n\nVăn bản.\n\n```mermaid\nflowchart TD\n  a --> b\n  a --x c\n```\n"
	tb, err := Parse(Source{Name: "doc.md", Data: []byte(src)})
	if err != nil {
		t.Fatal(err)
	}
	if tb.Source != "mermaid" || len(tb.Rows) != 5 {
		t.Errorf("source %q, %d rows", tb.Source, len(tb.Rows))
	}
	if len(tb.Issues) != 1 || tb.Issues[0].Loc.Line != 8 {
		t.Errorf("warning must point at line 8 of the file: %+v", tb.Issues)
	}
}
