package layout

import "sort"

// assignPorts đặt vị trí cổng ra trên mặt của node nguồn.
//
// Hình thoi và elip, hoặc mặt chỉ có một cạnh, luôn ra đúng giữa mặt. Hộp chữ
// nhật có nhiều cạnh ra cùng một mặt thì chia mặt đó thành nhiều cổng; cạnh A,
// B, C vẫn giữ giữa mặt, các cạnh D chia nhau phần còn lại.
func (l *Layout) assignPorts() {
	for key, es := range l.sideOut {
		u := l.items[key.id]
		side := key.side
		if len(es) == 0 {
			continue
		}
		if nonRect[u.Kind] || len(es) == 1 {
			for _, e := range es {
				e.ExitFrac = sideFrac[side]
			}
			continue
		}
		var fixed, loose []*Edge
		for _, e := range es {
			if e.Case == 'A' || e.Case == 'B' || e.Case == 'C' {
				fixed = append(fixed, e)
			} else {
				loose = append(loose, e)
			}
		}
		for _, e := range fixed {
			e.ExitFrac = sideFrac[side]
		}
		n := len(loose)
		var fr []float64
		if len(fixed) > 0 && n <= 6 {
			// Giữa mặt đã có cạnh cố định, nên các cổng còn lại né khỏi 0.5.
			fr = append(fr, []float64{0.25, 0.75, 0.125, 0.875, 0.375, 0.625}[:n]...)
			sort.Float64s(fr)
		} else {
			for i := 0; i < n; i++ {
				fr = append(fr, float64(i+1)/float64(n+1))
			}
		}
		// Dây rẽ về phía nào thì lấy cổng ở phía đó, để các đoạn đầu không cắt
		// nhau.
		sort.SliceStable(loose, func(i, j int) bool {
			a, b := l.items[loose[i].Dst], l.items[loose[j].Dst]
			if side == 'L' || side == 'R' {
				if a.Row != b.Row {
					return a.Row < b.Row
				}
				return loose[i].Order < loose[j].Order
			}
			if a.Lane != b.Lane {
				return a.Lane < b.Lane
			}
			if a.Col != b.Col {
				return a.Col < b.Col
			}
			return loose[i].Order < loose[j].Order
		})
		for i, e := range loose {
			switch side {
			case 'B':
				e.ExitFrac = [2]float64{fr[i], 1.0}
			case 'R':
				e.ExitFrac = [2]float64{1.0, fr[i]}
			default:
				e.ExitFrac = [2]float64{0.0, fr[i]}
			}
		}
	}
	for _, e := range l.Edges {
		e.EntryFrac = sideFrac[e.EntrySide]
	}
}

// assignTracks tô màu khoảng cho từng kênh và máng.
//
// Hai đoạn chồng nhau phải nằm ở hai track khác nhau, trừ khi cùng đích, vì
// khi đó chúng gộp thành một đường. Thêm một ràng buộc thứ tự: hai đoạn có
// chân nối vào cùng một vị trí từ hai phía thì đoạn phía thấp phải nằm ở track
// nhỏ hơn, nếu không hai chân sẽ đè lên nhau.
//
// Tham lam theo thứ tự đoạn được thêm vào, nên thứ tự đó là một phần của kết
// quả.
func (l *Layout) assignTracks() {
	var order []Res
	byRes := map[Res][]*Seg{}
	for _, s := range l.Segs {
		if _, ok := byRes[s.Res]; !ok {
			order = append(order, s.Res)
		}
		byRes[s.Res] = append(byRes[s.Res], s)
	}
	l.NTracks = map[Res]int{}

	for _, res := range order {
		var tracks [][]*Seg
		orderOK := func(s *Seg, ti int) bool {
			for tj, tl := range tracks {
				for _, t := range tl {
					if mustPrecede(s, t) && !(ti < tj) {
						return false
					}
					if mustPrecede(t, s) && !(tj < ti) {
						return false
					}
				}
			}
			return true
		}
		fits := func(s *Seg, ti int, busOnly bool) bool {
			tl := tracks[ti]
			if busOnly {
				shared := false
				for _, t := range tl {
					if t.Key == s.Key {
						shared = true
						break
					}
				}
				if !shared {
					return false
				}
			}
			for _, t := range tl {
				if !(s.Key != "" && t.Key == s.Key) && overlap(t, s) {
					return false
				}
			}
			return orderOK(s, ti)
		}

		for _, s := range byRes[res] {
			chosen := -1
			if s.Key != "" {
				for ti := range tracks {
					if fits(s, ti, true) {
						chosen = ti
						break
					}
				}
			}
			if chosen < 0 {
				for ti := range tracks {
					if fits(s, ti, false) {
						chosen = ti
						break
					}
				}
			}
			if chosen < 0 {
				lo, hi := 0, len(tracks)
				for tj, tl := range tracks {
					for _, t := range tl {
						if mustPrecede(t, s) && tj+1 > lo {
							lo = tj + 1
						}
						if mustPrecede(s, t) && tj < hi {
							hi = tj
						}
					}
				}
				if lo > hi {
					l.Warnings = append(l.Warnings, "không xếp được thứ tự track trong "+res.pyRepr())
					chosen = len(tracks)
				} else {
					chosen = max(lo, min(hi, len(tracks)))
				}
				tracks = append(tracks, nil)
				copy(tracks[chosen+1:], tracks[chosen:])
				tracks[chosen] = nil
			}
			tracks[chosen] = append(tracks[chosen], s)
		}
		for ti, tl := range tracks {
			for _, t := range tl {
				t.Track = ti
			}
		}
		l.NTracks[res] = len(tracks)
	}
}

func overlap(a, b *Seg) bool { return a.Lo <= b.Hi && b.Lo <= a.Hi }

// mustPrecede nói a phải nằm ở track nhỏ hơn b: cả hai có chân nối vào cùng một
// vị trí, và chân của a đến từ phía thấp hơn. Hai đoạn cùng đích thì không ràng
// buộc gì, vì chúng gộp làm một.
func mustPrecede(a, b *Seg) bool {
	if a.Key != "" && a.Key == b.Key {
		return false
	}
	for _, pa := range a.Stubs {
		for _, pb := range b.Stubs {
			if pa.Pos == pb.Pos && pa.Dir < pb.Dir {
				return true
			}
		}
	}
	return false
}
