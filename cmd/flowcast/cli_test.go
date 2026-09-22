package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/luytbq/flowcast/conformance"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const transcripts = "../../conformance/cli"

// TestCLIKhopBanThamChieu phát lại từng bản ghi phiên làm việc của CLI bản tham
// chiếu và so từng dòng in ra, mã thoát, và nội dung từng file sinh ra.
//
// Subagent flowtable-drawio đọc mã thoát và chép nguyên văn các dòng tool in
// ra, nên đây là điều kiện để flowcast thay được bản Python.
func TestCLIKhopBanThamChieu(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join(transcripts, "*.txt"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("không có bản ghi nào: %v; chạy tools/cli_transcripts.py", err)
	}
	// Bản ghi của case trong diverge.txt là hành vi cũ mà bản Go cố ý bỏ.
	skip, err := conformance.Diverged("../../conformance")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		want, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var input string
		fmt.Sscanf(string(want), "# input: %s", &input)
		if _, ok := skip[input]; ok {
			continue
		}
		if _, ok := skip["cli:"+strings.TrimSuffix(filepath.Base(p), ".txt")]; ok {
			continue
		}
		t.Run(strings.TrimSuffix(filepath.Base(p), ".txt"), func(t *testing.T) {
			got := replay(t, string(want))
			if got != string(want) {
				t.Errorf("lệch:\n--- Go\n%s--- Python\n%s", got, want)
			}
		})
	}
	t.Logf("đã phát lại %d bản ghi", len(paths))
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
		// Chỉ drawio giả nằm trong PATH, như lúc bản tham chiếu sinh bản ghi.
		dir, err := filepath.Abs(filepath.Join("../../conformance/drawio-fakes", fake))
		if err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", dir+":/usr/bin:/bin")
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

// inDir chạy f với thư mục làm việc là dir, như bản tham chiếu được chạy trong
// thư mục tạm.
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
