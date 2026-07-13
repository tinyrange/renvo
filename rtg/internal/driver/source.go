package driver

import (
	"j5.nz/rtg/rtg/internal/load"
)

const (
	SourceOK = iota
	SourceErrMissingModule
	SourceErrModule
	SourceErrPackageArg
	SourceErrReadDir
	SourceErrReadFile
	SourceErrParse
	SourceErrImport
)

type DirEntry struct {
	Name  string
	IsDir bool
}

type SourceFS interface {
	ReadDir(path string) ([]DirEntry, bool)
	ReadFile(path string) ([]byte, bool)
}

type SourceResult struct {
	Files     []load.SourceFile
	Module    load.Module
	Root      load.PackageRef
	Ok        bool
	Error     int
	ErrorPath string
}

type sourceCollector struct {
	fs      SourceFS
	module  load.Module
	stdRoot string
	target  string
	files   []load.SourceFile
	loaded  []string
	loading []string
	ok      bool
	err     int
	errPath string
}

func CollectSources(workDir string, stdRoot string, arg string, fs SourceFS) SourceResult {
	return CollectSourcesForTarget(workDir, stdRoot, arg, DefaultTarget, fs)
}

func CollectSourcesForTarget(workDir string, stdRoot string, arg string, target string, fs SourceFS) SourceResult {
	result := SourceResult{Ok: true, Error: SourceOK}
	workDir = load.CleanPath(workDir)
	stdRoot = load.CleanPath(stdRoot)
	moduleRoot, moduleSrc, modulePath, ok := findModuleSource(workDir, fs)
	if !ok {
		return sourceFail(result, SourceErrMissingModule, load.JoinPath(workDir, "go.mod"))
	}
	result.Files = append(result.Files, load.SourceFile{Path: modulePath, Src: moduleSrc})
	module := load.ParseModule(moduleRoot, moduleSrc)
	result.Module = module
	if !module.Ok {
		return sourceFail(result, SourceErrModule, modulePath)
	}
	root := load.ResolvePackageArg(module, workDir, arg)
	result.Root = root
	if !root.Ok {
		return sourceFail(result, SourceErrPackageArg, arg)
	}
	collector := sourceCollector{
		fs:      fs,
		module:  module,
		stdRoot: stdRoot,
		target:  target,
		files:   result.Files,
		ok:      true,
		err:     SourceOK,
	}
	collector.collectPackage(root)
	result.Files = collector.files
	if !collector.ok {
		return sourceFail(result, collector.err, collector.errPath)
	}
	return result
}

func findModuleSource(workDir string, fs SourceFS) (string, []byte, string, bool) {
	dir := load.CleanPath(workDir)
	for {
		path := load.JoinPath(dir, "go.mod")
		src, ok := fs.ReadFile(path)
		if ok {
			return dir, src, path, true
		}
		next := load.DirPath(dir)
		if next == dir || dir == "." || dir == "/" {
			break
		}
		dir = next
	}
	return "", nil, "", false
}

func (c *sourceCollector) collectPackage(ref load.PackageRef) {
	if !c.ok {
		return
	}
	if ref.Kind != load.PackageInModule && ref.Kind != load.PackageStandard {
		c.fail(SourceErrImport, ref.ImportPath)
		return
	}
	if findString(c.loaded, ref.ImportPath) >= 0 || findString(c.loading, ref.ImportPath) >= 0 {
		return
	}
	c.loading = append(c.loading, ref.ImportPath)
	entries, ok := c.fs.ReadDir(ref.Dir)
	if !ok {
		c.fail(SourceErrReadDir, ref.Dir)
		return
	}
	sortDirEntries(entries)
	found := false
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if entry.IsDir || !isGoSourceName(entry.Name) {
			continue
		}
		path := load.JoinPath(ref.Dir, entry.Name)
		src, ok := c.fs.ReadFile(path)
		if !ok {
			c.fail(SourceErrReadFile, path)
			return
		}
		if !sourceFileEnabled(src, c.target) {
			continue
		}
		found = true
		c.files = append(c.files, load.SourceFile{Path: path, Src: src})
		imports, importsOK := collectSourceImports(c.module, c.stdRoot, src)
		if !importsOK {
			c.fail(SourceErrParse, path)
			return
		}
		for j := 0; j < len(imports); j++ {
			if !imports[j].Ok {
				c.fail(SourceErrImport, imports[j].ImportPath)
				return
			}
			c.collectPackage(imports[j])
			if !c.ok {
				return
			}
		}
	}
	if !found {
		c.fail(SourceErrReadDir, ref.Dir)
		return
	}
	c.loading = c.loading[:len(c.loading)-1]
	c.loaded = append(c.loaded, ref.ImportPath)
}

func (c *sourceCollector) fail(err int, path string) {
	c.ok = false
	c.err = err
	c.errPath = path
}

func sortDirEntries(entries []DirEntry) {
	for i := 1; i < len(entries); i++ {
		item := entries[i]
		j := i - 1
		for j >= 0 && driverStringAfter(entries[j].Name, item.Name) {
			entries[j+1] = entries[j]
			j--
		}
		entries[j+1] = item
	}
}

func driverStringAfter(left string, right string) bool {
	return driverStringBefore(right, left)
}

func driverStringBefore(left string, right string) bool {
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

func isGoSourceName(name string) bool {
	return stringHasSuffix(name, ".go") && !stringHasSuffix(name, "_test.go")
}

func filterSourcesForTarget(files []load.SourceFile, target string) []load.SourceFile {
	out := make([]load.SourceFile, 0, len(files))
	for i := 0; i < len(files); i++ {
		file := files[i]
		if isGoSourceName(load.BasePath(file.Path)) && !sourceFileEnabled(file.Src, target) {
			continue
		}
		out = append(out, file)
	}
	return out
}

func sourceFileEnabled(src []byte, target string) bool {
	pos := 0
	for pos < len(src) {
		lineStart := pos
		for pos < len(src) && src[pos] != '\n' {
			pos++
		}
		lineEnd := pos
		if pos < len(src) && src[pos] == '\n' {
			pos++
		}
		line := trimBuildLine(src[lineStart:lineEnd])
		if len(line) == 0 {
			continue
		}
		if bytesHasPrefix(line, []byte("//go:build")) && (len(line) == len("//go:build") || isBuildSpace(line[len("//go:build")])) {
			expr := trimBuildLine(line[len("//go:build"):])
			return evalBuildExpr(expr, target)
		}
		if bytesHasPrefix(line, []byte("//")) {
			continue
		}
		break
	}
	return true
}

type buildExprParser struct {
	src    []byte
	pos    int
	target string
	ok     bool
}

func evalBuildExpr(src []byte, target string) bool {
	parser := buildExprParser{src: src, target: target, ok: true}
	value := parser.parseOr()
	parser.skipSpace()
	if parser.pos != len(parser.src) {
		parser.ok = false
	}
	return parser.ok && value
}

func (p *buildExprParser) parseOr() bool {
	value := p.parseAnd()
	for {
		p.skipSpace()
		if !p.consume([]byte("||")) {
			return value
		}
		right := p.parseAnd()
		value = value || right
	}
}

func (p *buildExprParser) parseAnd() bool {
	value := p.parseUnary()
	for {
		p.skipSpace()
		if !p.consume([]byte("&&")) {
			return value
		}
		right := p.parseUnary()
		value = value && right
	}
}

func (p *buildExprParser) parseUnary() bool {
	p.skipSpace()
	if p.consume([]byte("!")) {
		return !p.parseUnary()
	}
	if p.consume([]byte("(")) {
		value := p.parseOr()
		p.skipSpace()
		if !p.consume([]byte(")")) {
			p.ok = false
		}
		return value
	}
	return p.parseTag()
}

func (p *buildExprParser) parseTag() bool {
	p.skipSpace()
	start := p.pos
	for p.pos < len(p.src) && isBuildTagChar(p.src[p.pos]) {
		p.pos++
	}
	if start == p.pos {
		p.ok = false
		return false
	}
	return hasBuildTag(p.target, string(p.src[start:p.pos]))
}

func (p *buildExprParser) skipSpace() {
	for p.pos < len(p.src) && isBuildSpace(p.src[p.pos]) {
		p.pos++
	}
}

func (p *buildExprParser) consume(text []byte) bool {
	if p.pos+len(text) > len(p.src) {
		return false
	}
	for i := 0; i < len(text); i++ {
		if p.src[p.pos+i] != text[i] {
			return false
		}
	}
	p.pos += len(text)
	return true
}

func hasBuildTag(target string, tag string) bool {
	if tag == "rtg" {
		return true
	}
	if tag == "linux" {
		return stringHasPrefix(target, "linux/")
	}
	if tag == "darwin" {
		return stringHasPrefix(target, "darwin/")
	}
	if tag == "unix" {
		return stringHasPrefix(target, "linux/") || stringHasPrefix(target, "darwin/")
	}
	if tag == "windows" {
		return stringHasPrefix(target, "windows/")
	}
	if tag == "wasi" || tag == "wasip1" {
		return stringHasPrefix(target, "wasi/")
	}
	if tag == "amd64" {
		return stringHasSuffix(target, "/amd64")
	}
	if tag == "386" {
		return stringHasSuffix(target, "/386")
	}
	if tag == "arm" {
		return stringHasSuffix(target, "/arm")
	}
	if tag == "aarch64" || tag == "arm64" {
		return stringHasSuffix(target, "/aarch64") || stringHasSuffix(target, "/arm64")
	}
	if tag == "wasm32" || tag == "wasm" {
		return stringHasSuffix(target, "/wasm32")
	}
	return false
}

func trimBuildLine(line []byte) []byte {
	start := 0
	end := len(line)
	for start < end && isBuildSpace(line[start]) {
		start++
	}
	for end > start && (isBuildSpace(line[end-1]) || line[end-1] == '\r') {
		end--
	}
	return line[start:end]
}

func bytesHasPrefix(data []byte, prefix []byte) bool {
	if len(prefix) > len(data) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if data[i] != prefix[i] {
			return false
		}
	}
	return true
}

func isBuildSpace(c byte) bool {
	return c == ' ' || c == '\t'
}

func isBuildTagChar(c byte) bool {
	if c >= 'a' && c <= 'z' {
		return true
	}
	if c >= 'A' && c <= 'Z' {
		return true
	}
	if c >= '0' && c <= '9' {
		return true
	}
	if c == '_' {
		return true
	}
	return c == '.'
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

func stringHasPrefix(text string, prefix string) bool {
	if len(prefix) > len(text) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if text[i] != prefix[i] {
			return false
		}
	}
	return true
}

func findString(items []string, item string) int {
	for i := 0; i < len(items); i++ {
		if items[i] == item {
			return i
		}
	}
	return -1
}

func sourceFail(result SourceResult, err int, path string) SourceResult {
	result.Ok = false
	result.Error = err
	result.ErrorPath = path
	return result
}
