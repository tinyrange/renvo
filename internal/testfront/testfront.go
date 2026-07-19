//go:build !renvo

package testfront

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrNoTests       = errors.New("no tests found")
	ErrExternalTests = errors.New("external test packages are not supported")
)

type GeneratedFile struct {
	Name string
	Data []byte
}

type Result struct {
	Package string
	Tests   []string
	Files   []GeneratedFile
}

func GeneratePackage(dir string) (Result, error) {
	dir = filepath.Clean(dir)
	pkg, err := build.Default.ImportDir(dir, 0)
	if err != nil {
		return Result{}, err
	}
	if len(pkg.XTestGoFiles) > 0 {
		return Result{}, ErrExternalTests
	}
	files := append([]string{}, pkg.GoFiles...)
	files = append(files, pkg.TestGoFiles...)
	sort.Strings(files)
	var out Result
	out.Package = pkg.Name
	usedNames := make(map[string]bool, len(files)+1)
	for _, name := range files {
		path := filepath.Join(dir, name)
		src, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		rewritten, tests, err := rewriteFile(name, src)
		if err != nil {
			return Result{}, err
		}
		out.Tests = append(out.Tests, tests...)
		generatedName := generatedFileName(name)
		if usedNames[generatedName] {
			return Result{}, fmt.Errorf("%s: generated file name collision", generatedName)
		}
		usedNames[generatedName] = true
		out.Files = append(out.Files, GeneratedFile{Name: generatedName, Data: rewritten})
	}
	sort.Strings(out.Tests)
	if len(out.Tests) == 0 {
		return Result{}, ErrNoTests
	}
	out.Files = append(out.Files, GeneratedFile{Name: "renvo_testmain.go", Data: testMainSource(out.Tests)})
	return out, nil
}

func generatedFileName(name string) string {
	if strings.HasSuffix(name, "_test.go") {
		base := strings.TrimSuffix(name, "_test.go")
		return "renvotest_" + base + "_testsrc.go"
	}
	return name
}

func WritePackage(dir string, result Result) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, file := range result.Files {
		path := filepath.Join(dir, file.Name)
		if err := os.WriteFile(path, file.Data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func rewriteFile(name string, src []byte) ([]byte, []string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil, nil, err
	}
	start := fset.Position(file.Name.Pos()).Offset
	end := fset.Position(file.Name.End()).Offset
	if start < 0 || end < start || end > len(src) {
		return nil, nil, fmt.Errorf("%s: invalid package clause", name)
	}
	var out bytes.Buffer
	out.Grow(len(src) + 8)
	out.Write(src[:start])
	out.WriteString("main")
	out.Write(src[end:])
	tests, err := testNames(file)
	if err != nil {
		return nil, nil, err
	}
	return out.Bytes(), tests, nil
}

func testNames(file *ast.File) ([]string, error) {
	var names []string
	testingNames := testingImportNames(file)
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || !isTestName(fn.Name.Name) {
			continue
		}
		if !isTestFunc(fn, testingNames) {
			return nil, fmt.Errorf("%s: invalid test signature", fn.Name.Name)
		}
		names = append(names, fn.Name.Name)
	}
	return names, nil
}

func testingImportNames(file *ast.File) []string {
	var names []string
	for _, imp := range file.Imports {
		if imp.Path == nil || imp.Path.Value != "\"testing\"" {
			continue
		}
		name := "testing"
		if imp.Name != nil {
			name = imp.Name.Name
		}
		names = append(names, name)
	}
	return names
}

func isTestFunc(fn *ast.FuncDecl, testingNames []string) bool {
	if fn.Recv != nil || fn.Name == nil || !isTestName(fn.Name.Name) || fn.Type == nil {
		return false
	}
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		return false
	}
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}
	param := fn.Type.Params.List[0]
	if len(param.Names) > 1 {
		return false
	}
	return isTestingT(param.Type, testingNames)
}

func isTestName(name string) bool {
	if !strings.HasPrefix(name, "Test") {
		return false
	}
	if name == "Test" {
		return true
	}
	r, _ := utf8.DecodeRuneInString(name[len("Test"):])
	return r == utf8.RuneError || !unicode.IsLower(r)
}

func isTestingT(expr ast.Expr, testingNames []string) bool {
	ptr, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := ptr.X.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "T" {
		return false
	}
	base, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	for _, name := range testingNames {
		if base.Name == name {
			return true
		}
	}
	return false
}

func testMainSource(tests []string) []byte {
	var out bytes.Buffer
	out.WriteString("package main\n\n")
	out.WriteString("import (\n")
	out.WriteString("\t\"regexp\"\n")
	out.WriteString("\t\"testing\"\n")
	out.WriteString(")\n\n")
	out.WriteString("func matchString(pat string, str string) (bool, error) {\n")
	out.WriteString("\treturn regexp.MatchString(pat, str)\n")
	out.WriteString("}\n\n")
	out.WriteString("func main() {\n")
	out.WriteString("\ttesting.Main(matchString, []testing.InternalTest{\n")
	for _, test := range tests {
		fmt.Fprintf(&out, "\t\t{Name: %q, F: %s},\n", test, test)
	}
	out.WriteString("\t}, nil, nil)\n")
	out.WriteString("}\n")
	return out.Bytes()
}
