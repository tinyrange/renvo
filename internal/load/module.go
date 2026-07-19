package load

import "renvo.dev/internal/syntax"

const (
	ModuleOK = iota
	ModuleErrMissing
	ModuleErrPath
	ModuleErrDirective
)

const (
	PackageInvalid = iota
	PackageInModule
	PackageStandard
	PackageUnsupported
	PackageDependency
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

type ModuleConfig struct {
	Requires []ModuleVersion
	Replaces []ModuleReplace
	Excludes []ModuleVersion
}

type ModuleVersion struct {
	Path    string
	Version string
}

type ModuleReplace struct {
	OldPath    string
	OldVersion string
	NewPath    string
	NewVersion string
	Local      bool
}

// ModuleDependency maps the import path used by source code to an already
// available, read-only source tree. It is populated by source collection and
// deliberately contains no network location.
type ModuleDependency struct {
	Path string
	Root string
}

type PackageRef struct {
	Kind       int
	ImportPath string
	Dir        string
	Ok         bool
	Error      int
}

func ParseModule(root string, src []byte) Module {
	config := &ModuleConfig{}
	return ParseModuleConfig(root, src, config)
}

func ParseModuleConfig(root string, src []byte, config *ModuleConfig) Module {
	module := Module{Root: CleanPath(root), Ok: true, Error: ModuleOK, ErrorOffset: -1}
	block := ""
	blockOffset := -1
	i := 0
	for {
		i = skipGoModSpace(src, i)
		if i >= len(src) {
			break
		}
		start := i
		directive := block
		if block != "" && src[i] == ')' {
			block, blockOffset = "", -1
			i++
			continue
		}
		if block == "" {
			for i < len(src) && isGoModWord(src[i]) {
				i++
			}
			if i == start {
				i = skipGoModLine(src, i)
				continue
			}
			directive = string(src[start:i])
			i = skipGoModHorizontal(src, i)
			if (directive == "require" || directive == "replace" || directive == "exclude") && i < len(src) && src[i] == '(' {
				block, blockOffset = directive, start
				i++
				continue
			}
		}
		if directive == "module" {
			path, next, ok := parseModulePath(src, i)
			if !ok || path == "" || module.Path != "" {
				return moduleParseFail(module, ModuleErrPath, start)
			}
			module.Path, i = path, next
		} else if directive == "require" || directive == "exclude" {
			path, next, ok := parseModulePath(src, i)
			if !ok {
				return moduleParseFail(module, ModuleErrDirective, start)
			}
			version, next, ok := parseModulePath(src, skipGoModSpace(src, next))
			item := ModuleVersion{Path: path, Version: version}
			valid := ok
			if directive == "require" {
				valid = valid && appendModuleVersion(&config.Requires, item, false)
			} else {
				valid = valid && appendModuleVersion(&config.Excludes, item, true)
			}
			if !valid {
				return moduleParseFail(module, ModuleErrDirective, start)
			}
			i = next
		} else if directive == "replace" {
			oldPath, next, ok := parseModulePath(src, i)
			if !ok {
				return moduleParseFail(module, ModuleErrDirective, start)
			}
			replacement := ModuleReplace{OldPath: oldPath}
			word, next, ok := parseModulePath(src, skipGoModSpace(src, next))
			if word != "=>" {
				replacement.OldVersion = word
				word, next, ok = parseModulePath(src, skipGoModSpace(src, next))
			}
			if !ok || word != "=>" {
				return moduleParseFail(module, ModuleErrDirective, start)
			}
			replacement.NewPath, next, ok = parseModulePath(src, skipGoModSpace(src, next))
			replacement.Local = isLocalModulePath(replacement.NewPath)
			if ok && !replacement.Local {
				replacement.NewVersion, next, ok = parseModulePath(src, skipGoModSpace(src, next))
			}
			if !ok || !appendModuleReplace(config, replacement) {
				return moduleParseFail(module, ModuleErrDirective, start)
			}
			i = next
		} else {
			i = skipGoModLine(src, i)
			continue
		}
		if block == "" {
			i = skipGoModLine(src, i)
		}
	}
	if block != "" {
		return moduleParseFail(module, ModuleErrDirective, blockOffset)
	}
	if module.Path == "" {
		return moduleParseFail(module, ModuleErrMissing, len(src))
	}
	return module
}

func moduleParseFail(module Module, err int, offset int) Module {
	module.Ok = false
	module.Error = err
	module.ErrorOffset = offset
	return module
}

func appendModuleReplace(config *ModuleConfig, replacement ModuleReplace) bool {
	if replacement.OldPath == "" || replacement.NewPath == "" || replacement.Local == (replacement.NewVersion != "") ||
		(replacement.OldVersion != "" && !validModuleVersion(replacement.OldVersion)) ||
		(replacement.NewVersion != "" && !validModuleVersion(replacement.NewVersion)) {
		return false
	}
	for i := 0; i < len(config.Replaces); i++ {
		old := config.Replaces[i]
		if old.OldPath == replacement.OldPath && old.OldVersion == replacement.OldVersion {
			return old.NewPath == replacement.NewPath && old.NewVersion == replacement.NewVersion
		}
	}
	config.Replaces = append(config.Replaces, replacement)
	return true
}

func appendModuleVersion(items *[]ModuleVersion, item ModuleVersion, allowMultiple bool) bool {
	if item.Path == "" || !validModuleVersion(item.Version) {
		return false
	}
	values := *items
	for i := 0; i < len(values); i++ {
		old := values[i]
		if old.Path == item.Path {
			if old.Version == item.Version {
				return true
			}
			if !allowMultiple {
				return false
			}
		}
	}
	*items = append(values, item)
	return true
}

func validModuleVersion(version string) bool {
	if len(version) < 6 || version[0] != 'v' {
		return false
	}
	dots := 0
	componentDigit := false
	for i := 1; i < len(version); i++ {
		c := version[i]
		if c >= '0' && c <= '9' {
			componentDigit = true
			continue
		}
		if c == '.' && dots < 2 && componentDigit {
			dots++
			componentDigit = false
			continue
		}
		if (c == '-' || c == '+') && dots == 2 && componentDigit && i+1 < len(version) {
			plusSeen := c == '+'
			for j := i + 1; j < len(version); j++ {
				suffix := version[j]
				if (suffix >= 'a' && suffix <= 'z') || (suffix >= 'A' && suffix <= 'Z') || (suffix >= '0' && suffix <= '9') || suffix == '.' || suffix == '-' {
					continue
				}
				if suffix == '+' && !plusSeen && j+1 < len(version) {
					plusSeen = true
					continue
				}
				return false
			}
			return true
		}
		return false
	}
	return dots == 2 && componentDigit
}

func isLocalModulePath(path string) bool {
	return path == "." || path == ".." || (len(path) >= 2 && path[0] == '.' && path[1] == '/') || (len(path) >= 3 && path[0] == '.' && path[1] == '.' && path[2] == '/') || (len(path) > 0 && (path[0] == '/' || path[0] == '\\')) || (len(path) >= 3 && path[1] == ':' && (path[2] == '/' || path[2] == '\\'))
}

func ResolveImport(module Module, stdRoot string, importPath string) PackageRef {
	return ResolveImportWithDependencies(module, stdRoot, importPath, nil)
}

func ResolveImportWithDependencies(module Module, stdRoot string, importPath string, dependencies []ModuleDependency) PackageRef {
	if !module.Ok {
		return PackageRef{Kind: PackageInvalid, ImportPath: importPath, Ok: false, Error: ResolveErrModule}
	}
	if importPath == "" || isRelativeImport(importPath) {
		return PackageRef{Kind: PackageInvalid, ImportPath: importPath, Ok: false, Error: ResolveErrImport}
	}
	if IsStandardImport(importPath) {
		return PackageRef{Kind: PackageStandard, ImportPath: importPath, Dir: JoinPath(stdRoot, importPath), Ok: true, Error: ResolveOK}
	}
	bestPath := ""
	bestRoot := ""
	bestKind := PackageUnsupported
	if importPath == module.Path || HasImportPrefix(importPath, module.Path) {
		bestPath, bestRoot, bestKind = module.Path, module.Root, PackageInModule
	}
	for i := 0; i < len(dependencies); i++ {
		dependency := dependencies[i]
		if (importPath == dependency.Path || HasImportPrefix(importPath, dependency.Path)) && len(dependency.Path) > len(bestPath) {
			bestPath, bestRoot, bestKind = dependency.Path, dependency.Root, PackageDependency
		}
	}
	if bestPath != "" {
		dir := bestRoot
		if importPath != bestPath {
			dir = JoinPath(bestRoot, importPath[len(bestPath)+1:])
		}
		return PackageRef{Kind: bestKind, ImportPath: importPath, Dir: dir, Ok: true, Error: ResolveOK}
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
	return FileImportsWithDependencies(module, stdRoot, nil, file)
}

func FileImportsWithDependencies(module Module, stdRoot string, dependencies []ModuleDependency, file syntax.File) []PackageRef {
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
		out = append(out, ResolveImportWithDependencies(module, stdRoot, path, dependencies))
	}
	return out
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

func HasImportPrefix(path string, prefix string) bool {
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
