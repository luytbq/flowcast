package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/luytbq/flowcast/layout"
)

// args là đối số của một lệnh: cờ đứng trước hay sau tên file đều được, và
// nhận cả "--cờ giá trị" lẫn "--cờ=giá trị". Gói flag chuẩn của Go dừng ở đối
// số vị trí đầu tiên nên không dùng được.
type args struct {
	cmd        string
	file       string
	output     string
	title      string
	direction  string
	layoutJSON string
	mode       string
	png        string // rỗng: không xuất; "-" nghĩa là đường mặc định
	verify     bool
	noBackup   bool
	sheet      string
	delimiter  string
	encoding   string
	cfg        layout.Config
}

// configFlags ánh xạ cờ dòng lệnh tới trường của layout.Config. Tên cờ là giao
// diện mà script gọi flowcast dựa vào, nên chỉ đổi khi có chủ đích.
func configFlags(c *layout.Config) map[string]*int {
	out := map[string]*int{}
	for _, f := range layout.Fields() {
		out[f.Name] = f.Get(c)
	}
	return out
}

// errHelp báo rằng người dùng hỏi --help; caller in hướng dẫn rồi thoát 0.
var errHelp = errors.New("help")

// helpText dựng hướng dẫn. Phần tham số xếp hình sinh từ khai báo trong
// layout.Fields, nên thêm một tham số là nó có mặt ở đây.
func helpText() string {
	var b strings.Builder
	b.WriteString(`flowcast: bảng luồng hoặc flowchart mermaid thành file draw.io

dùng:
  flowcast check <file> [--sheet S] [--delimiter D] [--encoding E]
  flowcast build <file> [cờ]

cờ của build:
  -o, --output FILE    file .drawio ghi ra; mặc định cạnh file đầu vào
  --title T            tiêu đề, mặc định lấy từ nguồn
  --direction D        TD, BT, LR hoặc RL; mặc định theo nguồn, nguồn không nói thì TD
  --mode merge|force   khi file đích đã có: merge giữ chỉnh sửa tay, force sinh lại
  --no-backup          không ghi file .bak khi merge hoặc force
  --png [FILE]         xuất ảnh PNG bằng drawio CLI
  --verify             xuất SVG rồi so đường dây draw.io vẽ với toạ độ đã tính
  --layout-json FILE   ghi toạ độ đã tính ra JSON

tham số xếp hình, tính bằng điểm ảnh:
`)
	for _, f := range layout.Fields() {
		fmt.Fprintf(&b, "  --%-15s %s; mặc định %d, miền %d..%d\n", f.Name, f.Help, f.Default, f.Lo, f.Hi)
	}
	return b.String()
}

func parseArgs(argv []string) (*args, error) {
	if len(argv) == 0 {
		return nil, fmt.Errorf("thiếu lệnh; dùng check hoặc build")
	}
	if argv[0] == "--help" || argv[0] == "-h" {
		return nil, errHelp
	}
	a := &args{cmd: argv[0], cfg: layout.DefaultConfig()}
	if a.cmd != "check" && a.cmd != "build" {
		return nil, fmt.Errorf("lệnh %q không có; dùng check hoặc build", a.cmd)
	}
	ints := configFlags(&a.cfg)
	strs := map[string]*string{"sheet": &a.sheet, "delimiter": &a.delimiter, "encoding": &a.encoding}
	if a.cmd == "build" {
		strs["output"], strs["title"], strs["layout-json"] = &a.output, &a.title, &a.layoutJSON
		strs["mode"] = &a.mode
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
		case name == "help":
			return nil, errHelp
		case a.cmd == "build" && name == "png":
			// Giá trị không bắt buộc: lấy đối số kế tiếp làm đường ra nếu nó
			// không phải một cờ, ngược lại dùng đường mặc định.
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
