// Package flowcast biến một sơ đồ luồng viết bằng bảng thành file draw.io có bố
// cục tất định.
//
// Build là điểm vào duy nhất. Nó không chạm filesystem, không gọi tiến trình
// ngoài và không in gì ra: bytes vào, văn bản đích cộng dữ liệu chẩn đoán ra.
// Nhờ vậy cùng một đường code phục vụ được CLI đọc file và dịch vụ web nhận
// upload. Xem docs/core-design.md.
package flowcast

import (
	"sync"

	"github.com/luytbq/flowcast/data"
	"github.com/luytbq/flowcast/layout"
	"github.com/luytbq/flowcast/model"
	"github.com/luytbq/flowcast/source"
	"github.com/luytbq/flowcast/text"
	"github.com/luytbq/flowcast/validate"
	"github.com/luytbq/flowcast/writer/drawio"
)

type (
	Source  = source.Source
	Issue   = model.Issue
	Finding = layout.Finding
)

// Options là tham số của một lần dựng.
type Options struct {
	// Config là tham số xếp hình. Giá trị rỗng nghĩa là layout.DefaultConfig().
	Config *layout.Config
	// Title thay tiêu đề lấy từ nguồn. Rỗng thì giữ tiêu đề của nguồn.
	Title string
}

// Stats là kích thước của sơ đồ đã dựng.
type Stats struct {
	Lanes, Items, Edges int
	W, H                float64
}

// Result là kết quả của một lần dựng.
type Result struct {
	Title  string
	Source string // định dạng đã đọc, ví dụ "markdown"
	// Issues là phát hiện về bảng đầu vào, của cả tầng đọc lẫn tầng kiểm tra.
	Issues []Issue
	// Text là nội dung file .drawio. Rỗng khi bảng có lỗi.
	Text string
	// Warnings là cảnh báo của engine xếp hình, Findings là kết quả tự kiểm hình
	// học. Cả hai rỗng khi bảng có lỗi.
	Warnings []string
	Findings []Finding
	Layout   *layout.Result
	Stats    Stats
}

// HasErrors nói bảng có lỗi chặn việc dựng hay không.
func (r Result) HasErrors() bool {
	for _, i := range r.Issues {
		if i.Level == model.LevelError {
			return true
		}
	}
	return false
}

var (
	metricsOnce sync.Once
	metrics     *text.Metrics
	metricsErr  error
)

func defaultMetrics() (*text.Metrics, error) {
	metricsOnce.Do(func() { metrics, metricsErr = text.ParseMetrics(data.Verdana) })
	return metrics, metricsErr
}

// Check đọc và kiểm tra bảng mà không dựng sơ đồ.
//
// Lỗi trả về là lỗi khiến việc đọc không thể tiếp tục, như không tìm thấy
// header. Lỗi của từng dòng nằm trong Result.Issues.
func Check(src Source) (Result, error) {
	t, err := source.Parse(src)
	if err != nil {
		return Result{}, err
	}
	issues := append(append([]Issue{}, t.Issues...), validate.Validate(t.Rows)...)
	return Result{Title: t.Title, Source: t.Source, Issues: issues}, nil
}

// Build đọc, kiểm tra và dựng sơ đồ.
func Build(src Source, opt Options) (Result, error) {
	t, err := source.Parse(src)
	if err != nil {
		return Result{}, err
	}
	r := Result{Title: t.Title, Source: t.Source}
	r.Issues = append(append([]Issue{}, t.Issues...), validate.Validate(t.Rows)...)
	if opt.Title != "" {
		r.Title = opt.Title
	}
	if r.HasErrors() {
		return r, nil
	}
	m, err := defaultMetrics()
	if err != nil {
		return Result{}, err
	}
	cfg := layout.DefaultConfig()
	if opt.Config != nil {
		cfg = *opt.Config
	}
	l := layout.New(t.Rows, cfg, text.NewMeasure(m)).Run()
	res := l.Result()
	r.Layout = &res
	r.Warnings = l.Warnings
	r.Findings = layout.Check(res)
	r.Text = drawio.Write(res, r.Title)
	r.Stats = Stats{len(res.Lanes), len(res.Items), len(res.Edges), res.PoolW, res.PoolH}
	return r, nil
}
