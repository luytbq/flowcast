package source

import "testing"

// A leading dot in a file name is not an extension separator, so ".md" is a file
// with no extension, not a markdown file.
func TestExtIgnoresLeadingDot(t *testing.T) {
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
			t.Errorf("Ext(%q) = %q, want %q", in, got, want)
		}
	}
}

// guessFormat picks the reader by file extension, case-insensitively.
func TestGuessFormatByExtension(t *testing.T) {
	cases := map[string]string{
		"a.md": "markdown", "a.MD": "markdown", "a.markdown": "markdown", "a.txt": "markdown",
		"a.csv": "csv", "a.tsv": "csv", "a.xlsx": "xlsx", "a.xlsm": "xlsx",
		"a.mmd": "mermaid", "a.mermaid": "mermaid",
		".md": "", "a.dat": "", "a": "",
	}
	for in, want := range cases {
		if got := guessFormat(in); got != want {
			t.Errorf("guessFormat(%q) = %q, want %q", in, got, want)
		}
	}
}
