package flowcast

import (
	"time"

	"github.com/luytbq/flowcast/model"
	"github.com/luytbq/flowcast/source"
)

// Limits chặn tài nguyên mà một lần dựng được dùng. Giá trị 0 là không chặn.
//
// Số dòng là chặn chính, vì nó chặn luôn thời gian chạy: thời gian dựng tăng
// gần theo bình phương số phần tử, 1000 phần tử mất khoảng 0,1 giây và 3000
// phần tử mất khoảng 0,9 giây. Timeout là chặn phụ, chỉ kiểm giữa các pha chứ
// không cắt ngang một pha.
type Limits struct {
	MaxBytes    int   // kích thước đầu vào
	MaxUnzipped int64 // tổng số byte giải nén từ một file xlsx
	MaxRows     int   // số dòng bảng, gồm cả lane và cạnh
	MaxEdges    int
	Timeout     time.Duration
}

// Hai hồ sơ dựng sẵn. CLI chạy trên máy của chính người dùng nên nới; dịch vụ
// web nhận file của người lạ nên siết.
var (
	CLILimits = Limits{MaxBytes: 64 << 20, MaxUnzipped: 512 << 20, MaxRows: 50000, MaxEdges: 50000}
	WebLimits = Limits{MaxBytes: 1 << 20, MaxUnzipped: 16 << 20, MaxRows: 1000, MaxEdges: 1000,
		Timeout: 10 * time.Second}
)

// budget theo dõi một lần dựng so với Limits.
type budget struct {
	lim   Limits
	start time.Time
}

func newBudget(l *Limits) budget {
	if l == nil {
		return budget{start: time.Now()}
	}
	return budget{lim: *l, start: time.Now()}
}

func (b budget) source(src Source) (Source, error) {
	if b.lim.MaxBytes > 0 && len(src.Data) > b.lim.MaxBytes {
		return src, model.Errf("limit.bytes", "file %d byte, quá giới hạn %d byte", len(src.Data), b.lim.MaxBytes)
	}
	if b.lim.MaxUnzipped > 0 && (src.MaxUnzipped == 0 || src.MaxUnzipped > b.lim.MaxUnzipped) {
		src.MaxUnzipped = b.lim.MaxUnzipped
	}
	return src, nil
}

func (b budget) table(t model.Table) error {
	if b.lim.MaxRows > 0 && len(t.Rows) > b.lim.MaxRows {
		return model.Errf("limit.rows", "bảng có %d dòng, quá giới hạn %d dòng", len(t.Rows), b.lim.MaxRows)
	}
	edges := 0
	for _, r := range t.Rows {
		if r.Type == "edge" {
			edges++
		}
	}
	if b.lim.MaxEdges > 0 && edges > b.lim.MaxEdges {
		return model.Errf("limit.edges", "bảng có %d cạnh, quá giới hạn %d cạnh", edges, b.lim.MaxEdges)
	}
	return b.check()
}

// check báo quá thời gian. Gọi giữa các pha.
func (b budget) check() error {
	if b.lim.Timeout > 0 && time.Since(b.start) > b.lim.Timeout {
		return model.Errf("limit.timeout", "dựng quá %v, dừng", b.lim.Timeout)
	}
	return nil
}

func parseWithin(src Source, b budget) (model.Table, error) {
	src, err := b.source(src)
	if err != nil {
		return model.Table{}, err
	}
	t, err := source.Parse(src)
	if err != nil {
		return model.Table{}, err
	}
	return t, b.table(t)
}
