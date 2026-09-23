package source

import "testing"

// Ext theo đúng os.path.splitext của Python: dấu chấm ở đầu tên file không phải
// dấu tách đuôi, nên ".md" là một file không có đuôi chứ không phải file
// markdown.
func TestExtGiongSplitextCuaPython(t *testing.T) {
	cases := map[string]string{
		"a.md":          ".md",
		"so.do.md":      ".md",
		".md":           "",
		"..md":          "",
		".hidden.csv":   ".csv",
		"khong-duoi":    "",
		"a.":            ".",
		"thu/muc/a.CSV": ".CSV",
		"thu.muc/a":     "",
		"":              "",
	}
	for in, want := range cases {
		if got := Ext(in); got != want {
			t.Errorf("Ext(%q) = %q, mong %q", in, got, want)
		}
	}
}

// guessFormat chọn bộ đọc theo đuôi file, không phân biệt hoa thường.
func TestGuessFormatTheoDuoiFile(t *testing.T) {
	cases := map[string]string{
		"a.md": "markdown", "a.MD": "markdown", "a.markdown": "markdown", "a.txt": "markdown",
		"a.csv": "csv", "a.tsv": "csv", "a.xlsx": "xlsx", "a.xlsm": "xlsx",
		"a.mmd": "mermaid", "a.mermaid": "mermaid",
		".md": "", "a.dat": "", "a": "",
	}
	for in, want := range cases {
		if got := guessFormat(in); got != want {
			t.Errorf("guessFormat(%q) = %q, mong %q", in, got, want)
		}
	}
}
