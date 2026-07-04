package load

import "j5.nz/rtg/rtg/internal/syntax"

const (
	ModuleOK = iota
	ModuleErrMissing
	ModuleErrPath
)

const (
	PackageInvalid = iota
	PackageInModule
	PackageStandard
	PackageUnsupported
)

const (
	ResolveOK = iota
	ResolveErrModule
	ResolveErrImport
	ResolveErrOutsideModule
	ResolveErrUnsupported
)

type Module struct {
	Root        string
	Path        string
	Ok          bool
	Error       int
	ErrorOffset int
}

type PackageRef struct {
	Kind       int
	ImportPath string
	Dir        string
	Ok         bool
	Error      int
}

func ParseModule(root string, src []byte) Module {
	i := 0
	for i < len(src) {
		i = skipGoModSpace(src, i)
		if i >= len(src) {
			break
		}
		start := i
		for i < len(src) && isGoModWord(src[i]) {
			i++
		}
		if start == i {
			i = skipGoModLine(src, i)
			continue
		}
		if bytesEqual(src, start, i, "module") {
			i = skipGoModHorizontal(src, i)
			path, _, ok := parseModulePath(src, i)
			if !ok || len(path) == 0 {
				return Module{Root: CleanPath(root), Ok: false, Error: ModuleErrPath, ErrorOffset: i}
			}
			return Module{Root: CleanPath(root), Path: path, Ok: true, Error: ModuleOK, ErrorOffset: -1}
		}
		i = skipGoModLine(src, i)
	}
	return Module{Root: CleanPath(root), Ok: false, Error: ModuleErrMissing, ErrorOffset: len(src)}
}

func ResolveImport(module Module, stdRoot string, importPath string) PackageRef {
	if !module.Ok {
		return PackageRef{Kind: PackageInvalid, ImportPath: importPath, Ok: false, Error: ResolveErrModule}
	}
	if importPath == "" || isRelativeImport(importPath) {
		return PackageRef{Kind: PackageInvalid, ImportPath: importPath, Ok: false, Error: ResolveErrImport}
	}
	if importPath == module.Path {
		return PackageRef{Kind: PackageInModule, ImportPath: importPath, Dir: module.Root, Ok: true, Error: ResolveOK}
	}
	if hasImportPrefix(importPath, module.Path) {
		suffix := importPath[len(module.Path)+1:]
		return PackageRef{Kind: PackageInModule, ImportPath: importPath, Dir: JoinPath(module.Root, suffix), Ok: true, Error: ResolveOK}
	}
	if IsStandardImport(importPath) {
		return PackageRef{Kind: PackageStandard, ImportPath: importPath, Dir: JoinPath(stdRoot, importPath), Ok: true, Error: ResolveOK}
	}
	return PackageRef{Kind: PackageUnsupported, ImportPath: importPath, Ok: false, Error: ResolveErrUnsupported}
}

func ResolvePackageArg(module Module, workDir string, arg string) PackageRef {
	if !module.Ok {
		return PackageRef{Kind: PackageInvalid, ImportPath: arg, Ok: false, Error: ResolveErrModule}
	}
	if arg == "" {
		return PackageRef{Kind: PackageInvalid, ImportPath: arg, Ok: false, Error: ResolveErrImport}
	}
	if !isPathArg(arg) {
		return ResolveImport(module, "", arg)
	}
	dir := arg
	if !isAbsPath(arg) {
		dir = JoinPath(workDir, arg)
	} else {
		dir = CleanPath(arg)
	}
	rel, ok := RelPath(module.Root, dir)
	if !ok {
		return PackageRef{Kind: PackageInvalid, Dir: dir, Ok: false, Error: ResolveErrOutsideModule}
	}
	importPath := module.Path
	if rel != "." {
		importPath = module.Path + "/" + rel
	}
	return PackageRef{Kind: PackageInModule, ImportPath: importPath, Dir: dir, Ok: true, Error: ResolveOK}
}

func FileImports(module Module, stdRoot string, file syntax.File) []PackageRef {
	out := make([]PackageRef, 0, len(file.Imports))
	for i := 0; i < len(file.Imports); i++ {
		tok := file.Imports[i].PathTok
		if tok < 0 || tok >= len(file.Tokens) {
			out = append(out, PackageRef{Kind: PackageInvalid, Ok: false, Error: ResolveErrImport})
			continue
		}
		path, ok := syntax.StringLiteralValue(file.Src, file.Tokens[tok])
		if !ok {
			out = append(out, PackageRef{Kind: PackageInvalid, Ok: false, Error: ResolveErrImport})
			continue
		}
		out = append(out, ResolveImport(module, stdRoot, path))
	}
	return out
}

func CleanPath(path string) string {
	if path == "" {
		return "."
	}
	absolute := path[0] == '/'
	parts := make([]string, 0, 8)
	i := 0
	for i <= len(path) {
		start := i
		for i < len(path) && path[i] != '/' {
			i++
		}
		part := path[start:i]
		if part == "" || part == "." {
		} else if part == ".." {
			if len(parts) > 0 && parts[len(parts)-1] != ".." {
				parts = parts[:len(parts)-1]
			} else if !absolute {
				parts = append(parts, part)
			}
		} else {
			parts = append(parts, part)
		}
		if i >= len(path) {
			break
		}
		i++
	}
	if len(parts) == 0 {
		if absolute {
			return "/"
		}
		return "."
	}
	out := make([]byte, 0, len(path))
	if absolute {
		out = append(out, '/')
	}
	for i = 0; i < len(parts); i++ {
		if i > 0 {
			out = append(out, '/')
		}
		out = append(out, parts[i]...)
	}
	return string(out)
}

func JoinPath(base string, elem string) string {
	if elem == "" || elem == "." {
		return CleanPath(base)
	}
	if isAbsPath(elem) {
		return CleanPath(elem)
	}
	if base == "" || base == "." {
		return CleanPath(elem)
	}
	return CleanPath(base + "/" + elem)
}

func RelPath(root string, path string) (string, bool) {
	root = CleanPath(root)
	path = CleanPath(path)
	if root == path {
		return ".", true
	}
	if !hasPathPrefix(path, root) {
		return "", false
	}
	if root == "/" {
		return path[1:], true
	}
	return path[len(root)+1:], true
}

func IsStandardImport(importPath string) bool {
	if importPath == "" || importPath[0] == '.' || importPath[0] == '/' {
		return false
	}
	for i := 0; i < len(importPath); i++ {
		c := importPath[i]
		if c == '.' {
			return false
		}
		if c == '/' {
			return true
		}
	}
	return true
}

func parseModulePath(src []byte, start int) (string, int, bool) {
	if start >= len(src) || src[start] == '\n' || src[start] == '\r' {
		return "", start, false
	}
	if src[start] == '`' || src[start] == '"' {
		return parseGoModString(src, start)
	}
	i := start
	for i < len(src) && !isGoModSpace(src[i]) {
		if src[i] == '/' && i+1 < len(src) && (src[i+1] == '/' || src[i+1] == '*') {
			break
		}
		i++
	}
	if i == start {
		return "", start, false
	}
	return string(src[start:i]), i, true
}

func parseGoModString(src []byte, start int) (string, int, bool) {
	quote := src[start]
	i := start + 1
	out := make([]byte, 0, 16)
	for i < len(src) {
		c := src[i]
		if c == quote {
			return string(out), i + 1, true
		}
		if quote == '"' && c == '\\' {
			i++
			if i >= len(src) {
				return "", start, false
			}
			c = src[i]
			if c == '"' || c == '\\' {
				out = append(out, c)
			} else {
				return "", start, false
			}
			i++
			continue
		}
		if c == '\n' || c == '\r' {
			return "", start, false
		}
		out = append(out, c)
		i++
	}
	return "", start, false
}

func skipGoModSpace(src []byte, i int) int {
	for i < len(src) {
		c := src[i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			i++
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '/' {
			i += 2
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '*' {
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			if i+1 < len(src) {
				i += 2
			}
			continue
		}
		break
	}
	return i
}

func skipGoModHorizontal(src []byte, i int) int {
	for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
		i++
	}
	return i
}

func skipGoModLine(src []byte, i int) int {
	for i < len(src) && src[i] != '\n' {
		i++
	}
	return i
}

func isGoModWord(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isGoModSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

func bytesEqual(src []byte, start int, end int, text string) bool {
	if end-start != len(text) {
		return false
	}
	for i := 0; i < len(text); i++ {
		if src[start+i] != text[i] {
			return false
		}
	}
	return true
}

func isAbsPath(path string) bool {
	return len(path) > 0 && path[0] == '/'
}

func isPathArg(arg string) bool {
	if arg == "." || arg == ".." || isAbsPath(arg) {
		return true
	}
	if len(arg) >= 2 && arg[0] == '.' && arg[1] == '/' {
		return true
	}
	if len(arg) >= 3 && arg[0] == '.' && arg[1] == '.' && arg[2] == '/' {
		return true
	}
	return false
}

func isRelativeImport(path string) bool {
	if path == "." || path == ".." {
		return true
	}
	if len(path) >= 2 && path[0] == '.' && path[1] == '/' {
		return true
	}
	if len(path) >= 3 && path[0] == '.' && path[1] == '.' && path[2] == '/' {
		return true
	}
	return false
}

func hasImportPrefix(path string, prefix string) bool {
	return len(path) > len(prefix) && stringHasPrefix(path, prefix) && path[len(prefix)] == '/'
}

func hasPathPrefix(path string, prefix string) bool {
	if prefix == "/" {
		return len(path) > 0 && path[0] == '/'
	}
	return len(path) > len(prefix) && stringHasPrefix(path, prefix) && path[len(prefix)] == '/'
}

func stringHasPrefix(path string, prefix string) bool {
	if len(prefix) > len(path) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if path[i] != prefix[i] {
			return false
		}
	}
	return true
}
