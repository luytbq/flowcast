package layout

import "fmt"

// Res là một tài nguyên chứa dây: kênh ngang nằm giữa hai hàng, hoặc máng dọc
// nằm giữa hai cột. Nhiều đoạn dây chia nhau một Res bằng cách nằm ở các track
// khác nhau.
type Res struct {
	Kind byte // 'C' là kênh, 'G' là máng
	// Với kênh, A là hàng mà kênh nằm ngay phía trên. Với máng, A là lane và B
	// là chỉ số máng trong lane đó.
	A, B int
}

func (r Res) String() string {
	if r.Kind == 'C' {
		return fmt.Sprintf("C:%d", r.A)
	}
	return fmt.Sprintf("G:%d:%d", r.A, r.B)
}

// label gọi tên Res cho người đọc cảnh báo.
func (r Res) label() string {
	if r.Kind == 'C' {
		return fmt.Sprintf("kênh %d", r.A)
	}
	return fmt.Sprintf("máng %d của lane %d", r.B, r.A)
}

// Stub là chỗ một đoạn dây nối vào node hoặc vào đoạn khác, nhìn từ bên trong
// Res: Pos là vị trí dọc theo Res, Dir là phía nối vào, -1 từ phía thấp và 1
// từ phía cao.
type Stub struct{ Pos, Dir int }

// Seg là một đoạn dây nằm trong một Res.
type Seg struct {
	Res    Res
	Lo, Hi int
	// Key là id của node đích khi đoạn này dùng chung được với các đoạn khác
	// cùng đích, để chúng gộp thành một đường. Rỗng nghĩa là đoạn riêng.
	Key   string
	Track int
	Stubs []Stub
}

// RefKind là nguồn của một toạ độ trong Sym.
type RefKind byte

const (
	RefSrc RefKind = 's' // lấy từ cổng ra của node nguồn
	RefCol RefKind = 'c' // lấy từ tâm một cột
	RefSeg RefKind = 'g' // lấy từ track của một đoạn dây
)

type Ref struct {
	Kind      RefKind
	Lane, Col int
	Seg       *Seg
}

// SymPair nói điểm gấp thứ i lấy x từ đâu và y từ đâu. Pha hình học dựng toạ
// độ thật từ đây sau khi đã biết vị trí pixel của cột và track.
type SymPair struct{ X, Y Ref }

type sideKey struct {
	id   string
	side byte
}

func opp(s byte) byte {
	if s == 'R' {
		return 'L'
	}
	return 'R'
}

// sideFrac là điểm giữa của từng mặt, tính theo phần của bề rộng và bề cao.
var sideFrac = map[byte][2]float64{
	'R': {1.0, 0.5},
	'L': {0.0, 0.5},
	'B': {0.5, 1.0},
	'T': {0.5, 0.0},
}
