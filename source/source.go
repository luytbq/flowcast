package source

import (
	"github.com/luytbq/flowcast/internal/unistr"
	"path/filepath"
	"strings"

	"github.com/luytbq/flowcast/model"
)

// Source là đầu vào chưa phân tích.
//
// Bytes chứ không phải đường dẫn: CLI dựng Source từ file, web dựng từ upload,
// và core không cần biết bên nào.
type Source struct {
	Data []byte
	// Name là tên hiển thị, dùng làm tiêu đề dự phòng và để đoán định dạng.
	// Không bao giờ dùng để mở file.
	Name    string
	Format  string // rỗng nghĩa là tự đoán từ Name
	Options map[string]string
	// MaxUnzipped chặn tổng số byte giải nén từ một file xlsx, vì một file zip
	// vài KB có thể giải ra hàng GB. 0 là không chặn.
	MaxUnzipped int64
}

// Parse đọc một Source thành Table.
//
// Tiêu đề lùi về tên nguồn khi nguồn không tự khai báo, đúng như bản tham
// chiếu lùi về tên file.
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
			"không đoán được định dạng của %q; truyền Format", s.Name)
	default:
		return model.Table{}, model.Errf("source.unsupported",
			"định dạng %q chưa hỗ trợ", format)
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

// Ext tách đuôi file. Dấu chấm ở đầu tên file không tính là dấu tách đuôi, nên
// file ẩn ".md" không có đuôi.
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

// mermaidBlock lấy khối ```mermaid đầu tiên của một file markdown không có Flow
// Table, như README nhúng sơ đồ. Các dòng ngoài khối được thay bằng dòng trống
// để số dòng trong Issue vẫn là số dòng của file gốc.
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
