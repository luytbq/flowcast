// Package lint holds static checks on the module's own source code.
package lint

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestNoFloatMultiplicationExposedToFMA reports every float multiplication not
// wrapped in float64().
//
// The Go spec allows the compiler to fuse a*b + c into one FMA instruction,
// rounding once instead of twice, and it may fuse even across statements. On
// arm64 this happens in about a quarter of a*1.42 + c operations, off by one
// unit in the last bit compared with a machine that does not fuse, so the same
// table yields two different files. Only an explicit conversion prevents
// fusion, so every float multiplication must be wrapped, unless its result
// feeds straight into another multiplication or division: FMA only fuses
// multiplication with addition and subtraction.
func TestNoFloatMultiplicationExposedToFMA(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	var problems []string
	fset := token.NewFileSet()
	imp := importer.ForCompiler(fset, "source", nil)
	for _, dir := range packageDirs(t, root) {
		files := parseDir(t, fset, dir)
		if len(files) == 0 {
			continue
		}
		info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}}
		conf := types.Config{Importer: imp}
		if _, err := conf.Check(dir, fset, files, info); err != nil {
			t.Fatalf("%s: %v", dir, err)
		}
		for _, f := range files {
			problems = append(problems, scan(fset, root, f, info)...)
		}
	}
	sort.Strings(problems)
	for _, p := range problems {
		t.Error(p)
	}
}

func packageDirs(t *testing.T, root string) []string {
	var dirs []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if p != root && (strings.HasPrefix(name, ".") || name == "reference" || name == "testdata") {
			return filepath.SkipDir
		}
		dirs = append(dirs, p)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return dirs
}

// parseDir reads only the main source files, not test files: tests do not go
// into the binary and produce no coordinates.
func parseDir(t *testing.T, fset *token.FileSet, dir string) []*ast.File {
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var files []*ast.File
	for _, e := range entries {
		n := e.Name()
		if !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, n), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	return files
}

func isFloat(info *types.Info, e ast.Expr) bool {
	tv, ok := info.Types[e]
	if !ok {
		return false
	}
	b, ok := tv.Type.Underlying().(*types.Basic)
	return ok && b.Info()&types.IsFloat != 0 && tv.Value == nil
}

func isMul(e ast.Expr) bool {
	b, ok := e.(*ast.BinaryExpr)
	return ok && b.Op == token.MUL
}

func unparen(e ast.Expr) ast.Expr {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = p.X
	}
}

// scan walks the syntax tree, remembering each node's parent, and reports every
// multiplication without a safe parent.
func scan(fset *token.FileSet, root string, f *ast.File, info *types.Info) []string {
	var out []string
	var stack []ast.Node
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if b, ok := n.(*ast.BinaryExpr); ok && b.Op == token.MUL && isFloat(info, b) && !safeParent(stack, info) {
			pos := fset.Position(b.Pos())
			rel, _ := filepath.Rel(root, pos.Filename)
			out = append(out, rel+":"+itoa(pos.Line)+": float multiplication not wrapped in float64(), the compiler may fuse it into an FMA")
		}
		stack = append(stack, n)
		return true
	})
	return out
}

// safeParent: the nearest parent, ignoring parentheses, is a conversion to a
// float or integer type, or another float multiplication or division.
// Converting to an integer is also safe because it carries no addition to fuse
// with.
func safeParent(stack []ast.Node, info *types.Info) bool {
	for i := len(stack) - 1; i >= 0; i-- {
		switch p := stack[i].(type) {
		case *ast.ParenExpr:
			continue
		case *ast.CallExpr:
			id, ok := unparen(p.Fun).(*ast.Ident)
			return ok && conversions[id.Name] && len(p.Args) == 1
		case *ast.BinaryExpr:
			return (p.Op == token.MUL || p.Op == token.QUO) && isFloat(info, p)
		default:
			return false
		}
	}
	return false
}

var conversions = map[string]bool{"float64": true, "float32": true, "int": true, "int64": true}

func itoa(n int) string {
	s := ""
	if n == 0 {
		return "0"
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
