//go:build !rtg

package std_test

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type publicDeclaration struct {
	file      string
	signature string
}

func TestHostAndRTGCommonAPIsMatch(t *testing.T) {
	root := stdRoot(t)
	packages := stdPackageDirectories(t, root)
	targets := []struct {
		goos   string
		goarch string
	}{
		{goos: "linux", goarch: "amd64"},
		{goos: "darwin", goarch: "arm64"},
		{goos: "windows", goarch: "amd64"},
		{goos: "wasip1", goarch: "wasm"},
	}
	compared := 0
	for _, target := range targets {
		target := target
		t.Run(target.goos+"_"+target.goarch, func(t *testing.T) {
			hostContext := build.Default
			hostContext.CgoEnabled = false
			hostContext.GOOS = target.goos
			hostContext.GOARCH = target.goarch
			rtgContext := hostContext
			rtgContext.BuildTags = append([]string{}, hostContext.BuildTags...)
			rtgContext.BuildTags = append(rtgContext.BuildTags, "rtg")
			for _, dir := range packages {
				relative, err := filepath.Rel(root, dir)
				if err != nil {
					t.Fatal(err)
				}
				host := loadPublicAPI(t, dir, &hostContext)
				rtg := loadPublicAPI(t, dir, &rtgContext)
				for name, hostDeclaration := range host {
					rtgDeclaration, ok := rtg[name]
					if !ok {
						continue
					}
					compared++
					key := filepath.ToSlash(relative) + "." + name
					if publicAPIException(key) {
						continue
					}
					if hostDeclaration.signature != rtgDeclaration.signature {
						t.Errorf("%s: host %s (%s), rtg %s (%s)", key, hostDeclaration.signature, hostDeclaration.file, rtgDeclaration.signature, rtgDeclaration.file)
					}
				}
			}
		})
	}
	if compared == 0 {
		t.Fatal("no common host/RTG declarations were compared")
	}
}

func publicAPIException(key string) bool {
	// RTG's unsafe.Pointer is a compiler-recognized data pointer. The host shim
	// must alias Go's unsafe.Pointer so host-side tests can use unsafe intrinsics.
	return key == "unsafe.Pointer"
}

func stdPackageDirectories(t *testing.T, root string) []string {
	t.Helper()
	seen := make(map[string]bool)
	var directories []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		dir := filepath.Dir(path)
		if !seen[dir] {
			seen[dir] = true
			directories = append(directories, dir)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(directories)
	return directories
}

func loadPublicAPI(t *testing.T, dir string, context *build.Context) map[string]publicDeclaration {
	t.Helper()
	pkg, err := context.ImportDir(dir, 0)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			return nil
		}
		t.Fatalf("load %s for %s/%s: %v", dir, context.GOOS, context.GOARCH, err)
	}
	files := append([]string{}, pkg.GoFiles...)
	files = append(files, pkg.CgoFiles...)
	sort.Strings(files)
	api := make(map[string]publicDeclaration)
	fset := token.NewFileSet()
	for _, name := range files {
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", filepath.Join(dir, name), err)
		}
		for _, declaration := range file.Decls {
			switch declaration := declaration.(type) {
			case *ast.FuncDecl:
				if declaration.Name == nil || !declaration.Name.IsExported() {
					continue
				}
				key := declaration.Name.Name
				receiver := ""
				if declaration.Recv != nil && len(declaration.Recv.List) == 1 {
					receiverName := receiverBaseName(declaration.Recv.List[0].Type)
					if !ast.IsExported(receiverName) {
						continue
					}
					receiver = expressionString(fset, declaration.Recv.List[0].Type)
					key = receiverName + "." + key
				}
				signature := "func" + fieldListSignature(fset, declaration.Type.Params) + fieldListSignature(fset, declaration.Type.Results)
				if receiver != "" {
					signature = "method(" + receiver + ")" + signature
				}
				addPublicDeclaration(t, api, key, publicDeclaration{file: name, signature: signature})
			case *ast.GenDecl:
				for _, specification := range declaration.Specs {
					switch specification := specification.(type) {
					case *ast.TypeSpec:
						if specification.Name.IsExported() {
							addPublicDeclaration(t, api, specification.Name.Name, publicDeclaration{file: name, signature: publicTypeSignature(fset, specification.Type)})
						}
					case *ast.ValueSpec:
						kind := declaration.Tok.String()
						for _, valueName := range specification.Names {
							if valueName.IsExported() {
								addPublicDeclaration(t, api, valueName.Name, publicDeclaration{file: name, signature: kind})
							}
						}
					}
				}
			}
		}
	}
	return api
}

func addPublicDeclaration(t *testing.T, api map[string]publicDeclaration, key string, declaration publicDeclaration) {
	t.Helper()
	if previous, ok := api[key]; ok {
		t.Fatalf("duplicate public declaration %s in %s and %s", key, previous.file, declaration.file)
	}
	api[key] = declaration
}

func receiverBaseName(expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.StarExpr:
		return receiverBaseName(expression.X)
	case *ast.Ident:
		return expression.Name
	case *ast.IndexExpr:
		return receiverBaseName(expression.X)
	case *ast.IndexListExpr:
		return receiverBaseName(expression.X)
	}
	return "?"
}

func fieldListSignature(fset *token.FileSet, fields *ast.FieldList) string {
	if fields == nil || len(fields.List) == 0 {
		return "()"
	}
	var parts []string
	for _, field := range fields.List {
		fieldType := expressionString(fset, field.Type)
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			parts = append(parts, fieldType)
		}
	}
	return "(" + strings.Join(parts, ",") + ")"
}

func publicTypeSignature(fset *token.FileSet, expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.StructType:
		return "type struct" + publicFieldsSignature(fset, expression.Fields)
	case *ast.InterfaceType:
		return "type interface" + publicFieldsSignature(fset, expression.Methods)
	default:
		return "type " + expressionString(fset, expression)
	}
}

func publicFieldsSignature(fset *token.FileSet, fields *ast.FieldList) string {
	if fields == nil {
		return "{}"
	}
	var parts []string
	for _, field := range fields.List {
		if len(field.Names) == 0 {
			name := receiverBaseName(field.Type)
			if ast.IsExported(name) {
				parts = append(parts, expressionString(fset, field.Type))
			}
			continue
		}
		for _, name := range field.Names {
			if name.IsExported() {
				parts = append(parts, name.Name+" "+expressionString(fset, field.Type))
			}
		}
	}
	sort.Strings(parts)
	return "{" + strings.Join(parts, ";") + "}"
}

func expressionString(fset *token.FileSet, expression ast.Expr) string {
	var out bytes.Buffer
	if err := format.Node(&out, fset, expression); err != nil {
		return fmt.Sprintf("<invalid:%T>", expression)
	}
	return out.String()
}
