// Command extractmetrics extracts the character width table from a TTF file into JSON.
//
// flowcast does not read font files at run time: it reads this table, embedded in the
// binary. As a result the same input table yields the same layout on every machine,
// including machines without the font installed, and the distributed binary does not
// carry the font file.
//
//	go run ./tools/extractmetrics /usr/share/fonts/truetype/msttcorefonts/Verdana.ttf data/verdana.json
//
// Only the cmap format 4 table (platform 3, encoding 1 or 10), hmtx, head and hhea
// are read; enough for every character in the Basic Multilingual Plane.
package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type metrics struct {
	upem uint16
	adv  []uint16
	cmap map[int]int
}

func u16(b []byte, off int) uint16 { return binary.BigEndian.Uint16(b[off:]) }
func u32(b []byte, off int) uint32 { return binary.BigEndian.Uint32(b[off:]) }

func parse(data []byte) (*metrics, error) {
	tables := map[string]int{}
	for i := 0; i < int(u16(data, 4)); i++ {
		rec := 12 + 16*i
		tables[string(data[rec:rec+4])] = int(u32(data, rec+8))
	}
	for _, t := range []string{"head", "hhea", "hmtx", "cmap"} {
		if _, ok := tables[t]; !ok {
			return nil, fmt.Errorf("font is missing table %s", t)
		}
	}
	m := &metrics{upem: u16(data, tables["head"]+18)}
	nhm := int(u16(data, tables["hhea"]+34))
	for i := 0; i < nhm; i++ {
		m.adv = append(m.adv, u16(data, tables["hmtx"]+4*i))
	}
	cmap, err := cmap4(data, tables["cmap"])
	if err != nil {
		return nil, err
	}
	m.cmap = cmap
	return m, nil
}

func cmap4(data []byte, base int) (map[int]int, error) {
	sub := -1
	for i := 0; i < int(u16(data, base+2)); i++ {
		rec := base + 4 + 8*i
		pid, eid, off := u16(data, rec), u16(data, rec+2), int(u32(data, rec+4))
		if pid == 3 && (eid == 1 || eid == 10) && u16(data, base+off) == 4 {
			sub = base + off
			break
		}
	}
	if sub < 0 {
		return nil, errors.New("font has no cmap format 4")
	}
	segx2 := int(u16(data, sub+6))
	seg := segx2 / 2
	endsAt, startsAt := sub+14, sub+16+segx2
	deltasAt, rangesAt := startsAt+segx2, startsAt+2*segx2
	cmap := map[int]int{}
	for k := 0; k < seg; k++ {
		start, end := int(u16(data, startsAt+2*k)), int(u16(data, endsAt+2*k))
		delta := int(int16(u16(data, deltasAt+2*k)))
		rng := int(u16(data, rangesAt+2*k))
		for c := start; c <= end; c++ {
			if c == 0xFFFF {
				continue
			}
			var g int
			if rng == 0 {
				g = (c + delta) & 0xFFFF
			} else {
				g = int(u16(data, rangesAt+2*k+rng+2*(c-start)))
				if g != 0 {
					g = (g + delta) & 0xFFFF
				}
			}
			cmap[c] = g
		}
	}
	return cmap, nil
}

// render writes JSON in exactly the shape of the embedded file: one key per line,
// no indentation, codepoints in ascending order.
func render(m *metrics, family string) string {
	cps := make([]int, 0, len(m.cmap))
	for cp := range m.cmap {
		cps = append(cps, cp)
	}
	sort.Ints(cps)
	var b strings.Builder
	fmt.Fprintf(&b, "{\n\"family\": %q,\n\"upem\": %d,\n\"notdef\": %d,\n\"advances\": {\n", family, m.upem, m.adv[0])
	for i, cp := range cps {
		g := m.cmap[cp]
		w := m.adv[len(m.adv)-1]
		if g < len(m.adv) {
			w = m.adv[g]
		}
		sep := ","
		if i == len(cps)-1 {
			sep = ""
		}
		b.WriteString("\"" + strconv.Itoa(cp) + "\": " + strconv.Itoa(int(w)) + sep + "\n")
	}
	b.WriteString("}\n}\n")
	return b.String()
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: extractmetrics FONT.ttf OUT.json")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	m, err := parse(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, os.Args[1]+": "+err.Error())
		os.Exit(1)
	}
	if err := os.WriteFile(os.Args[2], []byte(render(m, "Verdana")), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%s: %d codepoints, upem %d\n", os.Args[2], len(m.cmap), m.upem)
}
