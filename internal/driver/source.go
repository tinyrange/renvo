package driver

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/load"
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
	SourceErrDependencyMissing
	SourceErrDependencyExcluded
	SourceErrDependencyModule
	SourceErrDependencyAmbiguous
	SourceErrEmbed
	SourceErrCgo
	SourceErrStandardPackage
	SourceErrFileDirectory
	SourceErrFileListEmpty
)

type DirEntry struct {
	Name  string
	IsDir bool
}

type SourceFS interface {
	ReadDir(path string) ([]DirEntry, bool)
	ReadFile(path string) ([]byte, bool)
	PathExists(path string) bool
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
	config        *load.ModuleConfig
	modules       []load.Module
	stdRoot       string
	moduleCache   string
	target        string
	tags          []string
	files         []load.SourceFile
	loaded        []string
	loading       []string
	resolved      []load.ModuleVersion
	ok            bool
	restart       bool
	err           int
	errPath       string
	errSourcePath string
	errOffset     int
	explicitRoot  string
	explicitFiles []string
}

func CollectSources(workDir string, stdRoot string, arg string, fs SourceFS) SourceResult {
	return CollectSourcesForTarget(workDir, stdRoot, arg, DefaultTarget, fs)
}

func CollectSourcesForTarget(workDir string, stdRoot string, arg string, target string, fs SourceFS) SourceResult {
	return CollectSourcesForTargetTags(workDir, stdRoot, arg, target, nil, fs)
}

func CollectSourcesForTargetTags(workDir string, stdRoot string, arg string, target string, tags []string, fs SourceFS) SourceResult {
	return CollectSourcesForTargetTagsWithModuleCache(workDir, stdRoot, arg, target, tags, "", fs)
}

// CollectSourcesForTargetTagsWithModuleCache resolves dependencies only from
// the main module's vendor tree, local replacements, or this read-only cache.
// It never performs network access.
func CollectSourcesForTargetTagsWithModuleCache(workDir string, stdRoot string, arg string, target string, tags []string, moduleCache string, fs SourceFS) SourceResult {
	return collectSourcesForTargetTagsWithModuleCache(workDir, stdRoot, arg, nil, target, tags, moduleCache, fs)
}

func CollectSourceFilesForTargetTagsWithModuleCache(workDir string, stdRoot string, files []string, target string, tags []string, moduleCache string, fs SourceFS) SourceResult {
	return collectSourcesForTargetTagsWithModuleCache(workDir, stdRoot, "", files, target, tags, moduleCache, fs)
}

func collectSourcesForTargetTagsWithModuleCache(workDir string, stdRoot string, arg string, explicitFiles []string, target string, tags []string, moduleCache string, fs SourceFS) SourceResult {
	result := SourceResult{Ok: true, Error: SourceOK}
	workDir = load.CleanPath(workDir)
	stdRoot = load.CleanPath(stdRoot)
	if moduleCache != "" {
		moduleCache = load.CleanPath(moduleCache)
	}
	moduleRoot, moduleSrc, modulePath, ok := findModuleSource(workDir, fs)
	if !ok {
		return sourceFail(result, SourceErrMissingModule, load.JoinPath(workDir, "go.mod"))
	}
	config := &load.ModuleConfig{}
	module := load.ParseModuleConfig(moduleRoot, moduleSrc, config)
	result.Module = module
	if !module.Ok {
		return sourceFail(result, SourceErrModule, modulePath)
	}
	result.Files = append(result.Files, load.SourceFile{Path: modulePath, Src: moduleSrc})
	var normalizedFiles []string
	if len(explicitFiles) > 0 {
		rootDir := ""
		for i := 0; i < len(explicitFiles); i++ {
			path := load.JoinPath(workDir, explicitFiles[i])
			if !isGoSourceName(load.BasePath(path)) {
				continue
			}
			dir := load.DirPath(path)
			if rootDir == "" {
				rootDir = dir
			} else if dir != rootDir {
				return sourceFail(result, SourceErrFileDirectory, path)
			}
			normalizedFiles = append(normalizedFiles, path)
		}
		if len(normalizedFiles) == 0 {
			return sourceFail(result, SourceErrFileListEmpty, explicitFiles[0])
		}
		arg = rootDir
	}
	root := load.ResolvePackageArg(module, workDir, arg)
	result.Root = root
	if !root.Ok {
		return sourceFail(result, SourceErrPackageArg, arg)
	}
	for attempt := 0; attempt < 64; attempt++ {
		collector := sourceCollector{
			fs:            fs,
			module:        module,
			config:        config,
			modules:       []load.Module{module},
			stdRoot:       stdRoot,
			moduleCache:   moduleCache,
			target:        target,
			tags:          tags,
			files:         result.Files,
			ok:            true,
			err:           SourceOK,
			explicitRoot:  root.ImportPath,
			explicitFiles: normalizedFiles,
		}
		collector.collectPackage(root)
		if collector.restart {
			continue
		}
		result.Files = collector.files
		if !collector.ok {
			result = sourceFail(result, collector.err, collector.errPath)
			result.ErrorSourcePath = collector.errSourcePath
			result.ErrorOffset = collector.errOffset
		}
		return result
	}
	return sourceFail(result, SourceErrDependencyAmbiguous, module.Path)
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
	if ref.Kind != load.PackageInModule && ref.Kind != load.PackageStandard && ref.Kind != load.PackageDependency {
		c.fail(SourceErrImport, ref.ImportPath)
		return
	}
	if findString(c.loaded, ref.ImportPath) >= 0 || findString(c.loading, ref.ImportPath) >= 0 {
		return
	}
	c.loading = append(c.loading, ref.ImportPath)
	owner := c.module
	if ref.Kind != load.PackageStandard {
		var ownerOK bool
		owner, ownerOK = c.ownerModule(ref.ImportPath)
		if !ownerOK {
			c.fail(SourceErrDependencyAmbiguous, ref.ImportPath)
			return
		}
		if c.crossesNestedModule(owner, ref.Dir) {
			c.fail(SourceErrDependencyAmbiguous, ref.ImportPath)
			return
		}
	}
	explicit := ref.ImportPath == c.explicitRoot && len(c.explicitFiles) > 0
	var paths []string
	if explicit {
		paths = c.explicitFiles
	} else {
		entries, ok := c.fs.ReadDir(ref.Dir)
		if !ok {
			c.fail(SourceErrReadDir, ref.Dir)
			return
		}
		sortDirEntries(entries)
		for i := 0; i < len(entries); i++ {
			if !entries[i].IsDir && isGoSourceName(entries[i].Name) {
				paths = append(paths, load.JoinPath(ref.Dir, entries[i].Name))
			}
		}
	}
	found := false
	for i := 0; i < len(paths); i++ {
		path := paths[i]
		if !explicit && !sourceFilenameEnabled(load.BasePath(path), c.target) {
			continue
		}
		arenaStart := arena.Mark()
		src, ok := c.fs.ReadFile(path)
		arenaEnd := arena.Mark()
		if !ok {
			c.fail(SourceErrReadFile, path)
			return
		}
		if !explicit {
			enabled, valid := sourceConstraintsEnabled(src, c.target, c.tags)
			if !valid {
				c.fail(SourceErrBuildConstraint, path)
				return
			}
			if !enabled {
				if sourceRequiresCgo(src, c.target, c.tags) {
					c.files = append(c.files, load.SourceFile{Path: path, Src: src, ArenaStart: arenaStart, ArenaEnd: arenaEnd})
					c.failAt(SourceErrCgo, "cgo", path, sourceTextOffset(src, "cgo"))
					return
				}
				arena.Discard(arenaStart, arenaEnd)
				continue
			}
		}
		expanded, embedOK, embedOffset, embedPath := expandSourceEmbeds(c.fs, path, owner.Root, src)
		if !embedOK {
			c.files = append(c.files, load.SourceFile{Path: path, Src: src, ArenaStart: arenaStart, ArenaEnd: arena.Mark()})
			c.failAt(SourceErrEmbed, embedPath, path, embedOffset)
			return
		}
		src = expanded
		arenaEnd = arena.Mark()
		found = true
		c.files = append(c.files, load.SourceFile{Path: path, Src: src, ArenaStart: arenaStart, ArenaEnd: arenaEnd})
		imports, importsOK := collectSourceImports(owner, c.stdRoot, src)
		if !importsOK {
			c.failAt(SourceErrParse, path, path, len(src))
			return
		}
		for j := 0; j < len(imports); j++ {
			importOffset := sourceTextOffset(src, imports[j].ImportPath)
			if imports[j].ImportPath == "C" {
				c.failAt(SourceErrCgo, "C", path, importOffset)
				return
			}
			if imports[j].Kind == load.PackageStandard {
				if _, present := c.fs.ReadDir(imports[j].Dir); !present {
					c.failAt(SourceErrStandardPackage, imports[j].ImportPath, path, importOffset)
					return
				}
			}
			if !imports[j].Ok {
				loaded := c.resolveLoadedImport(imports[j].ImportPath)
				if loaded.Ok {
					imports[j] = loaded
				}
			}
			_, required := longestModuleRequirement(c.config.Requires, imports[j].ImportPath)
			dependencyShadowsOwner := required && imports[j].Kind == load.PackageInModule
			if !imports[j].Ok || dependencyShadowsOwner {
				resolved := c.resolveDependency(imports[j].ImportPath)
				if !resolved.Ok {
					if c.ok {
						c.failAt(SourceErrImport, imports[j].ImportPath, path, importOffset)
					} else {
						c.errSourcePath = path
						c.errOffset = sourceTextOffset(src, imports[j].ImportPath)
					}
					return
				}
				imports[j] = resolved
			}
			c.collectPackage(imports[j])
			if !c.ok {
				if c.errSourcePath == "" {
					c.errSourcePath = path
					c.errOffset = sourceTextOffset(src, imports[j].ImportPath)
				}
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

func sourceGenericsOffset(src []byte) int {
	for pos := 0; pos < len(src); {
		pos = renvoImportSkipSpace(src, pos)
		if pos >= len(src) {
			break
		}
		if src[pos] == '"' || src[pos] == '\'' || src[pos] == '`' {
			pos = sourceEmbedSkipQuoted(src, pos, src[pos])
			continue
		}
		start, end, next, ok := renvoImportIdent(src, pos)
		if !ok {
			pos++
			continue
		}
		pos = next
		if !renvoImportTextIs(src, start, end, "func") && !renvoImportTextIs(src, start, end, "type") {
			continue
		}
		pos = renvoImportSkipSpace(src, pos)
		_, _, pos, ok = renvoImportIdent(src, pos)
		if !ok {
			continue
		}
		pos = renvoImportSkipSpace(src, pos)
		if pos < len(src) && src[pos] == '[' {
			return pos
		}
	}
	return -1
}

func sourceRequiresCgo(src []byte, target string, tags []string) bool {
	disabled, disabledOK := sourceConstraintsEnabled(src, target, tags)
	with := append(tags, "cgo")
	enabled, enabledOK := sourceConstraintsEnabled(src, target, with)
	return disabledOK && enabledOK && !disabled && enabled
}

func (c *sourceCollector) resolveLoadedImport(importPath string) load.PackageRef {
	module, ok := c.ownerModule(importPath)
	if !ok {
		return unsupportedPackage(importPath)
	}
	dir := module.Root
	if importPath != module.Path {
		dir = load.JoinPath(dir, importPath[len(module.Path)+1:])
	}
	kind := load.PackageDependency
	if module.Path == c.module.Path {
		kind = load.PackageInModule
	}
	return load.PackageRef{Kind: kind, ImportPath: importPath, Dir: dir, Ok: true, Error: load.ResolveOK}
}

func (c *sourceCollector) ownerModule(importPath string) (load.Module, bool) {
	best := -1
	for i := 0; i < len(c.modules); i++ {
		path := c.modules[i].Path
		if importPath == path || load.HasImportPrefix(importPath, path) {
			if best < 0 || len(path) > len(c.modules[best].Path) {
				best = i
			}
		}
	}
	if best < 0 {
		return load.Module{}, false
	}
	return c.modules[best], true
}

func (c *sourceCollector) resolveDependency(importPath string) load.PackageRef {
	requirement, ok := longestModuleRequirement(c.config.Requires, importPath)
	if !ok {
		return unsupportedPackage(importPath)
	}
	if moduleVersionExcluded(c.config, requirement) {
		c.fail(SourceErrDependencyExcluded, requirement.Path+"@"+requirement.Version)
		return unsupportedPackage(importPath)
	}
	root := ""
	moduleSourceRequired := true
	replacement, replaced := findModuleReplacement(c.config, requirement)
	localReplacement := replaced && replacement.Local
	cachePath := requirement.Path
	cacheVersion := requirement.Version
	if localReplacement {
		root = load.JoinPath(c.module.Root, replacement.NewPath)
	} else if replaced {
		cachePath, cacheVersion = replacement.NewPath, replacement.NewVersion
	}
	vendorRoot := load.JoinPath(load.JoinPath(c.module.Root, "vendor"), requirement.Path)
	vendorPackage := vendorRoot
	if importPath != requirement.Path {
		vendorPackage = load.JoinPath(vendorRoot, importPath[len(requirement.Path)+1:])
	}
	if root == "" {
		_, present := c.fs.ReadDir(vendorPackage)
		if present {
			root = vendorRoot
			moduleSourceRequired = false
		}
	}
	if root == "" && c.moduleCache != "" {
		root = load.JoinPath(c.moduleCache, escapeModuleCachePath(cachePath)+"@"+escapeModuleCachePath(cacheVersion))
	}
	if root == "" {
		c.fail(SourceErrDependencyMissing, requirement.Path+"@"+requirement.Version)
		return unsupportedPackage(importPath)
	}
	dependency, exists := c.moduleByPath(requirement.Path)
	if !exists {
		dependency = load.Module{Root: root, Path: requirement.Path, Ok: true, Error: load.ModuleOK, ErrorOffset: -1}
		goModPath := load.JoinPath(root, "go.mod")
		if moduleSourceRequired {
			goMod, readable := c.fs.ReadFile(goModPath)
			if !readable {
				if localReplacement {
					c.fail(SourceErrDependencyModule, goModPath)
				} else {
					c.fail(SourceErrDependencyMissing, requirement.Path+"@"+requirement.Version)
				}
				return unsupportedPackage(importPath)
			}
			dependencyConfig := &load.ModuleConfig{}
			dependency = load.ParseModuleConfig(root, goMod, dependencyConfig)
			if !dependency.Ok {
				c.fail(SourceErrDependencyModule, goModPath)
				return unsupportedPackage(importPath)
			}
			dependency.Path = requirement.Path
			if !c.selectRequirements(dependencyConfig.Requires) {
				return unsupportedPackage(importPath)
			}
		}
		manifest := []byte(requirement.Path)
		c.files = append(c.files, load.SourceFile{Path: goModPath, Src: manifest})
		c.modules = append(c.modules, dependency)
		c.resolved = append(c.resolved, requirement)
	} else if load.CleanPath(dependency.Root) != load.CleanPath(root) {
		c.fail(SourceErrDependencyAmbiguous, requirement.Path)
		return unsupportedPackage(importPath)
	}
	dir := dependency.Root
	if importPath != dependency.Path {
		dir = load.JoinPath(dir, importPath[len(dependency.Path)+1:])
	}
	return load.PackageRef{Kind: load.PackageDependency, ImportPath: importPath, Dir: dir, Ok: true, Error: load.ResolveOK}
}

func (c *sourceCollector) selectRequirements(requirements []load.ModuleVersion) bool {
	for i := 0; i < len(requirements); i++ {
		selected := -1
		for j := 0; j < len(c.config.Requires); j++ {
			if c.config.Requires[j].Path == requirements[i].Path {
				selected = j
				break
			}
		}
		if selected < 0 {
			c.config.Requires = append(c.config.Requires, requirements[i])
			continue
		}
		if compareModuleVersion(requirements[i].Version, c.config.Requires[selected].Version) <= 0 {
			continue
		}
		c.config.Requires[selected] = requirements[i]
		for j := 0; j < len(c.resolved); j++ {
			if c.resolved[j].Path == requirements[i].Path && c.resolved[j].Version != requirements[i].Version {
				c.restart = true
				c.ok = false
				return false
			}
		}
	}
	return true
}

func compareModuleVersion(left string, right string) int {
	leftEnd := moduleVersionCoreEnd(left)
	rightEnd := moduleVersionCoreEnd(right)
	leftAt := 0
	rightAt := 0
	if leftAt < leftEnd && left[leftAt] == 'v' {
		leftAt++
	}
	if rightAt < rightEnd && right[rightAt] == 'v' {
		rightAt++
	}
	for leftAt < leftEnd || rightAt < rightEnd {
		leftValue, leftNext := moduleVersionNumber(left, leftAt, leftEnd)
		rightValue, rightNext := moduleVersionNumber(right, rightAt, rightEnd)
		if leftValue < rightValue {
			return -1
		}
		if leftValue > rightValue {
			return 1
		}
		leftAt = leftNext
		rightAt = rightNext
	}
	leftPre := moduleVersionPrerelease(left)
	rightPre := moduleVersionPrerelease(right)
	if leftPre == rightPre {
		return 0
	}
	if leftPre == "" {
		return 1
	}
	if rightPre == "" {
		return -1
	}
	return compareModulePrerelease(leftPre, rightPre)
}

func moduleVersionCoreEnd(version string) int {
	for i := 0; i < len(version); i++ {
		if version[i] == '-' || version[i] == '+' {
			return i
		}
	}
	return len(version)
}

func moduleVersionNumber(version string, start int, end int) (int, int) {
	for start < end && version[start] == '.' {
		start++
	}
	value := 0
	for start < end && version[start] >= '0' && version[start] <= '9' {
		value = value*10 + int(version[start]-'0')
		start++
	}
	for start < end && version[start] != '.' {
		start++
	}
	return value, start
}

func moduleVersionPrerelease(version string) string {
	for i := 0; i < len(version); i++ {
		if version[i] == '-' {
			end := len(version)
			for j := i + 1; j < len(version); j++ {
				if version[j] == '+' {
					end = j
					break
				}
			}
			return version[i+1 : end]
		}
		if version[i] == '+' {
			break
		}
	}
	return ""
}

func compareModuleVersionText(left string, right string) int {
	count := len(left)
	if len(right) < count {
		count = len(right)
	}
	for i := 0; i < count; i++ {
		if left[i] < right[i] {
			return -1
		}
		if left[i] > right[i] {
			return 1
		}
	}
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	return 0
}

func compareModulePrerelease(left string, right string) int {
	leftAt := 0
	rightAt := 0
	for {
		if leftAt >= len(left) || rightAt >= len(right) {
			if leftAt >= len(left) && rightAt >= len(right) {
				return 0
			}
			if leftAt >= len(left) {
				return -1
			}
			return 1
		}
		leftEnd := leftAt
		for leftEnd < len(left) && left[leftEnd] != '.' {
			leftEnd++
		}
		rightEnd := rightAt
		for rightEnd < len(right) && right[rightEnd] != '.' {
			rightEnd++
		}
		leftNumeric := moduleVersionTextNumeric(left, leftAt, leftEnd)
		rightNumeric := moduleVersionTextNumeric(right, rightAt, rightEnd)
		comparison := 0
		if leftNumeric && rightNumeric {
			comparison = compareModuleNumericText(left[leftAt:leftEnd], right[rightAt:rightEnd])
		} else if leftNumeric {
			comparison = -1
		} else if rightNumeric {
			comparison = 1
		} else {
			comparison = compareModuleVersionText(left[leftAt:leftEnd], right[rightAt:rightEnd])
		}
		if comparison != 0 {
			return comparison
		}
		leftAt = leftEnd + 1
		rightAt = rightEnd + 1
	}
}

func moduleVersionTextNumeric(text string, start int, end int) bool {
	if start >= end {
		return false
	}
	for i := start; i < end; i++ {
		if text[i] < '0' || text[i] > '9' {
			return false
		}
	}
	return true
}

func compareModuleNumericText(left string, right string) int {
	for len(left) > 1 && left[0] == '0' {
		left = left[1:]
	}
	for len(right) > 1 && right[0] == '0' {
		right = right[1:]
	}
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	return compareModuleVersionText(left, right)
}

func unsupportedPackage(importPath string) load.PackageRef {
	return load.PackageRef{Kind: load.PackageUnsupported, ImportPath: importPath, Ok: false, Error: load.ResolveErrUnsupported}
}

func longestModuleRequirement(requirements []load.ModuleVersion, importPath string) (load.ModuleVersion, bool) {
	best := -1
	for i := 0; i < len(requirements); i++ {
		if importPath == requirements[i].Path || load.HasImportPrefix(importPath, requirements[i].Path) {
			if best < 0 || len(requirements[i].Path) > len(requirements[best].Path) {
				best = i
			}
		}
	}
	if best < 0 {
		return load.ModuleVersion{}, false
	}
	return requirements[best], true
}

func moduleVersionExcluded(config *load.ModuleConfig, version load.ModuleVersion) bool {
	if config == nil {
		return false
	}
	for i := 0; i < len(config.Excludes); i++ {
		if config.Excludes[i].Path == version.Path && config.Excludes[i].Version == version.Version {
			return true
		}
	}
	return false
}

func findModuleReplacement(config *load.ModuleConfig, version load.ModuleVersion) (load.ModuleReplace, bool) {
	if config == nil {
		return load.ModuleReplace{}, false
	}
	withoutVersion := -1
	for i := 0; i < len(config.Replaces); i++ {
		replacement := config.Replaces[i]
		if replacement.OldPath != version.Path {
			continue
		}
		if replacement.OldVersion == version.Version {
			return replacement, true
		}
		if replacement.OldVersion == "" {
			withoutVersion = i
		}
	}
	if withoutVersion >= 0 {
		return config.Replaces[withoutVersion], true
	}
	return load.ModuleReplace{}, false
}

func (c *sourceCollector) moduleByPath(path string) (load.Module, bool) {
	for i := 0; i < len(c.modules); i++ {
		if c.modules[i].Path == path {
			return c.modules[i], true
		}
	}
	return load.Module{}, false
}

func (c *sourceCollector) crossesNestedModule(owner load.Module, dir string) bool {
	dir = load.CleanPath(dir)
	root := load.CleanPath(owner.Root)
	for dir != root {
		if _, ok := c.fs.ReadFile(load.JoinPath(dir, "go.mod")); ok {
			return true
		}
		next := load.DirPath(dir)
		_, within := load.RelPath(root, next)
		if next == dir || !within {
			return true
		}
		dir = next
	}
	return false
}

func escapeModuleCachePath(path string) string {
	out := make([]byte, 0, len(path))
	for i := 0; i < len(path); i++ {
		c := path[i]
		if c >= 'A' && c <= 'Z' {
			out = append(out, '!')
			out = append(out, c-'A'+'a')
		} else {
			out = append(out, c)
		}
	}
	return string(out)
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

// SourceFileEnabled reports whether a source filename and its build constraints
// select the file for a target. Interactive tooling uses the same selection
// rules as compilation when discovering importable packages.
func SourceFileEnabled(name string, src []byte, target string, tags []string) (bool, bool) {
	if !isGoSourceName(name) || !sourceFilenameEnabled(name, target) {
		return false, true
	}
	return sourceConstraintsEnabled(src, target, tags)
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

func filterSourcesForOptions(files []load.SourceFile, workDir string, options Options) ([]load.SourceFile, string, int) {
	if len(options.Files) == 0 {
		filtered, errorPath, ok := filterSourcesForTargetTags(files, options.Target, options.Tags)
		if !ok {
			return nil, errorPath, SourceErrBuildConstraint
		}
		return filtered, "", SourceOK
	}
	rootDir := load.DirPath(load.JoinPath(workDir, options.Files[0]))
	var selected []string
	for i := 0; i < len(options.Files); i++ {
		path := load.JoinPath(workDir, options.Files[i])
		if load.DirPath(path) != rootDir {
			return nil, path, SourceErrFileDirectory
		}
		if isGoSourceName(load.BasePath(path)) {
			selected = append(selected, path)
		}
	}
	var out []load.SourceFile
	for i := 0; i < len(files); i++ {
		path := load.CleanPath(files[i].Path)
		if load.DirPath(path) == rootDir && isGoSourceName(load.BasePath(path)) {
			if findString(selected, path) >= 0 {
				out = append(out, files[i])
			}
			continue
		}
		filtered, errorPath, ok := filterSourcesForTargetTags(files[i:i+1], options.Target, options.Tags)
		if !ok {
			return nil, errorPath, SourceErrBuildConstraint
		}
		out = append(out, filtered...)
	}
	if len(selected) == 0 {
		return nil, options.Files[0], SourceErrFileListEmpty
	}
	return out, "", SourceOK
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
	return stringInBuildList(tag, "aix android browser darwin dragonfly freebsd hurd illumos ios js linux nacl netbsd openbsd plan9 solaris wasi wasip1 windows zos")
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
	if tag == "renvo" {
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
		return stringHasPrefix(target, "wasi/") || stringHasPrefix(target, "browser/")
	}
	if tag == "browser" {
		return stringHasPrefix(target, "browser/")
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
