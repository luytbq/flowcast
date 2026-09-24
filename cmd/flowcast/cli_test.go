package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/luytbq/flowcast/layout"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const transcripts = "../../conformance/cli"

var update = flag.Bool("update", false, "ghi lại bản ghi CLI từ kết quả hiện tại")

// TestCLIKhopBanGhi phát lại từng bản ghi phiên làm việc của CLI và so từng
// dòng in ra, mã thoát, và nội dung từng file sinh ra.
//
// Dòng in ra và mã thoát là giao diện mà script và agent dựa vào, nên chúng chỉ
// được đổi có chủ đích: sửa code, chạy go test ./cmd/flowcast -update, rồi đọc
// diff của các bản ghi.
func TestCLIKhopBanGhi(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join(transcripts, "*.txt"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("không có bản ghi nào: %v", err)
	}
	for _, p := range paths {
		want, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(strings.TrimSuffix(filepath.Base(p), ".txt"), func(t *testing.T) {
			got := replay(t, string(want))
			if *update {
				if err := os.WriteFile(p, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			if got != string(want) {
				t.Errorf("lệch:\n--- có\n%s--- bản ghi\n%s", got, want)
			}
		})
	}
	t.Logf("đã phát lại %d bản ghi", len(paths))
}

// --help liệt kê mọi tham số xếp hình, lấy từ khai báo trong layout.Fields.
func TestHelpLietKeMoiThamSo(t *testing.T) {
	for _, argv := range [][]string{{"--help"}, {"-h"}, {"build", "--help"}, {"check", "--help"}} {
		var out, errOut bytes.Buffer
		if code := run(argv, strings.NewReader(""), &out, &errOut); code != 0 || errOut.Len() > 0 {
			t.Fatalf("%v: mã %d, stderr %q", argv, code, errOut.String())
		}
		for _, f := range layout.Fields() {
			if !strings.Contains(out.String(), "--"+f.Name+" ") {
				t.Errorf("%v: hướng dẫn thiếu --%s", argv, f.Name)
			}
		}
		if !strings.Contains(out.String(), "--direction") || !strings.Contains(out.String(), "--mode") {
			t.Errorf("%v: hướng dẫn thiếu cờ cơ bản", argv)
		}
	}
}

func replay(t *testing.T, want string) string {
	t.Helper()
	lines := strings.Split(want, "\n")
	var caseName, fname string
	if _, err := fmt.Sscanf(lines[0], "# input: %s -> %s", &caseName, &fname); err != nil {
		t.Fatalf("dòng đầu %q không đúng dạng: %v", lines[0], err)
	}
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if dir, ok := strings.CutPrefix(caseName, "merge/"); ok {
		// Kịch bản merge: chép cả thư mục, gồm bảng và file .drawio cũ.
		src := filepath.Join("../../conformance/merge", dir)
		entries, err := os.ReadDir(src)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			data, err := os.ReadFile(filepath.Join(src, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(tmp, e.Name()), data, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	} else {
		var src []byte
		for _, ext := range []string{".md", ".csv", ".xlsx"} {
			if src, err = os.ReadFile(filepath.Join("../../conformance/cases", caseName+ext)); err == nil {
				break
			}
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmp, fname), src, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var b strings.Builder
	b.WriteString(lines[0] + "\n")
	if fake, ok := strings.CutPrefix(lines[1], "# drawio: "); ok {
		// PATH chỉ gồm thư mục drawio giả, để drawio thật trên máy không lọt vào.
		// drawio giả chỉ dùng lệnh có sẵn trong sh nên không cần thêm gì.
		dir, err := filepath.Abs(filepath.Join("../../conformance/drawio-fakes", fake))
		if err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", dir)
		b.WriteString(lines[1] + "\n")
	}
	for _, ln := range lines[1:] {
		if !strings.HasPrefix(ln, "$ ") {
			continue
		}
		var cmd []string
		if err := json.Unmarshal([]byte(ln[2:]), &cmd); err != nil {
			t.Fatalf("lệnh %q: %v", ln, err)
		}
		argv := make([]string, len(cmd))
		for i, a := range cmd {
			argv[i] = strings.ReplaceAll(a, "$T", tmp)
		}
		var out, errOut bytes.Buffer
		code := inDir(t, tmp, func() int { return run(argv, strings.NewReader(""), &out, &errOut) })
		if errOut.Len() > 0 {
			t.Errorf("CLI Go ghi ra stderr: %s", errOut.String())
		}
		b.WriteString(ln + "\n" + strings.ReplaceAll(out.String(), tmp, "$T") + fmt.Sprintf("[exit %d]\n", code))
	}
	b.WriteString("--- files\n")
	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, n := range names {
		data, err := os.ReadFile(filepath.Join(tmp, n))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		b.WriteString(hex.EncodeToString(sum[:])[:16] + "  " + n + "\n")
	}
	return b.String()
}

// inDir chạy f với thư mục làm việc là dir, để đường dẫn tương đối trong bản ghi
// trỏ vào thư mục tạm.
func inDir(t *testing.T, dir string, f func() int) int {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	return f()
}
