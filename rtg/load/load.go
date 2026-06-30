package load

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"j5.nz/rtg/rtg/arena"
	"j5.nz/rtg/rtg/mod"
	"j5.nz/rtg/rtg/parse"
	targetpkg "j5.nz/rtg/rtg/target"
)

type File struct {
	Path     string
	UnitPath string
	Source   []byte
	Parsed   parse.File
}

type Package struct {
	ImportPath      string
	Dir             string
	Name            string
	Entry           bool
	Files           []File
	Imports         []string
	ImportPositions []ImportPosition
}

type ImportPosition struct {
	ImportPath string
	Path       string
	Line       int
	Column     int
}

type Graph struct {
	Module   mod.Module
	Packages []Package
}

type Options struct {
	StdRoot string
	Target  string
}

type fileEntryGroup struct {
	dir   string
	files []string
}

func LoadEntries(entries []string, opts Options) (*Graph, error) {
	if len(entries) == 0 {
		entries = []string{"."}
	}
	module, err := mod.Find(entries[0])
	if err != nil {
		return nil, err
	}
	if module.Root == "" {
		root, err := filepath.Abs(".")
		if err != nil {
			return nil, err
		}
		module = moduleWithRoot(module, root)
	}
	if opts.StdRoot == "" {
		opts.StdRoot = defaultStdRoot(module.Root)
	}
	if opts.Target == "" {
		opts.Target = targetpkg.Default()
	}
	var graph Graph
	graph.Module = module
	g := &graph
	var seen []string
	var fileEntries []fileEntryGroup
	var fileDirs []string
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		dir, files, err := entryInput(entry)
		if err != nil {
			return nil, err
		}
		if !isWithinModuleRoot(module.Root, dir) {
			return nil, fmt.Errorf("%s: entry is outside module root %s", dir, module.Root)
		}
		if canCheckNestedModuleRoots(module.Root) {
			nested := nestedModuleRoot(module.Root, dir)
			if nested != "" {
				return nil, fmt.Errorf("%s: entry is inside nested module root %s", dir, nested)
			}
		}
		if len(files) > 0 {
			index := fileEntryGroupIndex(fileEntries, dir)
			if index < 0 {
				fileDirs = append(fileDirs, dir)
				fileEntries = append(fileEntries, fileEntryGroup{dir: dir})
				index = len(fileEntries) - 1
			}
			fileEntries[index].files = appendStrings(fileEntries[index].files, files)
			continue
		}
		if err := loadPackageRecursive(g, opts, &seen, dir); err != nil {
			return nil, err
		}
	}
	sortStrings(fileDirs)
	for i := 0; i < len(fileDirs); i++ {
		dir := fileDirs[i]
		index := fileEntryGroupIndex(fileEntries, dir)
		if index < 0 {
			continue
		}
		files := fileEntries[index].files
		if err := loadPackageFilesRecursive(g, opts, &seen, dir, files); err != nil {
			return nil, err
		}
	}
	return g, nil
}

func moduleWithRoot(module mod.Module, root string) mod.Module {
	return mod.Module{Root: root, Path: module.Path, Requires: module.Requires, Replaces: module.Replaces}
}

func defaultStdRoot(moduleRoot string) string {
	if env := os.Getenv("RTG_STD"); env != "" {
		return env
	}
	moduleStd := filepath.Join(filepath.Join(moduleRoot, "rtg"), "std")
	if info, err := os.Stat(moduleStd); err == nil && fileInfoIsDir(info) {
		return moduleStd
	}
	cwd, err := os.Getwd()
	if err == nil {
		root, ok := findStdRootUpward(cwd)
		if ok {
			return root
		}
	}
	return moduleStd
}

func findStdRootUpward(start string) (string, bool) {
	dir := filepath.Clean(start)
	for {
		candidate := filepath.Join(filepath.Join(dir, "rtg"), "std")
		if info, err := os.Stat(candidate); err == nil && fileInfoIsDir(info) {
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
	if fileInfoIsDir(info) {
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

func loadPackageRecursive(g *Graph, opts Options, seen *[]string, dir string) error {
	importPath := importPathForDir(g.Module, dir)
	return loadPackageRecursiveAs(g, opts, seen, dir, importPath, true)
}

func loadPackageFilesRecursive(g *Graph, opts Options, seen *[]string, dir string, files []string) error {
	importPath := importPathForDir(g.Module, dir)
	return loadPackageFilesRecursiveAs(g, opts, seen, dir, importPath, files, true)
}

func loadPackageRecursiveAs(g *Graph, opts Options, seen *[]string, dir string, importPath string, entry bool) error {
	dir = filepath.Clean(dir)
	seenValues := *seen
	if containsString(seenValues, dir) {
		if entry {
			markPackageEntry(g, importPath)
		}
		return nil
	}
	seenValues = append(seenValues, dir)
	*seen = seenValues
	pkg, err := readPackage(g.Module, dir, importPath, opts)
	if err != nil {
		return err
	}
	pkg.Entry = entry
	g.Packages = append(g.Packages, pkg)
	pkgImports := pkg.Imports
	for i := 0; i < len(pkgImports); i++ {
		imp := pkgImports[i]
		next, ok, err := resolveImport(g.Module, opts, imp)
		if err != nil {
			return importResolutionError(pkg, imp, err)
		}
		if ok {
			if err := loadPackageRecursiveAs(g, opts, seen, next.Dir, next.ImportPath, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func loadPackageFilesRecursiveAs(g *Graph, opts Options, seen *[]string, dir string, importPath string, files []string, entry bool) error {
	dir = filepath.Clean(dir)
	seenValues := *seen
	if containsString(seenValues, dir) {
		if entry {
			markPackageEntry(g, importPath)
		}
		return nil
	}
	seenValues = append(seenValues, dir)
	*seen = seenValues
	pkg, err := readPackageFiles(g.Module, dir, importPath, files)
	if err != nil {
		return err
	}
	pkg.Entry = entry
	g.Packages = append(g.Packages, pkg)
	pkgImports := pkg.Imports
	for i := 0; i < len(pkgImports); i++ {
		imp := pkgImports[i]
		next, ok, err := resolveImport(g.Module, opts, imp)
		if err != nil {
			return importResolutionError(pkg, imp, err)
		}
		if ok {
			if err := loadPackageRecursiveAs(g, opts, seen, next.Dir, next.ImportPath, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func markPackageEntry(g *Graph, importPath string) {
	for i := 0; i < len(g.Packages); i++ {
		if g.Packages[i].ImportPath == importPath {
			g.Packages[i].Entry = true
			return
		}
	}
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
	files = copyStrings(files)
	sortStrings(files)
	files = uniqueStrings(files)
	pkg := Package{Dir: dir, ImportPath: importPath}
	var importSet []string
	for i := 0; i < len(files); i++ {
		path := files[i]
		data, err := os.ReadFile(path)
		if err != nil {
			return Package{}, err
		}
		source := data
		info, err := ParseSourceInfo(path, source)
		if err != nil {
			return Package{}, err
		}
		if pkg.Name == "" {
			pkg.Name = info.PackageName
		} else if pkg.Name != info.PackageName {
			return Package{}, fmt.Errorf("%s: mixed package names %s and %s", dir, pkg.Name, info.PackageName)
		}
		for j := 0; j < len(info.Imports); j++ {
			imp := info.Imports[j]
			appendPackageImport(path, &pkg, &importSet, imp.Path, imp.Alias, imp.Line, imp.Column)
		}
		var file File
		file.Path = path
		file.UnitPath = unitFilePath(module, importPath, path)
		file.Source = source
		pkg.Files = append(pkg.Files, file)
	}
	sortStrings(pkg.Imports)
	return pkg, nil
}

func importResolutionError(pkg Package, imp string, err error) error {
	pos, ok := importPosition(pkg.ImportPositions, imp)
	if !ok || pos.Path == "" || pos.Line == 0 || pos.Column == 0 {
		return err
	}
	return fmt.Errorf("%s:%d:%d: %v", pos.Path, pos.Line, pos.Column, err)
}

func fileEntryGroupIndex(groups []fileEntryGroup, dir string) int {
	for i := 0; i < len(groups); i++ {
		if groups[i].dir == dir {
			return i
		}
	}
	return -1
}

func containsString(values []string, value string) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return true
		}
	}
	return false
}

func hasImportPosition(values []ImportPosition, path string) bool {
	for i := 0; i < len(values); i++ {
		if values[i].ImportPath == path {
			return true
		}
	}
	return false
}

func importPosition(values []ImportPosition, path string) (ImportPosition, bool) {
	for i := 0; i < len(values); i++ {
		if values[i].ImportPath == path {
			return values[i], true
		}
	}
	return ImportPosition{}, false
}

func uniqueStrings(values []string) []string {
	var out []string
	for i := 0; i < len(values); i++ {
		value := values[i]
		if len(out) > 0 && out[len(out)-1] == value {
			continue
		}
		out = append(out, value)
	}
	return out
}

func appendStrings(out []string, values []string) []string {
	for i := 0; i < len(values); i++ {
		out = append(out, values[i])
	}
	return out
}

func copyStrings(values []string) []string {
	out := make([]string, len(values))
	for i := 0; i < len(values); i++ {
		out[i] = values[i]
	}
	return out
}

func copyLoadString(value string) string {
	var out []byte
	for i := 0; i < len(value); i++ {
		out = append(out, value[i])
	}
	return string(out)
}

func copyBytes(values []byte) []byte {
	out := make([]byte, len(values))
	for i := 0; i < len(values); i++ {
		out[i] = values[i]
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
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if dirEntryIsDir(entry) {
			continue
		}
		name := dirEntryName(entry)
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
			continue
		}
		if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, ".rtg.go") {
			continue
		}
		if !fileNameMatchesTarget(name, targetOS, targetArch) {
			continue
		}
		path := filepath.Join(dir, name)
		mark := arena.Mark()
		ok, err := fileBuildTagsMatchTarget(path, targetOS, targetArch)
		if err != nil {
			return nil, err
		}
		arena.Reset(mark)
		if !ok {
			continue
		}
		files = append(files, path)
	}
	sortStrings(files)
	return files, nil
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		value := values[i]
		j := i - 1
		for j >= 0 && stringGreater(values[j], value) {
			values[j+1] = values[j]
			j = j - 1
		}
		values[j+1] = value
	}
}

func stringGreater(a string, b string) bool {
	i := 0
	for i < len(a) && i < len(b) {
		if a[i] > b[i] {
			return true
		}
		if a[i] < b[i] {
			return false
		}
		i = i + 1
	}
	return len(a) > len(b)
}

func fileBuildTagsMatchTarget(path string, targetOS string, targetArch string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	src := bytesToString(data)
	expr, ok := leadingGoBuildExpr(src)
	if ok {
		return evalGoBuildExpr(expr, targetOS, targetArch), nil
	}
	lines := leadingPlusBuildLines(src)
	if len(lines) == 0 {
		return true, nil
	}
	return evalPlusBuildLines(lines, targetOS, targetArch), nil
}

func bytesToString(data []byte) string {
	return string(data)
}

func leadingGoBuildExpr(src string) (string, bool) {
	inBlockComment := false
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
		if inBlockComment {
			if strings.Contains(line, "*/") {
				inBlockComment = false
			}
			continue
		}
		if strings.HasPrefix(line, "//go:build ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "//go:build ")), true
		}
		if strings.HasPrefix(line, "/*") {
			if !strings.Contains(line, "*/") {
				inBlockComment = true
			}
			continue
		}
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		return "", false
	}
	return "", false
}

func leadingPlusBuildLines(src string) []string {
	var lines []string
	inBlockComment := false
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
		if inBlockComment {
			if strings.Contains(line, "*/") {
				inBlockComment = false
			}
			continue
		}
		if strings.HasPrefix(line, "// +build ") {
			lines = append(lines, strings.TrimSpace(strings.TrimPrefix(line, "// +build ")))
			continue
		}
		if strings.HasPrefix(line, "/*") {
			if !strings.Contains(line, "*/") {
				inBlockComment = true
			}
			continue
		}
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		break
	}
	return lines
}

func evalPlusBuildLines(lines []string, targetOS string, targetArch string) bool {
	for i := 0; i < len(lines); i++ {
		line := lines[i]
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
	for i := 0; i < len(options); i++ {
		option := options[i]
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
	for i := 0; i < len(terms); i++ {
		term := terms[i]
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
		if c == '(' {
			toks = append(toks, "(")
			i++
			continue
		}
		if c == ')' {
			toks = append(toks, ")")
			i++
			continue
		}
		if c == '!' {
			toks = append(toks, "!")
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
	if tag == "rtg" {
		return true
	}
	if tag == "unix" && isUnixTargetOS(targetOS) {
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

func isUnixTargetOS(targetOS string) bool {
	switch targetOS {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "linux", "netbsd", "openbsd", "solaris":
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
		dir := filepath.Join(module.Root, filepath.FromSlash(strings.TrimPrefix(imp, prefix)))
		if canCheckNestedModuleRoots(module.Root) {
			nested := nestedModuleRoot(module.Root, dir)
			if nested != "" {
				return resolvedImport{}, false, fmt.Errorf("import %q crosses nested module root %s", imp, nested)
			}
		}
		return resolvedImport{Dir: dir, ImportPath: imp}, true, nil
	}
	next, ok, err := resolveReplacedImport(module, imp)
	if ok || err != nil {
		return next, ok, err
	}
	req, ok := requiredModuleForImport(module, imp)
	if ok {
		vendorNext, vendorOK := resolveVendorImport(module, req, imp)
		if vendorOK {
			return vendorNext, true, nil
		}
		return resolvedImport{}, false, fmt.Errorf("import %q uses required module %q; external module fetching is not supported", imp, req.Path)
	}
	stdDir := filepath.Join(opts.StdRoot, filepath.FromSlash(imp))
	if info, err := os.Stat(stdDir); err == nil && fileInfoIsDir(info) {
		return resolvedImport{Dir: stdDir, ImportPath: imp}, true, nil
	}
	if isStandardImportPath(imp) {
		return resolvedImport{}, false, fmt.Errorf("standard package %q is not available in rtg/std", imp)
	}
	return resolvedImport{}, false, fmt.Errorf("import %q is not in module %q and was not found in rtg/std", imp, module.Path)
}

func isStandardImportPath(path string) bool {
	first := path
	if slash := strings.IndexByte(path, '/'); slash >= 0 {
		first = path[:slash]
	}
	return first != "" && !strings.Contains(first, ".")
}

func resolveReplacedImport(module mod.Module, imp string) (resolvedImport, bool, error) {
	for i := 0; i < len(module.Replaces); i++ {
		repl := module.Replaces[i]
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

func resolveVendorImport(module mod.Module, req mod.Require, imp string) (resolvedImport, bool) {
	if imp != req.Path && !strings.HasPrefix(imp, req.Path+"/") {
		return resolvedImport{}, false
	}
	dir := filepath.Join(filepath.Join(module.Root, "vendor"), filepath.FromSlash(imp))
	if info, err := os.Stat(dir); err == nil && fileInfoIsDir(info) {
		return resolvedImport{Dir: dir, ImportPath: imp}, true
	}
	return resolvedImport{}, false
}

func requiredModuleForImport(module mod.Module, imp string) (mod.Require, bool) {
	for i := 0; i < len(module.Requires); i++ {
		req := module.Requires[i]
		if imp == req.Path || strings.HasPrefix(imp, req.Path+"/") {
			return req, true
		}
	}
	return mod.Require{}, false
}

func isLocalPath(path string) bool {
	return filepath.IsAbs(path) || strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") || path == "." || path == ".."
}

func isWithinModuleRoot(root string, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." {
		return false
	}
	parentPrefix := "../"
	if strings.HasPrefix(rel, parentPrefix) {
		return false
	}
	if filepath.IsAbs(rel) {
		return false
	}
	return true
}

func canCheckNestedModuleRoots(root string) bool {
	return filepath.IsAbs(root)
}

func nestedModuleRoot(root string, dir string) string {
	root = filepath.Clean(root)
	dir = filepath.Clean(dir)
	for {
		if dir == root {
			return ""
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func importPathForDir(module mod.Module, dir string) string {
	rel, err := filepath.Rel(module.Root, dir)
	if err != nil || rel == "." {
		return module.Path
	}
	return module.Path + "/" + filepath.ToSlash(rel)
}

func fileInfoIsDir(info os.FileInfo) bool {
	return info.IsDir()
}

func dirEntryIsDir(entry os.DirEntry) bool {
	return entry.IsDir()
}

func dirEntryName(entry os.DirEntry) string {
	return entry.Name()
}
