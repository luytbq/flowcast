// Package model giữ các kiểu dữ liệu chung của bảng đầu vào.
//
// Nằm riêng khỏi gói gốc vì mọi source adapter đều cần chúng, còn gói gốc lại
// cần các adapter. Gói gốc phơi lại bằng bí danh kiểu.
package model

// Row là một dòng bảng đã tách ô, chưa diễn giải theo ngữ nghĩa sơ đồ.
type Row struct {
	// Idx là thứ tự trong số các dòng đọc được, không phải số dòng trong file.
	// Dòng hỏng bị bỏ qua nên hai con số này lệch nhau.
	Idx    int
	Loc    Location
	ID     string
	Type   string
	Parent string
	Lines  []string
	Meta   map[string]string
	// MetaKeys là thứ tự key viết trong ô. Cần vì map của Go duyệt ngẫu nhiên,
	// còn thứ tự cảnh báo về key lạ phải theo đúng thứ tự người dùng viết.
	MetaKeys []string
}

// Text nối các dòng nội dung rồi cắt khoảng trắng, đúng cách bản tham chiếu
// dùng để quyết định một ô có chữ thật hay không.
func (r Row) Text() string { return trimSpace(joinLines(r.Lines)) }

// Table là kết quả đọc một nguồn đầu vào.
type Table struct {
	Title string
	Rows  []Row
	// Issues ở đây chỉ là phát hiện của tầng đọc. Kiểm tra theo ngữ nghĩa sơ đồ
	// nằm ở tầng validate.
	Issues []Issue
	Source string
}
