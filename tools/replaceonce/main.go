// Lệnh replaceonce thay đúng một chỗ trong một file, dùng cho tools/mutate.sh.
//
//	go run ./tools/replaceonce [-check] FILE TỪ THÀNH
//
// Chuỗi TỪ phải xuất hiện đúng một lần. Thay chỗ đầu khi có nhiều chỗ sẽ để lại
// bản sao còn nguyên, và đột biến thành vô hại, khiến bộ test trông như bỏ lọt.
// -check chỉ kiểm số chỗ khớp, không ghi file.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	check := flag.Bool("check", false, "chỉ kiểm, không ghi")
	flag.Parse()
	if flag.NArg() != 3 {
		fmt.Fprintln(os.Stderr, "dùng: replaceonce [-check] FILE TỪ THÀNH")
		os.Exit(2)
	}
	path, from, to := flag.Arg(0), flag.Arg(1), flag.Arg(2)
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	s := string(data)
	if n := strings.Count(s, from); n != 1 {
		fmt.Fprintf(os.Stderr, "ERROR %s: tìm thấy %d chỗ khớp %q, cần đúng 1\n", path, n, from)
		os.Exit(1)
	}
	if *check {
		return
	}
	if err := os.WriteFile(path, []byte(strings.Replace(s, from, to, 1)), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
