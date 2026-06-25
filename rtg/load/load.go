package load

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"j5.nz/rtg/rtg/mod"
)

type File struct {
	Path   string
	Source []byte
	Body   string
}

type Package struct {
	ImportPath  string
	Dir         string
	Name        string
	Files       []File
	Imports     []string
	ImportNames map[string]string
}

type Graph struct {
	Module   mod.Module
	Packages []Package
}

type Options struct {
	StdRoot string
}

func LoadEntries(entries []string, opts Options) (*Graph, error) {
	if len(entries) == 0 {
		entries = []string{"."}
	}
	module, err := mod.Find(entries[0])
	if err != nil {
		return nil, err
	}
	if opts.StdRoot == "" {
		opts.StdRoot = filepath.Join(module.Root, "rtg", "std")
	}
	g := &Graph{Module: module}
	seen := map[string]bool{}
	for _, entry := range entries {
		dir, err := entryDir(entry)
		if err != nil {
			return nil, err
		}
		if err := loadPackageRecursive(g, opts, seen, dir); err != nil {
			return nil, err
		}
	}
	return g, nil
}

func entryDir(entry string) (string, error) {
	info, err := os.Stat(entry)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return filepath.Abs(entry)
	}
	return filepath.Abs(filepath.Dir(entry))
}

func loadPackageRecursive(g *Graph, opts Options, seen map[string]bool, dir string) error {
	return loadPackageRecursiveAs(g, opts, seen, dir, importPathForDir(g.Module, dir))
}

func loadPackageRecursiveAs(g *Graph, opts Options, seen map[string]bool, dir string, importPath string) error {
	dir = filepath.Clean(dir)
	if seen[dir] {
		return nil
	}
	seen[dir] = true
	pkg, err := readPackage(g.Module, dir, importPath)
	if err != nil {
		return err
	}
	g.Packages = append(g.Packages, pkg)
	for _, imp := range pkg.Imports {
		next, ok, err := resolveImport(g.Module, opts, imp)
		if err != nil {
			return err
		}
		if ok {
			if err := loadPackageRecursiveAs(g, opts, seen, next.Dir, next.ImportPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func readPackage(module mod.Module, dir string, importPath string) (Package, error) {
	files, err := goFiles(dir)
	if err != nil {
		return Package{}, err
	}
	if len(files) == 0 {
		return Package{}, fmt.Errorf("%s: no Go source files", dir)
	}
	pkg := Package{Dir: dir, ImportPath: importPath, ImportNames: map[string]string{}}
	importSet := map[string]bool{}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return Package{}, err
		}
		info, err := ParseSourceInfo(path, data)
		if err != nil {
			return Package{}, err
		}
		if pkg.Name == "" {
			pkg.Name = info.PackageName
		} else if pkg.Name != info.PackageName {
			return Package{}, fmt.Errorf("%s: mixed package names %s and %s", dir, pkg.Name, info.PackageName)
		}
		for _, imp := range info.Imports {
			importSet[imp.Path] = true
			if imp.Alias != "." && imp.Alias != "_" {
				pkg.ImportNames[imp.Path] = imp.Name
			}
		}
		body := string(data[info.BodyStart:])
		pkg.Files = append(pkg.Files, File{Path: path, Source: data, Body: strings.TrimLeft(body, " \t\r\n")})
	}
	for imp := range importSet {
		pkg.Imports = append(pkg.Imports, imp)
	}
	sort.Strings(pkg.Imports)
	return pkg, nil
}

func PackageNameFromImportPath(path string) string {
	slash := -1
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			slash = i
		}
	}
	if slash >= 0 {
		return path[slash+1:]
	}
	return path
}

func goFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, ".rtg.go") {
			continue
		}
		files = append(files, filepath.Join(dir, name))
	}
	sort.Strings(files)
	return files, nil
}

type resolvedImport struct {
	Dir        string
	ImportPath string
}

func resolveImport(module mod.Module, opts Options, imp string) (resolvedImport, bool, error) {
	if imp == module.Path {
		return resolvedImport{Dir: module.Root, ImportPath: imp}, true, nil
	}
	prefix := module.Path + "/"
	if strings.HasPrefix(imp, prefix) {
		return resolvedImport{Dir: filepath.Join(module.Root, filepath.FromSlash(strings.TrimPrefix(imp, prefix))), ImportPath: imp}, true, nil
	}
	stdDir := filepath.Join(opts.StdRoot, filepath.FromSlash(imp))
	if info, err := os.Stat(stdDir); err == nil && info.IsDir() {
		return resolvedImport{Dir: stdDir, ImportPath: imp}, true, nil
	}
	return resolvedImport{}, false, fmt.Errorf("import %q is not in module %q and was not found in rtg/std", imp, module.Path)
}

func importPathForDir(module mod.Module, dir string) string {
	rel, err := filepath.Rel(module.Root, dir)
	if err != nil || rel == "." {
		return module.Path
	}
	return module.Path + "/" + filepath.ToSlash(rel)
}
