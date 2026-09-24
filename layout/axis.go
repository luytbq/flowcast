package layout

// Hướng của sơ đồ: chiều mà luồng đi.
const (
	DirTD = "TD" // từ trên xuống
	DirBT = "BT" // từ dưới lên
	DirLR = "LR" // từ trái sang phải
	DirRL = "RL" // từ phải sang trái
)

// ValidDirection nói dir có phải một hướng hợp lệ không.
func ValidDirection(dir string) bool {
	switch dir {
	case DirTD, DirBT, DirLR, DirRL:
		return true
	}
	return false
}

// Engine chỉ biết một hướng: từ trên xuống. Hướng khác được dựng bằng cách xếp
// trong một không gian TD ảo rồi đổi trục ở cuối. Với LR và RL, bề rộng và bề
// cao của mọi phần tử và nhãn được hoán đổi trước khi xếp, để hàng của không
// gian ảo có đúng độ dày của cột thật. Chữ vẫn được ngắt dòng theo chiều thật.
//
// Nhờ vậy place, route, geometry và labels không có nhánh code nào cho từng
// hướng, và sơ đồ TD không bị ảnh hưởng khi thêm hướng. Tự kiểm chạy trên kết
// quả ảo: đổi trục và lật giữ nguyên mọi quan hệ chồng lấn và khoảng cách.

func transposed(dir string) bool { return dir == DirLR || dir == DirRL }

// SetDirection chọn hướng. Gọi sau New và trước Run.
func (l *Layout) SetDirection(dir string) {
	l.Dir = dir
	if !transposed(dir) {
		return
	}
	for _, it := range l.ItemOrder {
		it.W, it.H = it.H, it.W
	}
	for _, e := range l.Edges {
		e.LW, e.LH = e.LH, e.LW
	}
}

// orient là phép biến đổi từ không gian ảo sang hướng thật.
type orient struct {
	dir string
	// Luồng lật trong vùng nội dung [hdr, end]: header của pool và lane vẫn ở
	// đầu băng, chỉ nội dung đổi chiều.
	hdr, end float64
}

func (o orient) pt(p [2]float64) [2]float64 {
	x, y := p[0], p[1]
	switch o.dir {
	case DirBT:
		return [2]float64{x, o.hdr + o.end - y}
	case DirLR:
		return [2]float64{y, x}
	case DirRL:
		return [2]float64{o.hdr + o.end - y, x}
	}
	return p
}

func (o orient) box(b [4]float64) [4]float64 {
	a, c := o.pt([2]float64{b[0], b[1]}), o.pt([2]float64{b[2], b[3]})
	return [4]float64{fmin(a[0], c[0]), fmin(a[1], c[1]), fmax(a[0], c[0]), fmax(a[1], c[1])}
}

// frac đổi điểm neo dạng tỉ lệ trên cạnh của một hộp.
func (o orient) frac(f [2]float64) [2]float64 {
	switch o.dir {
	case DirBT:
		return [2]float64{f[0], 1 - f[1]}
	case DirLR:
		return [2]float64{f[1], f[0]}
	case DirRL:
		return [2]float64{1 - f[1], f[0]}
	}
	return f
}

// vec đổi một độ lệch, như độ lệch của nhãn so với điểm neo trên đường.
func (o orient) vec(v [2]float64) [2]float64 {
	switch o.dir {
	case DirBT:
		return [2]float64{v[0], -v[1]}
	case DirLR:
		return [2]float64{v[1], v[0]}
	case DirRL:
		return [2]float64{-v[1], v[0]}
	}
	return v
}

// Orient đưa kết quả từ không gian ảo về hướng thật. Kết quả trả về không dùng
// chung vùng nhớ với r. Với TD nó trả nguyên r.
func Orient(r Result) Result {
	if r.Dir == "" || r.Dir == DirTD {
		return r
	}
	o := orient{dir: r.Dir, hdr: float64(r.PoolHeader + r.LaneHeader), end: r.PoolH}
	out := r
	if transposed(r.Dir) {
		out.PoolW, out.PoolH = r.PoolH, r.PoolW
	}
	out.Items = make([]PlacedItem, len(r.Items))
	for i, it := range r.Items {
		b := o.box([4]float64{it.X, it.Y, it.X + it.W, it.Y + it.H})
		it.X, it.Y, it.W, it.H = b[0], b[1], b[2]-b[0], b[3]-b[1]
		out.Items[i] = it
	}
	out.Edges = make([]PlacedEdge, len(r.Edges))
	for i, e := range r.Edges {
		pts := make([][2]float64, len(e.Pts))
		for k, p := range e.Pts {
			pts[k] = o.pt(p)
		}
		e.Pts = pts
		e.ExitFrac, e.EntryFrac = o.frac(e.ExitFrac), o.frac(e.EntryFrac)
		if e.Label != nil {
			b := o.box(*e.Label)
			e.Label = &b
		}
		e.LabelOff = o.vec(e.LabelOff)
		out.Edges[i] = e
	}
	return out
}
