package layout

import "testing"

// Orient phải đổi mọi thứ mang toạ độ: điểm của dây, hộp của phần tử, hộp của
// nhãn, điểm neo và độ lệch nhãn. Bỏ sót một thứ là sơ đồ vẫn vẽ ra được nhưng
// một phần nằm sai chỗ.
func TestOrientDoiMoiThuMangToaDo(t *testing.T) {
	label := [4]float64{10, 100, 40, 120}
	base := Result{
		PoolW: 200, PoolH: 400, PoolHeader: 30, LaneHeader: 30,
		Items: []PlacedItem{{ID: "a", X: 10, Y: 100, W: 30, H: 20}},
		Edges: []PlacedEdge{{ID: "e", Pts: [][2]float64{{20, 120}, {20, 200}},
			ExitFrac: [2]float64{0.5, 1}, EntryFrac: [2]float64{0.5, 0},
			Label: &label, LabelOff: [2]float64{5, -7}}},
	}
	cases := map[string]struct {
		item  [4]float64 // x, y, w, h
		pt0   [2]float64
		exit  [2]float64
		off   [2]float64
		poolW float64
	}{
		DirBT: {[4]float64{10, 340, 30, 20}, [2]float64{20, 340}, [2]float64{0.5, 0}, [2]float64{5, 7}, 200},
		DirLR: {[4]float64{100, 10, 20, 30}, [2]float64{120, 20}, [2]float64{1, 0.5}, [2]float64{-7, 5}, 400},
		DirRL: {[4]float64{340, 10, 20, 30}, [2]float64{340, 20}, [2]float64{0, 0.5}, [2]float64{7, 5}, 400},
	}
	for dir, want := range cases {
		r := base
		r.Dir = dir
		got := Orient(r)
		it := got.Items[0]
		if [4]float64{it.X, it.Y, it.W, it.H} != want.item {
			t.Errorf("%s: phần tử %v, mong %v", dir, [4]float64{it.X, it.Y, it.W, it.H}, want.item)
		}
		e := got.Edges[0]
		if e.Pts[0] != want.pt0 {
			t.Errorf("%s: điểm đầu %v, mong %v", dir, e.Pts[0], want.pt0)
		}
		if e.ExitFrac != want.exit {
			t.Errorf("%s: điểm neo ra %v, mong %v", dir, e.ExitFrac, want.exit)
		}
		if e.LabelOff != want.off {
			t.Errorf("%s: độ lệch nhãn %v, mong %v", dir, e.LabelOff, want.off)
		}
		if got.PoolW != want.poolW {
			t.Errorf("%s: bề rộng pool %v, mong %v", dir, got.PoolW, want.poolW)
		}
		// Hộp nhãn phải đi cùng dây, không được giữ nguyên toạ độ cũ.
		if *e.Label == label {
			t.Errorf("%s: hộp nhãn không đổi trục", dir)
		}
		b := *e.Label
		if b[0] > b[2] || b[1] > b[3] {
			t.Errorf("%s: hộp nhãn ngược %v", dir, b)
		}
		if base.Edges[0].Pts[0] != [2]float64{20, 120} || *base.Edges[0].Label != label {
			t.Errorf("%s: Orient sửa cả kết quả gốc", dir)
		}
	}
	if r := Orient(base); r.Edges[0].Pts[0] != [2]float64{20, 120} {
		t.Error("hướng rỗng phải trả nguyên kết quả")
	}
}

func TestValidateConfigChanCaHaiDau(t *testing.T) {
	c := DefaultConfig()
	if err := ValidateConfig(c); err != nil {
		t.Fatalf("cấu hình mặc định phải hợp lệ: %v", err)
	}
	for _, f := range Fields() {
		for _, v := range []int{f.Lo - 1, f.Hi + 1} {
			bad := DefaultConfig()
			*f.Get(&bad) = v
			if err := ValidateConfig(bad); err == nil {
				t.Errorf("%s = %d nằm ngoài miền %d..%d mà không bị chặn", f.Name, v, f.Lo, f.Hi)
			}
		}
	}
}

// Fields trả về bản sao: người gọi sửa danh sách không được làm hỏng cấu hình
// mặc định của tiến trình.
func TestFieldsTraVeBanSao(t *testing.T) {
	fs := Fields()
	if len(fs) == 0 {
		t.Fatal("không có trường nào")
	}
	fs[0].Default, fs[0].Name = -1, "hỏng"
	again := Fields()
	if again[0].Name == "hỏng" || again[0].Default == -1 {
		t.Error("Fields trả về chính slice bên trong")
	}
	if *again[0].Get(&Config{}) != 0 {
		t.Error("Get phải trỏ vào Config được truyền")
	}
}
