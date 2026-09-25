package source

import (
	"github.com/luytbq/flowcast/internal/unistr"
	"path/filepath"
	"strings"

	"github.com/luytbq/flowcast/model"
)

// Source is unparsed input.
//
// Bytes rather than a path: the CLI builds a Source from a file, the web service
// from an upload, and the core does not need to know which.
type Source struct {
	Data []byte
	// Name is the display name, used as the fallback title and to guess the format.
	// It is never used to open a file.
	Name    string
	Format  string // empty means guess from Name
	Options map[string]string
	// MaxUnzipped caps the total bytes decompressed from an xlsx file, because a
	// zip file of a few KB can expand to gigabytes. 0 means no cap.
	MaxUnzipped int64
}

// Parse reads a Source into a Table.
//
// The title falls back to the source name when the source does not declare one,
// just as the reference implementation falls back to the file name.
func Parse(s Source) (model.Table, error) {
	format := s.Format
	if format == "" {
		format = guessFormat(s.Name)
	}
	var (
		t   model.Table
		err error
	)
	switch format {
	case "markdown":
		t, err = ParseMarkdown(s.Data)
		if block, ok := mermaidBlock(s.Data); ok && isCode(err, "source.no_header") {
			t, err = ParseMermaid(block, s.Name)
		}
	case "mermaid":
		t, err = ParseMermaid(s.Data, s.Name)
	case "csv":
		t, err = ParseCSV(s.Data, s.Name, s.Options["delimiter"], s.Options["encoding"])
	case "xlsx":
		t, err = parseXLSX(s.Data, s.Name, s.Options["sheet"], s.MaxUnzipped)
	case "":
		return model.Table{}, model.Errf("source.unknown_format",
			"cannot guess the format of %q; pass Format", s.Name)
	default:
		return model.Table{}, model.Errf("source.unsupported",
			"format %q is not supported", format)
	}
	if err != nil {
		return model.Table{}, err
	}
	if t.Title == "" {
		t.Title = stem(s.Name)
	}
	return t, nil
}

func guessFormat(name string) string {
	switch unistr.Lower(Ext(name)) {
	case ".md", ".markdown", ".txt":
		return "markdown"
	case ".csv", ".tsv":
		return "csv"
	case ".xlsx", ".xlsm":
		return "xlsx"
	case ".mmd", ".mermaid":
		return "mermaid"
	}
	return ""
}

// Ext returns the file extension. A leading dot in the file name does not count
// as an extension separator, so the hidden file ".md" has no extension.
func Ext(path string) string {
	base := filepath.Base(path)
	trimmed := strings.TrimLeft(base, ".")
	i := strings.LastIndex(trimmed, ".")
	if i < 0 {
		return ""
	}
	return trimmed[i:]
}

func stem(name string) string {
	base := filepath.Base(name)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func isCode(err error, code string) bool {
	e, ok := err.(*model.Error)
	return ok && e.Code == code
}

// mermaidBlock takes the first ```mermaid block of a markdown file that has no
// Flow Table, such as a README embedding a diagram. Lines outside the block are
// replaced with blank lines so line numbers in Issues are still those of the
// original file.
func mermaidBlock(data []byte) ([]byte, bool) {
	lines := strings.Split(string(data), "\n")
	start := -1
	for i, ln := range lines {
		t := strings.TrimSpace(ln)
		if start < 0 {
			if strings.HasPrefix(t, "```") && strings.TrimSpace(strings.TrimLeft(t, "`")) == "mermaid" {
				start = i
			}
			continue
		}
		if strings.HasPrefix(t, "```") {
			out := make([]string, len(lines))
			copy(out[start+1:i], lines[start+1:i])
			return []byte(strings.Join(out[:i], "\n")), true
		}
	}
	return nil, false
}
