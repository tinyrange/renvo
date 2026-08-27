package load

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/c11"
	"renvo.dev/internal/syntax"
)

const (
	PackageOK = iota
	PackageErrRef
	PackageErrNoFiles
	PackageErrParse
	PackageErrName
	PackageErrImport
	PackageErrC11
)

const (
	GraphOK = iota
	GraphErrRoot
	GraphErrPackage
	GraphErrCycle
)

type SourceFile struct {
	Path              string
	Src               []byte
	CPrelude          []byte
	CObject           bool
	CCompiler         bool
	CDataModel        int
	CFunctionSections bool
	CDataSections     bool
	CShortWChar       bool
	CUnsignedChar     bool
	CKernelCodeModel  bool
	COptimize         bool
	ArenaStart        int
	ArenaEnd          int
}

type ParsedFile struct {
	Path       string
	Src        []byte
	File       syntax.File
	C          bool
	ArenaStart int
	ArenaEnd   int
}

type AssemblyFile struct {
	Path       string
	Src        []byte
	ArenaStart int
	ArenaEnd   int
}

type Package struct {
	Ref            PackageRef
	Name           string
	Files          []ParsedFile
	Assemblies     []AssemblyFile
	Imports        []PackageRef
	Ok             bool
	Error          int
	ErrorFile      int
	ErrorImport    int
	ErrorOffset    int
	C11Error       int
	ExplicitC      bool
	CoreArenaStart int
	CoreArenaEnd   int
}

type Graph struct {
	Module       Module
	Root         string
	Packages     []Package
	Ok           bool
	Error        int
	ErrorPackage int
	ErrorPath    string
	ErrorOffset  int
}

func LoadGraph(module Module, stdRoot string, workDir string, arg string, files []SourceFile) Graph {
	return LoadGraphWithDependencies(module, stdRoot, workDir, arg, nil, files)
}

func LoadGraphWithDependencies(module Module, stdRoot string, workDir string, arg string, dependencies []ModuleDependency, files []SourceFile) Graph {
	ref := ResolvePackageArg(module, workDir, arg)
	if !ref.Ok {
		return Graph{Module: module, Ok: false, Error: GraphErrRoot, ErrorPackage: -1}
	}
	return loadGraphFromRoot(module, stdRoot, ref, dependencies, files)
}

func LoadGraphFromRoot(module Module, stdRoot string, root PackageRef, files []SourceFile) Graph {
	return loadGraphFromRoot(module, stdRoot, root, nil, files)
}

func loadGraphFromRoot(module Module, stdRoot string, root PackageRef, dependencies []ModuleDependency, files []SourceFile) Graph {
	var builder graphBuilder
	builder.module = module
	builder.stdRoot = CleanPath(stdRoot)
	builder.dependencies = dependencies
	builder.files = files
	builder.graph = Graph{Module: module, Root: root.ImportPath, Ok: true, Error: GraphOK, ErrorPackage: -1}
	builder.load(root)
	if !builder.graph.Ok {
		return builder.graph
	}
	return builder.graph
}

func LoadPackage(module Module, stdRoot string, ref PackageRef, files []SourceFile) Package {
	return loadPackage(module, stdRoot, ref, nil, files, false)
}

func loadPackage(module Module, stdRoot string, ref PackageRef, dependencies []ModuleDependency, files []SourceFile, root bool) Package {
	pkg := Package{
		Ref:         ref,
		Ok:          true,
		Error:       PackageOK,
		ErrorFile:   -1,
		ErrorImport: -1,
		ErrorOffset: -1,
	}
	if !ref.Ok || ref.Dir == "" {
		return packageFail(pkg, PackageErrRef, -1, -1)
	}
	selected := selectPackageFiles(ref.Dir, files)
	if len(selected) == 0 {
		return packageFail(pkg, PackageErrNoFiles, -1, -1)
	}
	code := selected[:0]
	for i := 0; i < len(selected); i++ {
		if !stringHasSuffix(selected[i].Path, ".rtgasm") {
			code = append(code, selected[i])
			continue
		}
		pkg.Assemblies = append(pkg.Assemblies, AssemblyFile{
			Path: selected[i].Path, Src: selected[i].Src,
			ArenaStart: selected[i].ArenaStart, ArenaEnd: selected[i].ArenaEnd,
		})
	}
	selected = code
	if len(selected) == 0 {
		return packageFail(pkg, PackageErrNoFiles, -1, -1)
	}
	hasC := false
	hasGo := false
	cCompiler := false
	for i := 0; i < len(selected); i++ {
		if isCSourceFile(selected[i].Path) {
			hasC = true
			cCompiler = cCompiler || selected[i].CCompiler
		} else if isGoSourceFile(selected[i].Path) {
			hasGo = true
		}
	}
	// Ordinary mixed packages use the explicit import "C" boundary. Explicit
	// C compiler invocations, including #pragma go, intentionally retain their
	// C-first shared-name contract.
	pkg.ExplicitC = hasC && hasGo && !cCompiler
	var parsedGo []syntax.File
	var goExportDefs []goExportDefinition
	var goPreambleErrors []c11.Result
	var goPreambleOffsets []int
	if hasC {
		parsedGo = make([]syntax.File, len(selected))
		goPreambleErrors = make([]c11.Result, len(selected))
		goPreambleOffsets = make([]int, len(selected))
		cgoDataModel := c11.DataModelLP64
		for i := 0; i < len(selected); i++ {
			if isCSourceFile(selected[i].Path) && selected[i].CDataModel != c11.DataModelInvalid {
				cgoDataModel = selected[i].CDataModel
				break
			}
		}
		for i := 0; i < len(selected); i++ {
			if !isGoSourceFile(selected[i].Path) {
				continue
			}
			parsedGo[i] = syntax.ParseFile(selected[i].Src)
			if !parsedGo[i].Ok {
				continue
			}
			name := string(syntax.TokenText(parsedGo[i].Src, parsedGo[i].Tokens[parsedGo[i].PackageName]))
			if pkg.Name == "" {
				pkg.Name = name
			} else if pkg.Name != name {
				// Preserve the established file index contract below.
				break
			}
			for j := 0; j < len(parsedGo[i].Imports); j++ {
				preamble, offset, cgoImport := syntax.CgoPreamble(parsedGo[i], parsedGo[i].Imports[j])
				if !cgoImport {
					continue
				}
				inspected := c11.InspectDeclarationsWithConfig(preamble, c11.ObjectConfig{DataModel: cgoDataModel})
				if !inspected.Ok {
					goPreambleErrors[i] = inspected
					goPreambleOffsets[i] = offset
					continue
				}
				if len(inspected.Source) > 0 {
					// Keep mixed-only declaration metadata behind the parsed EOF;
					// token offsets remain relative to the original source.
					if len(parsedGo[i].Src) == len(selected[i].Src) {
						metadata := make([]byte, len(parsedGo[i].Src))
						copy(metadata, parsedGo[i].Src)
						parsedGo[i].Src = metadata
						parsedGo[i].Src = append(parsedGo[i].Src, '\n')
					}
					parsedGo[i].Src = append(parsedGo[i].Src, inspected.Source...)
				}
			}
		}
		if pkg.Name == "" {
			pkg.Name = cPackageName(ref, root)
		}
		if pkg.ExplicitC {
			for i := 0; i < len(parsedGo); i++ {
				file := parsedGo[i]
				if !file.Ok {
					continue
				}
				for j := 0; j < len(file.Funcs); j++ {
					if name := syntax.ExportDirective(file, file.Funcs[j]); name != "" {
						goName := string(syntax.TokenText(file.Src, file.Tokens[file.Funcs[j].NameTok]))
						goExportDefs = append(goExportDefs, goExportDefinition{
							Mapping: c11.GoExport{CName: name, GoName: goName}, File: i, Func: j,
						})
					}
				}
			}
		}
	}
	for i := 0; i < len(selected); i++ {
		var parsed syntax.File
		if hasC {
			parsed = parsedGo[i]
		} else {
			parsed = syntax.ParseFile(selected[i].Src)
		}
		source := selected[i]
		if isGoSourceFile(source.Path) && goPreambleErrors != nil && !goPreambleErrors[i].Ok && goPreambleErrors[i].Error != c11.TranslateOK {
			pkg.ErrorOffset = goPreambleOffsets[i] + goPreambleErrors[i].ErrorAt
			pkg.C11Error = goPreambleErrors[i].Error
			pkg.Files = append(pkg.Files, newParsedFile(source, parsed))
			return packageFail(pkg, PackageErrC11, i, -1)
		}
		if isCSourceFile(source.Path) {
			goExports := make([]c11.GoExport, len(goExportDefs))
			for j := 0; j < len(goExportDefs); j++ {
				goExports[j] = goExportDefs[j].Mapping
			}
			var exportHeader []byte
			if cgoSourceIncludesExportHeader(source.Src) {
				dataModel := source.CDataModel
				if dataModel == c11.DataModelInvalid {
					dataModel = c11.DataModelLP64
				}
				exportHeader = cgoExportHeader(parsedGo, goExportDefs, dataModel)
			}
			var translated c11.Result
			if source.CObject {
				translated = c11.TranslateObjectWithConfig(pkg.Name, source.Src, source.CPrelude, c11.ObjectConfig{
					DataModel: source.CDataModel, FunctionSections: source.CFunctionSections,
					DataSections:       source.CDataSections,
					ShortWChar:         source.CShortWChar,
					UnsignedChar:       source.CUnsignedChar,
					KernelCodeModel:    source.CKernelCodeModel,
					PruneUnusedStatics: source.COptimize,
					IsolateGoBuiltins:  source.CCompiler,
					GoExports:          goExports,
				})
			} else {
				dataModel := source.CDataModel
				if dataModel == c11.DataModelInvalid {
					dataModel = c11.DataModelLP64
				}
				translated = c11.TranslateWithPreludeConfig(pkg.Name, source.Src, exportHeader, c11.ObjectConfig{
					DataModel: dataModel, ShortWChar: source.CShortWChar, UnsignedChar: source.CUnsignedChar,
					PruneUnusedStatics: source.COptimize, IsolateGoBuiltins: source.CCompiler, GoExports: goExports,
				})
			}
			if !translated.Ok {
				pkg.ErrorOffset = translated.ErrorAt
				pkg.C11Error = translated.Error
				pkg.Files = append(pkg.Files, ParsedFile{Path: source.Path, Src: source.Src, ArenaStart: source.ArenaStart, ArenaEnd: source.ArenaEnd})
				return packageFail(pkg, PackageErrC11, i, -1)
			}
			source.Src = translated.Source
			parsed = syntax.ParseFile(source.Src)
		}
		if !parsed.Ok {
			pkg.Files = append(pkg.Files, newParsedFile(source, parsed))
			return packageFail(pkg, PackageErrParse, i, -1)
		}
		name := string(syntax.TokenText(parsed.Src, parsed.Tokens[parsed.PackageName]))
		if pkg.Name == "" {
			pkg.Name = name
		} else if pkg.Name != name {
			pkg.Files = append(pkg.Files, newParsedFile(source, parsed))
			return packageFail(pkg, PackageErrName, i, -1)
		}
		refs := FileImportsWithDependencies(module, stdRoot, dependencies, parsed)
		for j := 0; j < len(refs); j++ {
			pkg.Imports = appendImport(pkg.Imports, refs[j])
			if !refs[j].Ok {
				pkg.Files = append(pkg.Files, newParsedFile(source, parsed))
				return packageFail(pkg, PackageErrImport, i, len(pkg.Imports)-1)
			}
		}
		pkg.Files = append(pkg.Files, newParsedFile(source, parsed))
	}
	return pkg
}

func newParsedFile(source SourceFile, file syntax.File) ParsedFile {
	return ParsedFile{
		Path:       source.Path,
		Src:        source.Src,
		File:       file,
		C:          isCSourceFile(source.Path),
		ArenaStart: source.ArenaStart,
		ArenaEnd:   source.ArenaEnd,
	}
}

type graphBuilder struct {
	module       Module
	stdRoot      string
	dependencies []ModuleDependency
	files        []SourceFile
	loading      []string
	graph        Graph
}

func (b *graphBuilder) load(ref PackageRef) int {
	if !b.graph.Ok {
		return -1
	}
	if ref.Kind != PackageInModule && ref.Kind != PackageStandard && ref.Kind != PackageDependency {
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
	packageStart := arena.Mark()
	pkg := loadPackage(b.module, b.stdRoot, ref, b.dependencies, b.files, ref.ImportPath == b.graph.Root)
	pkg.CoreArenaStart = packageStart
	pkg.CoreArenaEnd = arena.Mark()
	if !pkg.Ok {
		b.graph.Packages = append(b.graph.Packages, pkg)
		b.graph = graphFail(b.graph, GraphErrPackage, len(b.graph.Packages)-1)
		b.loading = b.loading[:len(b.loading)-1]
		return -1
	}
	for i := 0; i < len(pkg.Imports); i++ {
		imp := pkg.Imports[i]
		if imp.Kind == PackageInModule || imp.Kind == PackageStandard || imp.Kind == PackageDependency {
			b.load(imp)
			if !b.graph.Ok {
				if b.graph.Error == GraphErrCycle && b.graph.ErrorPath == "" {
					b.graph.ErrorPath, b.graph.ErrorOffset = packageImportLocation(pkg, imp.ImportPath)
				}
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

func packageImportLocation(pkg Package, importPath string) (string, int) {
	for i := 0; i < len(pkg.Files); i++ {
		file := pkg.Files[i]
		for j := 0; j < len(file.File.Imports); j++ {
			tokenIndex := file.File.Imports[j].PathTok
			if tokenIndex < 0 || tokenIndex >= len(file.File.Tokens) {
				continue
			}
			path, ok := syntax.StringLiteralValue(file.Src, file.File.Tokens[tokenIndex])
			if ok && path == importPath {
				return file.Path, syntax.TokenStart(file.File.Tokens[tokenIndex])
			}
		}
	}
	return "", 0
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
		if !isFrontendSourceFile(path) {
			continue
		}
		if DirPath(path) != dir {
			continue
		}
		selected = append(selected, SourceFile{Path: path, Src: files[i].Src, CPrelude: files[i].CPrelude,
			CObject: files[i].CObject, CCompiler: files[i].CCompiler, CDataModel: files[i].CDataModel, CFunctionSections: files[i].CFunctionSections,
			CDataSections: files[i].CDataSections, CShortWChar: files[i].CShortWChar,
			CUnsignedChar:    files[i].CUnsignedChar,
			CKernelCodeModel: files[i].CKernelCodeModel, COptimize: files[i].COptimize,
			ArenaStart: files[i].ArenaStart, ArenaEnd: files[i].ArenaEnd})
	}
	sortSourceFiles(selected)
	return selected
}

func sortSourceFiles(files []SourceFile) {
	for root := len(files)/2 - 1; root >= 0; root-- {
		siftSourceFiles(files, root, len(files))
	}
	for end := len(files) - 1; end > 0; end-- {
		files[0], files[end] = files[end], files[0]
		siftSourceFiles(files, 0, end)
	}
}

func siftSourceFiles(files []SourceFile, root int, end int) {
	for {
		child := root*2 + 1
		if child >= end {
			return
		}
		if child+1 < end && stringBefore(files[child].Path, files[child+1].Path) {
			child++
		}
		if !stringBefore(files[root].Path, files[child].Path) {
			return
		}
		files[root], files[child] = files[child], files[root]
		root = child
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

func isCSourceFile(path string) bool {
	base := BasePath(path)
	return stringHasSuffix(base, ".c") && !stringHasSuffix(base, "_test.c")
}

func isRTGAsmSourceFile(path string) bool {
	base := BasePath(path)
	return stringHasSuffix(base, ".rtgasm") && !stringHasSuffix(base, "_test.rtgasm")
}

func isFrontendSourceFile(path string) bool {
	return isGoSourceFile(path) || isCSourceFile(path) || isRTGAsmSourceFile(path)
}

func cPackageName(ref PackageRef, root bool) string {
	if root {
		return "main"
	}
	name := BasePath(ref.Dir)
	if name == "" || !cPackageIdentStart(name[0]) {
		return "c"
	}
	out := make([]byte, len(name))
	for i := 0; i < len(name); i++ {
		if i == 0 && cPackageIdentStart(name[i]) || i > 0 && cPackageIdentPart(name[i]) {
			out[i] = name[i]
		} else {
			out[i] = '_'
		}
	}
	return string(out)
}

func cPackageIdentStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func cPackageIdentPart(c byte) bool {
	return cPackageIdentStart(c) || c >= '0' && c <= '9'
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

type goExportDefinition struct {
	Mapping c11.GoExport
	File    int
	Func    int
}

func cgoExportHeader(files []syntax.File, definitions []goExportDefinition, dataModel int) []byte {
	out := "/* Code generated by Renvo cgo; DO NOT EDIT. */\n"
	intType := "long long"
	if dataModel == c11.DataModelILP32 {
		intType = "int"
	}
	for i := 0; i < len(definitions); i++ {
		definition := definitions[i]
		file := files[definition.File]
		fn := file.Funcs[definition.Func]
		out += "extern "
		if fn.ResultStart < fn.ResultEnd {
			out += intType
		} else {
			out += "void"
		}
		out += " " + definition.Mapping.CName + "("
		start, end := fn.ParamsStart+1, fn.ParamsEnd-1
		if start >= end {
			out += "void"
		} else {
			out += intType
			for ; start < end; start++ {
				if file.Tokens[start].KindLine>>syntax.TokenOperatorCharShift&syntax.TokenOperatorCharMask == ',' {
					out += ", " + intType
				}
			}
		}
		out += ");\n"
	}
	return []byte(out)
}

func cgoSourceIncludesExportHeader(src []byte) bool {
	const name = "_cgo_export.h"
	for i := 0; i+len(name) <= len(src); i++ {
		if bytesEqual(src, i, i+len(name), name) {
			return true
		}
	}
	return false
}
