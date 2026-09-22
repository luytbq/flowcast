package source

import (
	"github.com/luytbq/flowcast/internal/pystr"
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
	case "csv":
		t, err = ParseCSV(s.Data, s.Name, s.Options["delimiter"], s.Options["encoding"])
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
	switch pystr.Lower(Ext(name)) {
	case ".md", ".markdown", ".txt":
		return "markdown"
	case ".csv", ".tsv":
		return "csv"
	case ".xlsx", ".xlsm":
		return "xlsx"
	}
	return ""
}

// Ext tách đuôi file như os.path.splitext của Python: dấu chấm ở đầu tên file
// không tính là dấu tách đuôi, nên ".md" không có đuôi.
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
