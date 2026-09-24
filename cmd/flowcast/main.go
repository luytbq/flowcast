// Lệnh flowcast đọc một Flow Table rồi ghi file .drawio.
//
// Dòng in ra, mã thoát và tên cờ là giao diện mà script và agent dựa vào: chúng
// đọc mã thoát và chép nguyên văn các dòng ERROR layout:, ERROR render:, merge:.
// Vì vậy chúng được chốt bằng bản ghi CLI và chỉ đổi khi có chủ đích.
//
// Mã thoát:
//
//	0  ổn, có thể vẫn có cảnh báo
//	1  bảng có lỗi, không vẽ
//	2  tự kiểm hình học có lỗi
//	3  ảnh render lệch toạ độ, hoặc drawio CLI lỗi
//	4  file đích đã tồn tại mà chưa chọn chế độ, hoặc không đọc được file đích
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unicode"

	"github.com/luytbq/flowcast"
	"github.com/luytbq/flowcast/internal/unistr"
	"github.com/luytbq/flowcast/merge"
	"github.com/luytbq/flowcast/model"
	"github.com/luytbq/flowcast/num"
	"github.com/luytbq/flowcast/render"
	"github.com/luytbq/flowcast/source"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(argv []string, stdin io.Reader, stdout, stderr io.Writer) int {
	a, err := parseArgs(argv)
	if errors.Is(err, errHelp) {
		fmt.Fprint(stdout, helpText())
		return 0
	}
	if err != nil {
		fmt.Fprintln(stderr, "flowcast:", err)
		fmt.Fprintln(stderr, "dùng: flowcast check <file> | flowcast build <file> [-o out.drawio] [--mode merge|force] [--direction LR] ...")
		return 2
	}
	c := &cli{a: a, stdin: stdin, out: stdout}
	if a.cmd == "check" {
		return c.check()
	}
	return c.build()
}

type cli struct {
	a     *args
	stdin io.Reader
	out   io.Writer
}

func (c *cli) println(s ...any) { fmt.Fprintln(c.out, s...) }

// supported là các đuôi file đọc được, xét trước khi mở file, đúng như bản
// tham chiếu: đuôi lạ bị từ chối kể cả khi file không tồn tại.
var supported = map[string]bool{".md": true, ".markdown": true, ".txt": true,
	".csv": true, ".tsv": true, ".xlsx": true, ".xlsm": true, ".mmd": true, ".mermaid": true}

// read đọc file đầu vào thành Source. Lỗi trả về đã ở dạng thông điệp in ra.
func (c *cli) read() (flowcast.Source, error) {
	ext := unistr.Lower(source.Ext(c.a.file))
	if !supported[ext] {
		return flowcast.Source{}, fmt.Errorf("đuôi file \"%s\" không hỗ trợ; dùng .md, .csv, .xlsx hoặc .mmd", ext)
	}
	data, err := os.ReadFile(c.a.file)
	if err != nil {
		return flowcast.Source{}, fmt.Errorf("không đọc được %s: %s", c.a.file, strerror(err))
	}
	opts := map[string]string{}
	for k, v := range map[string]string{"sheet": c.a.sheet, "delimiter": c.a.delimiter, "encoding": c.a.encoding} {
		if v != "" {
			opts[k] = v
		}
	}
	return flowcast.Source{Data: data, Name: c.a.file, Options: opts}, nil
}

// printIssues in lỗi trước cảnh báo sau, giữ nguyên thứ tự trong mỗi nhóm, rồi
// trả về số lỗi.
func (c *cli) printIssues(issues []model.Issue) int {
	sorted := append([]model.Issue(nil), issues...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Level == model.LevelError && sorted[j].Level != model.LevelError
	})
	ne := 0
	for _, i := range sorted {
		c.println(i.String())
		if i.Level == model.LevelError {
			ne++
		}
	}
	c.println(fmt.Sprintf("check: %d lỗi, %d cảnh báo", ne, len(issues)-ne))
	return ne
}

func (c *cli) check() int {
	src, err := c.read()
	if err != nil {
		c.println("ERROR   " + err.Error())
		return 1
	}
	r, err := flowcast.Check(src, flowcast.Options{Limits: &flowcast.CLILimits})
	if err != nil {
		c.println("ERROR   " + err.Error())
		return 1
	}
	c.println("check: đọc bảng từ " + r.Source)
	if c.printIssues(r.Issues) > 0 {
		return 1
	}
	return 0
}

func (c *cli) build() int {
	src, err := c.read()
	if err != nil {
		c.println("ERROR   " + err.Error())
		return 1
	}
	chk, err := flowcast.Check(src, flowcast.Options{Limits: &flowcast.CLILimits})
	if err != nil {
		c.println("ERROR   " + err.Error())
		return 1
	}
	c.println("check: đọc bảng từ " + chk.Source)
	if c.printIssues(chk.Issues) > 0 {
		c.println("build: dừng vì bảng có lỗi")
		return 1
	}

	out := c.a.output
	if out == "" {
		out = strings.TrimSuffix(c.a.file, source.Ext(c.a.file)) + ".drawio"
	}
	mode, ok := c.chooseMode(out)
	if !ok {
		c.println("build: dừng, chưa chọn chế độ ghi")
		return 4
	}
	var prev *merge.Old
	if mode == "merge" {
		data, err := os.ReadFile(out)
		if err == nil {
			prev, err = merge.Read(data, out)
		} else {
			err = fmt.Errorf("không đọc được %s: %v", out, err)
		}
		if err != nil {
			c.println("ERROR   " + err.Error() + "; không ghi đè. Dùng --mode force nếu muốn sinh lại toàn bộ")
			return 4
		}
	}

	r, err := flowcast.Build(src, flowcast.Options{Config: &c.a.cfg, Title: c.a.title, Previous: prev,
		Direction: c.a.direction, Limits: &flowcast.CLILimits})
	if err != nil {
		c.println("ERROR   " + err.Error())
		return 1
	}
	backup := ""
	if mode != "new" && !c.a.noBackup {
		backup = out + ".bak"
		if err := copyFile(out, backup); err != nil {
			c.println("ERROR   " + err.Error())
			return 4
		}
	}
	if err := os.WriteFile(out, []byte(r.Text), 0o644); err != nil {
		c.println("ERROR   " + err.Error())
		return 1
	}
	c.println("build: chế độ " + map[string]string{
		"new": "tạo mới", "force": "force, sinh lại toàn bộ", "merge": "merge, giữ chỉnh sửa tay"}[mode])
	s := r.Stats
	c.println(fmt.Sprintf("build: đã ghi %s (%d lane, %d phần tử, %d cạnh, %sx%spx)",
		out, s.Lanes, s.Items, s.Edges, num.Fmt(s.W), num.Fmt(s.H)))
	if backup != "" {
		c.println("build: bản sao file cũ " + backup)
	}
	if c.a.layoutJSON != "" {
		if err := writeLayoutJSON(c.a.layoutJSON, r); err != nil {
			c.println("ERROR   " + err.Error())
			return 1
		}
	}
	for _, w := range r.Warnings {
		c.println("WARNING layout: " + w.Msg)
	}
	if r.Merge != nil {
		for _, l := range r.Merge.Lines() {
			c.println(l)
		}
		c.println("layout: merge không tự kiểm dây và nhãn; xem danh sách ở trên và ảnh PNG")
		return c.exportAndVerify(out, r, 0)
	}
	nerr := 0
	for _, f := range r.Findings {
		c.println(fmt.Sprintf("%-7s layout: %s", strings.ToUpper(f.Level), f.Msg))
		if f.Level == "error" {
			nerr++
		}
	}
	c.println(fmt.Sprintf("layout: %d lỗi, %d cảnh báo", nerr, len(r.Findings)-nerr))
	code := 0
	if nerr > 0 {
		code = 2
	}
	return c.exportAndVerify(out, r, code)
}

// exportAndVerify xuất PNG và kiểm render khi được yêu cầu. Lỗi xuất ảnh không
// làm hỏng file .drawio đã ghi, chỉ đổi mã thoát thành 3 nếu chưa có mã khác.
func (c *cli) exportAndVerify(out string, r flowcast.Result, code int) int {
	if c.a.png == "" && !c.a.verify {
		return code
	}
	exe := render.Bin()
	if exe == "" {
		c.println("WARNING không có drawio CLI, bỏ qua xuất ảnh và kiểm render")
		return code
	}
	verify := c.a.verify
	if r.Merge != nil && verify {
		c.println("render: bỏ qua kiểm render ở chế độ merge (đường dây giữ từ file hoặc do draw.io tự đi)")
		verify = false
	}
	fail := func(err error) int {
		c.println("ERROR   " + err.Error())
		if code != 0 {
			return code
		}
		return 3
	}
	ctx := context.Background()
	if c.a.png != "" {
		png := c.a.png
		if png == "-" {
			png = strings.TrimSuffix(out, source.Ext(out)) + ".png"
		}
		if err := render.Export(ctx, exe, out, png, "png", 2); err != nil {
			return fail(err)
		}
		c.println("png: " + png)
	}
	if verify {
		dir, err := os.MkdirTemp("", "flowcast-verify")
		if err != nil {
			return fail(err)
		}
		defer os.RemoveAll(dir)
		svg := filepath.Join(dir, "render.svg")
		if err := render.Export(ctx, exe, out, svg, "svg", 0); err != nil {
			return fail(err)
		}
		data, err := os.ReadFile(svg)
		if err != nil {
			return fail(err)
		}
		probs, err := render.Verify(*r.Layout, data)
		if err != nil {
			return fail(fmt.Errorf("không đọc được SVG do drawio xuất: %v", err))
		}
		for _, p := range probs {
			c.println("ERROR   render: " + p)
		}
		c.println(fmt.Sprintf("render: %d điểm lệch", len(probs)))
		if len(probs) > 0 && code == 0 {
			code = 3
		}
	}
	return code
}

// chooseMode trả về "new", "force" hoặc "merge". ok là false khi phải dừng: file
// đích đã có, chưa chọn chế độ, và không có ai ở terminal để hỏi.
func (c *cli) chooseMode(out string) (string, bool) {
	if _, err := os.Stat(out); err != nil {
		return "new", true
	}
	if c.a.mode != "" {
		return c.a.mode, true
	}
	if !isTerminal(c.stdin) {
		c.println("ERROR   " + out + " đã tồn tại; chọn --mode merge (giữ vị trí, lane, waypoint đã sửa tay) " +
			"hoặc --mode force (sinh lại toàn bộ)")
		return "", false
	}
	fmt.Fprint(c.out, out+" đã tồn tại. [m]erge giữ chỉnh sửa tay / [f]orce sinh lại toàn bộ / [q]uit: ")
	line, _ := bufio.NewReader(c.stdin).ReadString('\n')
	switch unistr.Lower(unistr.Strip(line)) {
	case "m", "merge":
		return "merge", true
	case "f", "force":
		return "force", true
	}
	return "", false
}

func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

// strerror dựng lại thông điệp lỗi hệ thống theo kiểu strerror của C, thứ bản
// tham chiếu in ra: "No such file or directory" chứ không phải "open x: no such
// file or directory" như Go.
func strerror(err error) string {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		msg := errno.Error()
		r := []rune(msg)
		r[0] = unicode.ToUpper(r[0])
		return string(r)
	}
	return err.Error()
}

// copyFile chép file cũ trước khi ghi đè, giữ quyền và thời điểm sửa của file
// gốc.
func copyFile(src, dst string) error {
	st, err := os.Stat(src)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, st.Mode().Perm()); err != nil {
		return err
	}
	return os.Chtimes(dst, st.ModTime(), st.ModTime())
}

// writeLayoutJSON ghi toạ độ đã tính ra JSON, để gỡ lỗi bố cục hoặc cho công cụ
// khác đọc. Cấu trúc khóa ổn định; cách viết số thì không được cam kết.
func writeLayoutJSON(path string, r flowcast.Result) error {
	l := r.Layout
	lanes := []map[string]any{}
	for i, ln := range l.Lanes {
		lanes = append(lanes, map[string]any{"id": ln.ID, "x": l.LaneX[i], "w": l.LaneW[i]})
	}
	items := map[string]any{}
	for _, it := range l.Items {
		items[it.ID] = map[string]any{"kind": it.Kind, "row": it.Row, "col": it.Col,
			"x": it.X, "y": it.Y, "w": it.W, "h": it.H}
	}
	edges := map[string]any{}
	for _, e := range l.Edges {
		edges[e.ID] = map[string]any{"case": string(e.Case), "exit": string(e.ExitSide),
			"points": e.Pts, "label": e.Label}
	}
	b, err := json.MarshalIndent(map[string]any{
		"pool": map[string]any{"w": l.PoolW, "h": l.PoolH}, "lanes": lanes, "items": items, "edges": edges,
	}, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
