package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/luytbq/flowcast/layout"
)

// args là đối số của một lệnh, đọc theo đúng cách argparse của bản tham chiếu
// hiểu: cờ đứng trước hay sau tên file đều được, và nhận cả "--cờ giá trị" lẫn
// "--cờ=giá trị". Gói flag chuẩn của Go dừng ở đối số vị trí đầu tiên nên không
// dùng được.
type args struct {
	cmd        string
	file       string
	output     string
	title      string
	direction  string
	layoutJSON string
	mode       string
	font       string
	png        string // rỗng: không xuất; "-" nghĩa là đường mặc định
	verify     bool
	noBackup   bool
	sheet      string
	delimiter  string
	encoding   string
	cfg        layout.Config
}

// configFlags ánh xạ cờ dòng lệnh tới trường của layout.Config. Tên cờ giữ đúng
// như bản tham chiếu, vì subagent flowtable-drawio gọi bằng chính các tên này.
func configFlags(c *layout.Config) map[string]*int {
	out := map[string]*int{}
	for _, f := range layout.Fields() {
		out[f.Name] = f.Get(c)
	}
	return out
}

func parseArgs(argv []string) (*args, error) {
	if len(argv) == 0 {
		return nil, fmt.Errorf("thiếu lệnh; dùng check hoặc build")
	}
	a := &args{cmd: argv[0], cfg: layout.DefaultConfig()}
	if a.cmd != "check" && a.cmd != "build" {
		return nil, fmt.Errorf("lệnh %q không có; dùng check hoặc build", a.cmd)
	}
	ints := configFlags(&a.cfg)
	strs := map[string]*string{"sheet": &a.sheet, "delimiter": &a.delimiter, "encoding": &a.encoding}
	if a.cmd == "build" {
		strs["output"], strs["title"], strs["layout-json"] = &a.output, &a.title, &a.layoutJSON
		strs["mode"], strs["font"] = &a.mode, &a.font
		strs["direction"] = &a.direction
	}
	rest := argv[1:]
	for i := 0; i < len(rest); i++ {
		tok := rest[i]
		if tok == "-o" {
			tok = "--output"
		}
		if !strings.HasPrefix(tok, "--") {
			if a.file != "" {
				return nil, fmt.Errorf("thừa đối số %q", tok)
			}
			a.file = tok
			continue
		}
		name, val, hasVal := strings.Cut(tok[2:], "=")
		next := func() (string, error) {
			if hasVal {
				return val, nil
			}
			if i+1 >= len(rest) {
				return "", fmt.Errorf("--%s cần một giá trị", name)
			}
			i++
			return rest[i], nil
		}
		switch {
		case a.cmd == "build" && name == "png":
			// Như nargs='?' của argparse: lấy đối số kế tiếp làm đường ra nếu
			// nó không phải một cờ, ngược lại dùng đường mặc định.
			a.png = "-"
			if hasVal {
				a.png = val
			} else if i+1 < len(rest) && !strings.HasPrefix(rest[i+1], "-") {
				i++
				a.png = rest[i]
			}
		case a.cmd == "build" && name == "verify":
			a.verify = true
		case a.cmd == "build" && name == "no-backup":
			a.noBackup = true
		case strs[name] != nil:
			v, err := next()
			if err != nil {
				return nil, err
			}
			*strs[name] = v
		case a.cmd == "build" && ints[name] != nil:
			v, err := next()
			if err != nil {
				return nil, err
			}
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("--%s cần số nguyên, nhận %q", name, v)
			}
			*ints[name] = n
		default:
			return nil, fmt.Errorf("cờ --%s không có cho lệnh %s", name, a.cmd)
		}
	}
	if a.file == "" {
		return nil, fmt.Errorf("thiếu file đầu vào")
	}
	if a.mode != "" && a.mode != "merge" && a.mode != "force" {
		return nil, fmt.Errorf("--mode chỉ nhận merge hoặc force, nhận %q", a.mode)
	}
	if a.direction != "" && !layout.ValidDirection(a.direction) {
		return nil, fmt.Errorf("--direction chỉ nhận TD, BT, LR hoặc RL, nhận %q", a.direction)
	}
	return a, nil
}
