package unistr

import "testing"

func TestKhoangTrangGomCacKyTuPhanTach(t *testing.T) {
	for r := rune(0x1c); r <= 0x1f; r++ {
		if !IsSpace(r) {
			t.Errorf("U+%04X phải là khoảng trắng", r)
		}
	}
	if Strip("\x1f a \x1c") != "a" {
		t.Errorf("Strip không cắt U+001C và U+001F")
	}
}

func TestLowerChuIMangChamTrenRaHaiKyTu(t *testing.T) {
	if got := Lower("İD"); got != "i̇d" {
		t.Errorf("Lower(İD) = %q, muốn %q", got, "i̇d")
	}
}
