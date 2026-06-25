package load

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"j5.nz/rtg/rtg/mod"
	targetpkg "j5.nz/rtg/rtg/target"
)

type File struct {
	Path     string
	UnitPath string
	Source   []byte
}

type Package struct {
	ImportPath string
	Dir        string
	Name       string
	Files      []File
	Imports    []string
}

type Graph struct {
	Module   mod.Module
	Packages []Package
}

type Options struct {
	StdRoot string
	Target  string
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
		opts.StdRoot = defaultStdRoot(module.Root)
	}
	if opts.Target == "" {
		opts.Target = targetpkg.Default()
	}
	g := &Graph{Module: module}
	seen := map[string]bool{}
	fileEntries := map[string][]string{}
	for _, entry := range entries {
		dir, files, err := entryInput(entry)
		if err != nil {
			return nil, err
		}
		if len(files) > 0 {
			fileEntries[dir] = append(fileEntries[dir], files...)
			continue
		}
		if err := loadPackageRecursive(g, opts, seen, dir); err != nil {
			return nil, err
		}
	}
	var fileDirs []string
	for dir := range fileEntries {
		fileDirs = append(fileDirs, dir)
	}
	sort.Strings(fileDirs)
	for _, dir := range fileDirs {
		files := fileEntries[dir]
		if err := loadPackageFilesRecursive(g, opts, seen, dir, files); err != nil {
			return nil, err
		}
	}
	return g, nil
}

func defaultStdRoot(moduleRoot string) string {
	if env := os.Getenv("RTG_STD"); env != "" {
		return env
	}
	moduleStd := filepath.Join(moduleRoot, "rtg", "std")
	if info, err := os.Stat(moduleStd); err == nil && info.IsDir() {
		return moduleStd
	}
	if cwd, err := os.Getwd(); err == nil {
		if root, ok := findStdRootUpward(cwd); ok {
			return root
		}
	}
	return moduleStd
}

func findStdRootUpward(start string) (string, bool) {
	dir := filepath.Clean(start)
	for {
		candidate := filepath.Join(dir, "rtg", "std")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func entryInput(entry string) (string, []string, error) {
	info, err := os.Stat(entry)
	if err != nil {
		return "", nil, err
	}
	if info.IsDir() {
		dir, err := filepath.Abs(entry)
		return dir, nil, err
	}
	path, err := filepath.Abs(entry)
	if err != nil {
		return "", nil, err
	}
	if err := validateFrontendFileInput(path); err != nil {
		return "", nil, err
	}
	return filepath.Dir(path), []string{path}, nil
}

func validateFrontendFileInput(path string) error {
	name := filepath.Base(path)
	if !strings.HasSuffix(name, ".go") {
		return fmt.Errorf("%s: frontend file input must be a .go source file", path)
	}
	if strings.HasSuffix(name, "_test.go") {
		return fmt.Errorf("%s: frontend file input must not be a Go test file", path)
	}
	if strings.HasSuffix(name, ".rtg.go") {
		return fmt.Errorf("%s: frontend file input must not be an emitted RTG unit; use -link for .rtg.go files", path)
	}
	return nil
}

func loadPackageRecursive(g *Graph, opts Options, seen map[string]bool, dir string) error {
	return loadPackageRecursiveAs(g, opts, seen, dir, importPathForDir(g.Module, dir))
}

func loadPackageFilesRecursive(g *Graph, opts Options, seen map[string]bool, dir string, files []string) error {
	return loadPackageFilesRecursiveAs(g, opts, seen, dir, importPathForDir(g.Module, dir), files)
}

func loadPackageRecursiveAs(g *Graph, opts Options, seen map[string]bool, dir string, importPath string) error {
	dir = filepath.Clean(dir)
	if seen[dir] {
		return nil
	}
	seen[dir] = true
	pkg, err := readPackage(g.Module, dir, importPath, opts)
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

func loadPackageFilesRecursiveAs(g *Graph, opts Options, seen map[string]bool, dir string, importPath string, files []string) error {
	dir = filepath.Clean(dir)
	if seen[dir] {
		return nil
	}
	seen[dir] = true
	pkg, err := readPackageFiles(g.Module, dir, importPath, files)
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

func readPackage(module mod.Module, dir string, importPath string, opts Options) (Package, error) {
	files, err := goFiles(dir, opts.Target)
	if err != nil {
		return Package{}, err
	}
	if len(files) == 0 {
		return Package{}, fmt.Errorf("%s: no Go source files", dir)
	}
	return readPackageFiles(module, dir, importPath, files)
}

func readPackageFiles(module mod.Module, dir string, importPath string, files []string) (Package, error) {
	if len(files) == 0 {
		return Package{}, fmt.Errorf("%s: no Go source files", dir)
	}
	files = append([]string(nil), files...)
	sort.Strings(files)
	files = uniqueStrings(files)
	pkg := Package{Dir: dir, ImportPath: importPath}
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
		}
		pkg.Files = append(pkg.Files, File{Path: path, UnitPath: unitFilePath(module, importPath, path), Source: data})
	}
	for imp := range importSet {
		pkg.Imports = append(pkg.Imports, imp)
	}
	sort.Strings(pkg.Imports)
	return pkg, nil
}

func uniqueStrings(values []string) []string {
	var out []string
	for _, value := range values {
		if len(out) > 0 && out[len(out)-1] == value {
			continue
		}
		out = append(out, value)
	}
	return out
}

func unitFilePath(module mod.Module, importPath string, path string) string {
	if importPath == module.Path || strings.HasPrefix(importPath, module.Path+"/") {
		rel, err := filepath.Rel(module.Root, path)
		rel = filepath.ToSlash(rel)
		if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, "../") {
			return rel
		}
	}
	if importPath != "" {
		return filepath.ToSlash(filepath.Join(filepath.FromSlash(importPath), filepath.Base(path)))
	}
	return filepath.ToSlash(filepath.Base(path))
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

func goFiles(dir string, target string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	targetOS, targetArch := targetFileParts(target)
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
		if !fileNameMatchesTarget(name, targetOS, targetArch) {
			continue
		}
		path := filepath.Join(dir, name)
		ok, err := fileBuildTagsMatchTarget(path, targetOS, targetArch)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		files = append(files, path)
	}
	sort.Strings(files)
	return files, nil
}

func fileBuildTagsMatchTarget(path string, targetOS string, targetArch string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	expr, ok := leadingGoBuildExpr(string(data))
	if ok {
		return evalGoBuildExpr(expr, targetOS, targetArch), nil
	}
	lines := leadingPlusBuildLines(string(data))
	if len(lines) == 0 {
		return true, nil
	}
	return evalPlusBuildLines(lines, targetOS, targetArch), nil
}

func leadingGoBuildExpr(src string) (string, bool) {
	for len(src) > 0 {
		line := src
		next := strings.IndexByte(src, '\n')
		if next >= 0 {
			line = src[:next]
			src = src[next+1:]
		} else {
			src = ""
		}
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if strings.HasPrefix(line, "//go:build ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "//go:build ")), true
		}
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}
		return "", false
	}
	return "", false
}

func leadingPlusBuildLines(src string) []string {
	var lines []string
	for len(src) > 0 {
		line := src
		next := strings.IndexByte(src, '\n')
		if next >= 0 {
			line = src[:next]
			src = src[next+1:]
		} else {
			src = ""
		}
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if strings.HasPrefix(line, "// +build ") {
			lines = append(lines, strings.TrimSpace(strings.TrimPrefix(line, "// +build ")))
			continue
		}
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}
		break
	}
	return lines
}

func evalPlusBuildLines(lines []string, targetOS string, targetArch string) bool {
	for _, line := range lines {
		if !evalPlusBuildLine(line, targetOS, targetArch) {
			return false
		}
	}
	return true
}

func evalPlusBuildLine(line string, targetOS string, targetArch string) bool {
	options := strings.Fields(line)
	if len(options) == 0 {
		return false
	}
	for _, option := range options {
		if evalPlusBuildOption(option, targetOS, targetArch) {
			return true
		}
	}
	return false
}

func evalPlusBuildOption(option string, targetOS string, targetArch string) bool {
	terms := strings.Split(option, ",")
	if len(terms) == 0 {
		return false
	}
	for _, term := range terms {
		if term == "" {
			return false
		}
		negated := false
		if strings.HasPrefix(term, "!") {
			negated = true
			term = strings.TrimPrefix(term, "!")
			if term == "" {
				return false
			}
		}
		matches := goBuildTagMatches(term, targetOS, targetArch)
		if negated {
			matches = !matches
		}
		if !matches {
			return false
		}
	}
	return true
}

func evalGoBuildExpr(expr string, targetOS string, targetArch string) bool {
	toks := goBuildExprTokens(expr)
	pos := 0
	value, ok := parseGoBuildOr(toks, &pos, targetOS, targetArch)
	if !ok || pos != len(toks) {
		return false
	}
	return value
}

func goBuildExprTokens(expr string) []string {
	var toks []string
	for i := 0; i < len(expr); {
		c := expr[i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			i++
			continue
		}
		if c == '(' || c == ')' || c == '!' {
			toks = append(toks, string(c))
			i++
			continue
		}
		if i+1 < len(expr) && ((expr[i] == '&' && expr[i+1] == '&') || (expr[i] == '|' && expr[i+1] == '|')) {
			toks = append(toks, expr[i:i+2])
			i += 2
			continue
		}
		start := i
		for i < len(expr) && isGoBuildTagChar(expr[i]) {
			i++
		}
		if start == i {
			return nil
		}
		toks = append(toks, expr[start:i])
	}
	return toks
}

func isGoBuildTagChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '.'
}

func parseGoBuildOr(toks []string, pos *int, targetOS string, targetArch string) (bool, bool) {
	left, ok := parseGoBuildAnd(toks, pos, targetOS, targetArch)
	if !ok {
		return false, false
	}
	for *pos < len(toks) && toks[*pos] == "||" {
		*pos = *pos + 1
		right, ok := parseGoBuildAnd(toks, pos, targetOS, targetArch)
		if !ok {
			return false, false
		}
		left = left || right
	}
	return left, true
}

func parseGoBuildAnd(toks []string, pos *int, targetOS string, targetArch string) (bool, bool) {
	left, ok := parseGoBuildUnary(toks, pos, targetOS, targetArch)
	if !ok {
		return false, false
	}
	for *pos < len(toks) && toks[*pos] == "&&" {
		*pos = *pos + 1
		right, ok := parseGoBuildUnary(toks, pos, targetOS, targetArch)
		if !ok {
			return false, false
		}
		left = left && right
	}
	return left, true
}

func parseGoBuildUnary(toks []string, pos *int, targetOS string, targetArch string) (bool, bool) {
	if *pos >= len(toks) {
		return false, false
	}
	tok := toks[*pos]
	if tok == "!" {
		*pos = *pos + 1
		value, ok := parseGoBuildUnary(toks, pos, targetOS, targetArch)
		return !value, ok
	}
	if tok == "(" {
		*pos = *pos + 1
		value, ok := parseGoBuildOr(toks, pos, targetOS, targetArch)
		if !ok || *pos >= len(toks) || toks[*pos] != ")" {
			return false, false
		}
		*pos = *pos + 1
		return value, true
	}
	if tok == ")" || tok == "&&" || tok == "||" {
		return false, false
	}
	*pos = *pos + 1
	return goBuildTagMatches(tok, targetOS, targetArch), true
}

func goBuildTagMatches(tag string, targetOS string, targetArch string) bool {
	if tag == targetOS || tag == targetArch {
		return true
	}
	if targetArch == "arm64" && tag == "aarch64" {
		return true
	}
	if targetArch == "wasm" && tag == "wasm32" {
		return true
	}
	return false
}

func targetFileParts(target string) (string, string) {
	slash := strings.IndexByte(target, '/')
	if slash < 0 {
		return "", ""
	}
	osPart := target[:slash]
	archPart := target[slash+1:]
	if archPart == "aarch64" {
		archPart = "arm64"
	}
	if archPart == "wasm32" {
		archPart = "wasm"
	}
	return osPart, archPart
}

func fileNameMatchesTarget(name string, targetOS string, targetArch string) bool {
	base := strings.TrimSuffix(name, ".go")
	parts := strings.Split(base, "_")
	if len(parts) < 2 {
		return true
	}
	last := parts[len(parts)-1]
	if isGoArchName(last) {
		if targetArch != "" && last != targetArch {
			return false
		}
		parts = parts[:len(parts)-1]
		if len(parts) < 2 {
			return true
		}
		last = parts[len(parts)-1]
	}
	if isGoOSName(last) {
		if targetOS != "" && last != targetOS {
			return false
		}
	}
	return true
}

func isGoOSName(name string) bool {
	switch name {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js", "linux", "netbsd", "openbsd", "plan9", "solaris", "wasi", "wasip1", "windows":
		return true
	}
	return false
}

func isGoArchName(name string) bool {
	switch name {
	case "386", "amd64", "amd64p32", "arm", "arm64", "loong64", "mips", "mips64", "mips64le", "mipsle", "ppc64", "ppc64le", "riscv64", "s390x", "sparc64", "wasm", "wasm32":
		return true
	}
	return false
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
	if next, ok, err := resolveReplacedImport(module, imp); ok || err != nil {
		return next, ok, err
	}
	if req, ok := requiredModuleForImport(module, imp); ok {
		return resolvedImport{}, false, fmt.Errorf("import %q uses required module %q; external module fetching is not supported", imp, req.Path)
	}
	stdDir := filepath.Join(opts.StdRoot, filepath.FromSlash(imp))
	if info, err := os.Stat(stdDir); err == nil && info.IsDir() {
		return resolvedImport{Dir: stdDir, ImportPath: imp}, true, nil
	}
	return resolvedImport{}, false, fmt.Errorf("import %q is not in module %q and was not found in rtg/std", imp, module.Path)
}

func resolveReplacedImport(module mod.Module, imp string) (resolvedImport, bool, error) {
	for _, repl := range module.Replaces {
		if imp != repl.Old && !strings.HasPrefix(imp, repl.Old+"/") {
			continue
		}
		if !isLocalPath(repl.New) {
			return resolvedImport{}, false, fmt.Errorf("import %q uses non-local replace target %q; external module fetching is not supported", imp, repl.New)
		}
		root := repl.New
		if !filepath.IsAbs(root) {
			root = filepath.Join(module.Root, root)
		}
		suffix := strings.TrimPrefix(imp, repl.Old)
		suffix = strings.TrimPrefix(suffix, "/")
		return resolvedImport{Dir: filepath.Join(root, filepath.FromSlash(suffix)), ImportPath: imp}, true, nil
	}
	return resolvedImport{}, false, nil
}

func requiredModuleForImport(module mod.Module, imp string) (mod.Require, bool) {
	for _, req := range module.Requires {
		if imp == req.Path || strings.HasPrefix(imp, req.Path+"/") {
			return req, true
		}
	}
	return mod.Require{}, false
}

func isLocalPath(path string) bool {
	return filepath.IsAbs(path) || strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") || path == "." || path == ".."
}

func importPathForDir(module mod.Module, dir string) string {
	rel, err := filepath.Rel(module.Root, dir)
	if err != nil || rel == "." {
		return module.Path
	}
	return module.Path + "/" + filepath.ToSlash(rel)
}
