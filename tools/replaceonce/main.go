// Command replaceonce replaces exactly one occurrence in a file, used by tools/mutate.sh.
//
//	go run ./tools/replaceonce [-check] FILE FROM TO
//
// The string FROM must occur exactly once. Replacing the first of several
// occurrences would leave an intact copy behind and make the mutation harmless,
// so the test suite would look as if it missed it.
// -check only checks the match count and does not write the file.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	check := flag.Bool("check", false, "check only, do not write")
	flag.Parse()
	if flag.NArg() != 3 {
		fmt.Fprintln(os.Stderr, "usage: replaceonce [-check] FILE FROM TO")
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
		fmt.Fprintf(os.Stderr, "ERROR %s: found %d matches of %q, need exactly 1\n", path, n, from)
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
