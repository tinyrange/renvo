package load

import "j5.nz/rtg/rtg/internal/syntax"

const (
	PackageOK = iota
	PackageErrRef
	PackageErrNoFiles
	PackageErrParse
	PackageErrName
	PackageErrImport
)

const (
	GraphOK = iota
	GraphErrRoot
	GraphErrPackage
	GraphErrCycle
)

type SourceFile struct {
	Path string
	Src  []byte
}

type ParsedFile struct {
	Path string
	Src  []byte
	File syntax.File
}

type Package struct {
	Ref         PackageRef
	Name        string
	Files       []ParsedFile
	Imports     []PackageRef
	Ok          bool
	Error       int
	ErrorFile   int
	ErrorImport int
}

type Graph struct {
	Module       Module
	Root         string
	Packages     []Package
	Ok           bool
	Error        int
	ErrorPackage int
}

func LoadGraph(module Module, stdRoot string, workDir string, arg string, files []SourceFile) Graph {
	ref := ResolvePackageArg(module, workDir, arg)
	if !ref.Ok {
		return Graph{Module: module, Ok: false, Error: GraphErrRoot, ErrorPackage: -1}
	}
	return LoadGraphFromRoot(module, stdRoot, ref, files)
}

func LoadGraphFromRoot(module Module, stdRoot string, root PackageRef, files []SourceFile) Graph {
	var builder graphBuilder
	builder.module = module
	builder.stdRoot = CleanPath(stdRoot)
	builder.files = files
	builder.graph = Graph{Module: module, Root: root.ImportPath, Ok: true, Error: GraphOK, ErrorPackage: -1}
	builder.load(root)
	if !builder.graph.Ok {
		return builder.graph
	}
	return builder.graph
}

func LoadPackage(module Module, stdRoot string, ref PackageRef, files []SourceFile) Package {
	pkg := Package{
		Ref:         ref,
		Ok:          true,
		Error:       PackageOK,
		ErrorFile:   -1,
		ErrorImport: -1,
	}
	if !ref.Ok || ref.Dir == "" {
		return packageFail(pkg, PackageErrRef, -1, -1)
	}
	selected := selectPackageFiles(ref.Dir, files)
	if len(selected) == 0 {
		return packageFail(pkg, PackageErrNoFiles, -1, -1)
	}
	for i := 0; i < len(selected); i++ {
		parsed := syntax.ParseFile(selected[i].Src)
		if !parsed.Ok {
			pkg.Files = append(pkg.Files, newParsedFile(selected[i].Path, selected[i].Src, parsed))
			return packageFail(pkg, PackageErrParse, i, -1)
		}
		name := string(syntax.TokenText(parsed.Src, parsed.Tokens[parsed.PackageName]))
		if pkg.Name == "" {
			pkg.Name = name
		} else if pkg.Name != name {
			pkg.Files = append(pkg.Files, newParsedFile(selected[i].Path, selected[i].Src, parsed))
			return packageFail(pkg, PackageErrName, i, -1)
		}
		refs := FileImports(module, stdRoot, parsed)
		for j := 0; j < len(refs); j++ {
			pkg.Imports = appendImport(pkg.Imports, refs[j])
			if !refs[j].Ok {
				pkg.Files = append(pkg.Files, newParsedFile(selected[i].Path, selected[i].Src, parsed))
				return packageFail(pkg, PackageErrImport, i, len(pkg.Imports)-1)
			}
		}
		pkg.Files = append(pkg.Files, newParsedFile(selected[i].Path, selected[i].Src, parsed))
	}
	return pkg
}

func newParsedFile(path string, src []byte, file syntax.File) ParsedFile {
	return ParsedFile{
		Path: path,
		Src:  src,
		File: file,
	}
}

type graphBuilder struct {
	module  Module
	stdRoot string
	files   []SourceFile
	loading []string
	graph   Graph
}

func (b *graphBuilder) load(ref PackageRef) int {
	if !b.graph.Ok {
		return -1
	}
	if ref.Kind != PackageInModule && ref.Kind != PackageStandard {
		b.graph = graphFail(b.graph, GraphErrPackage, -1)
		return -1
	}
	loaded := findLoadedPackage(b.graph.Packages, ref.ImportPath)
	if loaded >= 0 {
		return loaded
	}
	if findString(b.loading, ref.ImportPath) >= 0 {
		b.graph = graphFail(b.graph, GraphErrCycle, -1)
		return -1
	}
	b.loading = append(b.loading, ref.ImportPath)
	pkg := LoadPackage(b.module, b.stdRoot, ref, b.files)
	if !pkg.Ok {
		b.graph.Packages = append(b.graph.Packages, pkg)
		b.graph = graphFail(b.graph, GraphErrPackage, len(b.graph.Packages)-1)
		b.loading = b.loading[:len(b.loading)-1]
		return -1
	}
	for i := 0; i < len(pkg.Imports); i++ {
		imp := pkg.Imports[i]
		if imp.Kind == PackageInModule || imp.Kind == PackageStandard {
			b.load(imp)
			if !b.graph.Ok {
				b.loading = b.loading[:len(b.loading)-1]
				return -1
			}
		} else if !imp.Ok {
			b.graph.Packages = append(b.graph.Packages, pkg)
			b.graph = graphFail(b.graph, GraphErrPackage, len(b.graph.Packages)-1)
			b.loading = b.loading[:len(b.loading)-1]
			return -1
		}
	}
	b.loading = b.loading[:len(b.loading)-1]
	loaded = findLoadedPackage(b.graph.Packages, ref.ImportPath)
	if loaded >= 0 {
		return loaded
	}
	b.graph.Packages = append(b.graph.Packages, pkg)
	return len(b.graph.Packages) - 1
}

func packageFail(pkg Package, err int, file int, imp int) Package {
	pkg.Ok = false
	pkg.Error = err
	pkg.ErrorFile = file
	pkg.ErrorImport = imp
	return pkg
}

func graphFail(graph Graph, err int, pkg int) Graph {
	graph.Ok = false
	graph.Error = err
	graph.ErrorPackage = pkg
	return graph
}

func selectPackageFiles(dir string, files []SourceFile) []SourceFile {
	dir = CleanPath(dir)
	var selected []SourceFile
	for i := 0; i < len(files); i++ {
		path := CleanPath(files[i].Path)
		if !isGoSourceFile(path) {
			continue
		}
		if DirPath(path) != dir {
			continue
		}
		selected = append(selected, SourceFile{Path: path, Src: files[i].Src})
	}
	sortSourceFiles(selected)
	return selected
}

func sortSourceFiles(files []SourceFile) {
	for i := 1; i < len(files); i++ {
		item := files[i]
		j := i - 1
		for j >= 0 && stringAfter(files[j].Path, item.Path) {
			files[j+1] = files[j]
			j--
		}
		files[j+1] = item
	}
}

func stringAfter(left string, right string) bool {
	return stringBefore(right, left)
}

func stringBefore(left string, right string) bool {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for i := 0; i < limit; i++ {
		if left[i] < right[i] {
			return true
		}
		if left[i] > right[i] {
			return false
		}
	}
	return len(left) < len(right)
}

func appendImport(imports []PackageRef, ref PackageRef) []PackageRef {
	for i := 0; i < len(imports); i++ {
		if imports[i].ImportPath == ref.ImportPath {
			return imports
		}
	}
	return append(imports, ref)
}

func findLoadedPackage(packages []Package, importPath string) int {
	for i := 0; i < len(packages); i++ {
		if packages[i].Ref.ImportPath == importPath {
			return i
		}
	}
	return -1
}

func findString(items []string, item string) int {
	for i := 0; i < len(items); i++ {
		if items[i] == item {
			return i
		}
	}
	return -1
}

func DirPath(path string) string {
	path = CleanPath(path)
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			if i == 0 {
				return "/"
			}
			return path[:i]
		}
	}
	return "."
}

func BasePath(path string) string {
	path = CleanPath(path)
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

func isGoSourceFile(path string) bool {
	base := BasePath(path)
	return stringHasSuffix(base, ".go") && !stringHasSuffix(base, "_test.go")
}

func stringHasSuffix(text string, suffix string) bool {
	if len(suffix) > len(text) {
		return false
	}
	off := len(text) - len(suffix)
	for i := 0; i < len(suffix); i++ {
		if text[off+i] != suffix[i] {
			return false
		}
	}
	return true
}
