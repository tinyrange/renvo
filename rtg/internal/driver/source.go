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
	SourceErrBuildConstraint
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
	Files           []load.SourceFile
	Module          load.Module
	Root            load.PackageRef
	Ok              bool
	Error           int
	ErrorPath       string
	ErrorSourcePath string
	ErrorOffset     int
}

type sourceCollector struct {
	fs            SourceFS
	module        load.Module
	stdRoot       string
	target        string
	tags          []string
	files         []load.SourceFile
	loaded        []string
	loading       []string
	ok            bool
	err           int
	errPath       string
	errSourcePath string
	errOffset     int
}

func CollectSources(workDir string, stdRoot string, arg string, fs SourceFS) SourceResult {
	return CollectSourcesForTarget(workDir, stdRoot, arg, DefaultTarget, fs)
}

func CollectSourcesForTarget(workDir string, stdRoot string, arg string, target string, fs SourceFS) SourceResult {
	return CollectSourcesForTargetTags(workDir, stdRoot, arg, target, nil, fs)
}

func CollectSourcesForTargetTags(workDir string, stdRoot string, arg string, target string, tags []string, fs SourceFS) SourceResult {
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
		tags:    tags,
		files:   result.Files,
		ok:      true,
		err:     SourceOK,
	}
	collector.collectPackage(root)
	result.Files = collector.files
	if !collector.ok {
		result = sourceFail(result, collector.err, collector.errPath)
		result.ErrorSourcePath = collector.errSourcePath
		result.ErrorOffset = collector.errOffset
		return result
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
		if !sourceFilenameEnabled(entry.Name, c.target) {
			continue
		}
		path := load.JoinPath(ref.Dir, entry.Name)
		src, ok := c.fs.ReadFile(path)
		if !ok {
			c.fail(SourceErrReadFile, path)
			return
		}
		enabled, valid := sourceConstraintsEnabled(src, c.target, c.tags)
		if !valid {
			c.fail(SourceErrBuildConstraint, path)
			return
		}
		if !enabled {
			continue
		}
		found = true
		c.files = append(c.files, load.SourceFile{Path: path, Src: src})
		imports, importsOK := collectSourceImports(c.module, c.stdRoot, src)
		if !importsOK {
			c.failAt(SourceErrParse, path, path, len(src))
			return
		}
		for j := 0; j < len(imports); j++ {
			if !imports[j].Ok {
				c.failAt(SourceErrImport, imports[j].ImportPath, path, sourceTextOffset(src, imports[j].ImportPath))
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

func (c *sourceCollector) failAt(err int, path string, sourcePath string, offset int) {
	c.fail(err, path)
	c.errSourcePath = sourcePath
	c.errOffset = offset
}

func sourceTextOffset(src []byte, text string) int {
	if len(text) == 0 || len(text) > len(src) {
		return 0
	}
	for i := 0; i+len(text) <= len(src); i++ {
		matched := true
		for j := 0; j < len(text); j++ {
			if src[i+j] != text[j] {
				matched = false
				break
			}
		}
		if matched {
			return i
		}
	}
	return 0
}

func sortDirEntries(entries []DirEntry) {
	for i := 1; i < len(entries); i++ {
		item := entries[i]
		j := i - 1
		for j >= 0 && entries[j].Name > item.Name {
			entries[j+1] = entries[j]
			j--
		}
		entries[j+1] = item
	}
}

func isGoSourceName(name string) bool {
	return stringHasSuffix(name, ".go") && name[0] != '.' && name[0] != '_' && !stringHasSuffix(name, "_test.go")
}

func filterSourcesForTargetTags(files []load.SourceFile, target string, tags []string) ([]load.SourceFile, string, bool) {
	out := make([]load.SourceFile, 0, len(files))
	for i := 0; i < len(files); i++ {
		file := files[i]
		name := load.BasePath(file.Path)
		if isGoSourceName(name) {
			if !sourceFilenameEnabled(name, target) {
				continue
			}
			enabled, valid := sourceConstraintsEnabled(file.Src, target, tags)
			if !valid {
				return nil, file.Path, false
			}
			if !enabled {
				continue
			}
		}
		out = append(out, file)
	}
	return out, "", true
}

func sourceFilenameEnabled(name string, target string) bool {
	stem := name[:len(name)-len(".go")]
	last := stringLastIndexByte(stem, '_')
	if last < 1 {
		return true
	}
	lastTag := stem[last+1:]
	if filenameKnownArch(lastTag) {
		if !hasBuildTag(target, lastTag, nil) {
			return false
		}
		before := stem[:last]
		previous := stringLastIndexByte(before, '_')
		if previous >= 1 && filenameKnownOS(before[previous+1:]) {
			return hasBuildTag(target, before[previous+1:], nil)
		}
		return true
	}
	if filenameKnownOS(lastTag) {
		return hasBuildTag(target, lastTag, nil)
	}
	return true
}

func filenameKnownOS(tag string) bool {
	return stringInBuildList(tag, "aix android darwin dragonfly freebsd hurd illumos ios js linux nacl netbsd openbsd plan9 solaris wasi wasip1 windows zos")
}

func filenameKnownArch(tag string) bool {
	return stringInBuildList(tag, "386 amd64 amd64p32 arm armbe arm64 aarch64 arm64be loong64 mips mipsle mips64 mips64le mips64p32 mips64p32le ppc ppc64 ppc64le riscv riscv64 s390 s390x sparc sparc64 wasm wasm32")
}

func stringInBuildList(item string, list string) bool {
	start := 0
	for i := 0; i <= len(list); i++ {
		if i < len(list) && list[i] != ' ' {
			continue
		}
		if list[start:i] == item {
			return true
		}
		start = i + 1
	}
	return false
}

func stringLastIndexByte(text string, value byte) int {
	for i := len(text) - 1; i >= 0; i-- {
		if text[i] == value {
			return i
		}
	}
	return -1
}

func sourceConstraintsEnabled(src []byte, target string, tags []string) (bool, bool) {
	pos := 0
	enabled := true
	modern := false
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
			if modern {
				return false, false
			}
			modern = true
			expr := trimBuildLine(line[len("//go:build"):])
			var valid bool
			enabled, valid = evalBuildExprWithTags(expr, target, tags)
			if !valid {
				return false, false
			}
			continue
		}
		if bytesHasPrefix(line, []byte("// +build")) && (len(line) == len("// +build") || isBuildSpace(line[len("// +build")])) {
			if modern {
				continue
			}
			lineEnabled, valid := evalPlusBuildLine(line[len("// +build"):], target, tags)
			if !valid {
				return false, false
			}
			enabled = enabled && lineEnabled
			continue
		}
		if bytesHasPrefix(line, []byte("//")) {
			continue
		}
		break
	}
	return enabled, true
}

type buildExprParser struct {
	src    []byte
	pos    int
	target string
	tags   []string
	ok     bool
}

func evalBuildExprWithTags(src []byte, target string, tags []string) (bool, bool) {
	parser := buildExprParser{src: src, target: target, tags: tags, ok: true}
	value := parser.parseOr()
	parser.skipSpace()
	if parser.pos != len(parser.src) {
		parser.ok = false
	}
	return value, parser.ok
}

func evalPlusBuildLine(src []byte, target string, tags []string) (bool, bool) {
	src = trimBuildLine(src)
	pos := 0
	enabled := false
	for pos < len(src) {
		for pos < len(src) && isBuildSpace(src[pos]) {
			pos++
		}
		option := true
		for {
			negated := false
			if src[pos] == '!' {
				negated = true
				pos++
			}
			start := pos
			for pos < len(src) && isBuildTagChar(src[pos]) {
				pos++
			}
			if start == pos {
				return false, false
			}
			if hasBuildTag(target, string(src[start:pos]), tags) == negated {
				option = false
			}
			if pos >= len(src) || isBuildSpace(src[pos]) {
				break
			}
			if src[pos] != ',' {
				return false, false
			}
			pos++
		}
		enabled = enabled || option
	}
	return enabled, len(src) > 0
}

func (p *buildExprParser) parseOr() bool {
	value := p.parseAnd()
	for {
		p.skipSpace()
		if p.pos+1 >= len(p.src) || p.src[p.pos] != '|' || p.src[p.pos+1] != '|' {
			return value
		}
		p.pos += 2
		right := p.parseAnd()
		value = value || right
	}
}

func (p *buildExprParser) parseAnd() bool {
	value := p.parseUnary()
	for {
		p.skipSpace()
		if p.pos+1 >= len(p.src) || p.src[p.pos] != '&' || p.src[p.pos+1] != '&' {
			return value
		}
		p.pos += 2
		right := p.parseUnary()
		value = value && right
	}
}

func (p *buildExprParser) parseUnary() bool {
	p.skipSpace()
	if p.pos < len(p.src) && p.src[p.pos] == '!' {
		p.pos++
		return !p.parseUnary()
	}
	if p.pos < len(p.src) && p.src[p.pos] == '(' {
		p.pos++
		value := p.parseOr()
		p.skipSpace()
		if p.pos >= len(p.src) || p.src[p.pos] != ')' {
			p.ok = false
		} else {
			p.pos++
		}
		return value
	}
	start := p.pos
	for p.pos < len(p.src) && isBuildTagChar(p.src[p.pos]) {
		p.pos++
	}
	if start == p.pos {
		p.ok = false
		return false
	}
	return hasBuildTag(p.target, string(p.src[start:p.pos]), p.tags)
}

func (p *buildExprParser) skipSpace() {
	for p.pos < len(p.src) && isBuildSpace(p.src[p.pos]) {
		p.pos++
	}
}

func hasBuildTag(target string, tag string, tags []string) bool {
	if findString(tags, tag) >= 0 {
		return true
	}
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
	return len(suffix) <= len(text) && text[len(text)-len(suffix):] == suffix
}

func stringHasPrefix(text string, prefix string) bool {
	return len(prefix) <= len(text) && text[:len(prefix)] == prefix
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
