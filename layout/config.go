// Package layout xếp chỗ và đi dây cho sơ đồ activity-swimlane.
package layout

// Config là tham số xếp hình của kind này. Mọi giá trị tính bằng điểm ảnh.
//
// Đây là cấu hình của kind, không phải của core: mọi trường ở đây đều giả định
// một sơ đồ có lane, có task và có hình thoi rẽ nhánh. Cấu hình không phụ thuộc
// kind nằm ở CoreConfig. Xem mục 6 của docs/core-design.md.
//
// Khai báo miền giá trị để CLI sinh cờ và web sinh form từ cùng một nguồn sẽ
// thêm ở bước 5 của lộ trình port.
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

// DefaultConfig trả về đúng bộ mặc định của bản tham chiếu.
func DefaultConfig() Config {
	return Config{
		TaskMinW:      120,
		TaskMaxW:      240,
		CondWrap:      150,
		TermWrap:      170,
		DBWrap:        130,
		TextWrap:      260,
		LabelWrap:     180,
		TrackGap:      12,
		GutterMargin:  15,
		ChannelMargin: 12,
		MinGutter:     24,
		MinChannel:    30,
		AttachGap:     40,
		LaneHeader:    30,
		PoolHeader:    30,
		MinLaneW:      120,
		LabelPad:      4,
	}
}
