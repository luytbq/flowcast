// Package layout xếp chỗ và đi dây cho sơ đồ activity-swimlane.
package layout

import "github.com/luytbq/flowcast/model"

// Config là tham số xếp hình của kind này. Mọi giá trị tính bằng điểm ảnh.
//
// Đây là cấu hình của kind, không phải của core: mọi trường ở đây đều giả định
// một sơ đồ có lane, có task và có hình thoi rẽ nhánh. Cấu hình không phụ thuộc
// kind nằm ở CoreConfig. Xem mục 6 của docs/core-design.md.
//
// Miền giá trị của từng trường được khai báo trong Fields, để CLI sinh cờ và
// web sinh form từ cùng một nguồn.
type Config struct {
	TaskMinW      int // bề rộng tối thiểu của hộp task
	TaskMaxW      int // bề rộng tối đa của hộp task
	CondWrap      int // bề rộng ngắt dòng trong hình thoi
	TermWrap      int // ngắt dòng start, end, external
	DBWrap        int
	TextWrap      int
	LabelWrap     int // ngắt dòng nhãn cạnh
	TrackGap      int // khoảng cách giữa hai track
	GutterMargin  int // lề từ mép máng tới track đầu
	ChannelMargin int
	MinGutter     int // khoảng trống tối thiểu giữa hai cột
	MinChannel    int // khoảng trống tối thiểu giữa hai hàng
	AttachGap     int // khoảng cách từ db hoặc text tới node nó bám
	LaneHeader    int
	PoolHeader    int
	MinLaneW      int
	LabelPad      int
}

// Field khai báo một trường cấu hình: tên, miền giá trị và lời giải thích.
//
// CLI sinh cờ từ danh sách này, web sinh form và kiểm giá trị cũng từ nó, nên
// hai bên không lệch nhau được. Miền giá trị rộng có chủ đích: người dùng được
// phép chọn cấu hình cho ra bố cục xấu, tự kiểm sẽ báo, nhưng không được phép
// chọn giá trị làm engine chạy sai như số âm.
type Field struct {
	Name    string // tên cờ của CLI, cũng là tên trường của web
	Help    string
	Default int
	Lo, Hi  int
	get     func(*Config) *int
}

// Get trả về con trỏ tới trường của c mà f khai báo.
func (f Field) Get(c *Config) *int { return f.get(c) }

var fields = []Field{
	{"task-min-w", "bề rộng tối thiểu của hộp task", 120, 20, 2000, func(c *Config) *int { return &c.TaskMinW }},
	{"task-max-w", "bề rộng tối đa của hộp task", 240, 20, 2000, func(c *Config) *int { return &c.TaskMaxW }},
	{"cond-wrap", "bề rộng ngắt dòng trong hình thoi", 150, 20, 2000, func(c *Config) *int { return &c.CondWrap }},
	{"term-wrap", "bề rộng ngắt dòng của start, end và external", 170, 20, 2000, func(c *Config) *int { return &c.TermWrap }},
	{"db-wrap", "bề rộng ngắt dòng của db", 130, 20, 2000, func(c *Config) *int { return &c.DBWrap }},
	{"text-wrap", "bề rộng ngắt dòng của ghi chú", 260, 20, 2000, func(c *Config) *int { return &c.TextWrap }},
	{"label-wrap", "bề rộng ngắt dòng của nhãn cạnh", 180, 20, 2000, func(c *Config) *int { return &c.LabelWrap }},
	{"track-gap", "khoảng cách giữa hai track dây", 12, 0, 500, func(c *Config) *int { return &c.TrackGap }},
	{"gutter-margin", "lề từ mép máng tới track đầu", 15, 0, 500, func(c *Config) *int { return &c.GutterMargin }},
	{"channel-margin", "lề từ mép kênh tới track đầu", 12, 0, 500, func(c *Config) *int { return &c.ChannelMargin }},
	{"min-gutter", "khoảng trống tối thiểu giữa hai cột", 24, 0, 2000, func(c *Config) *int { return &c.MinGutter }},
	{"min-channel", "khoảng trống tối thiểu giữa hai hàng", 30, 0, 2000, func(c *Config) *int { return &c.MinChannel }},
	{"attach-gap", "khoảng cách từ db hoặc ghi chú tới node nó bám", 40, 0, 2000, func(c *Config) *int { return &c.AttachGap }},
	{"lane-header", "bề dày header của lane", 30, 0, 500, func(c *Config) *int { return &c.LaneHeader }},
	{"pool-header", "bề dày header của pool", 30, 0, 500, func(c *Config) *int { return &c.PoolHeader }},
	{"min-lane-w", "bề dày tối thiểu của một lane", 120, 20, 4000, func(c *Config) *int { return &c.MinLaneW }},
	{"label-pad", "lề quanh chữ của nhãn cạnh", 4, 0, 100, func(c *Config) *int { return &c.LabelPad }},
}

// Fields trả về khai báo của mọi trường cấu hình, theo thứ tự trong Config.
func Fields() []Field { return append([]Field(nil), fields...) }

// DefaultConfig trả về đúng bộ mặc định của bản tham chiếu.
func DefaultConfig() Config {
	var c Config
	for _, f := range fields {
		*f.Get(&c) = f.Default
	}
	return c
}

// ValidateConfig trả về lỗi đầu tiên tìm được khi một trường nằm ngoài miền
// của nó.
func ValidateConfig(c Config) error {
	for _, f := range fields {
		if v := *f.Get(&c); v < f.Lo || v > f.Hi {
			return model.Errf("schema.out_of_range", "%s = %d nằm ngoài miền %d..%d", f.Name, v, f.Lo, f.Hi)
		}
	}
	return nil
}
