package load

func packageModuleGoVersion(module Module, ref PackageRef, dependencies []ModuleDependency) string {
	if ref.Kind == PackageInModule {
		return module.GoVersion
	}
	if ref.Kind != PackageDependency {
		return ""
	}
	path, version := "", ""
	for _, dependency := range dependencies {
		if (ref.ImportPath == dependency.Path || HasImportPrefix(ref.ImportPath, dependency.Path)) && len(dependency.Path) > len(path) {
			path, version = dependency.Path, dependency.GoVersion
		}
	}
	return version
}

func goDirectiveLineEnd(src []byte, pos int) bool {
	for {
		pos = skipGoModHorizontal(src, pos)
		if pos >= len(src) || src[pos] == '\n' || src[pos] == '\r' {
			return true
		}
		if pos+1 >= len(src) || src[pos] != '/' {
			return false
		}
		if src[pos+1] == '/' {
			return true
		}
		if src[pos+1] != '*' {
			return false
		}
		pos += 2
		for pos+1 < len(src) && (src[pos] != '*' || src[pos+1] != '/') {
			pos++
		}
		if pos+1 >= len(src) {
			return false
		}
		pos += 2
	}
}

// Match the module-file Go version grammar, preserving the spelling for later
// language/toolchain comparisons. Do not truncate potentially large components
// into machine integers while validating syntax.
func validGoDirectiveVersion(version string) bool {
	if len(version) == 0 || version[0] < '1' || version[0] > '9' {
		return false
	}
	pos := goVersionDigitsEnd(version, 0)
	if pos >= len(version) || version[pos] != '.' {
		return false
	}
	pos++
	end := goVersionDigitsEnd(version, pos)
	if end == pos || end-pos > 1 && version[pos] == '0' {
		return false
	}
	pos = end
	if pos < len(version) && version[pos] == '.' {
		pos++
		end = goVersionDigitsEnd(version, pos)
		if end == pos || end-pos > 1 && version[pos] == '0' {
			return false
		}
		pos = end
	}
	if pos == len(version) {
		return true
	}
	start := pos
	for pos < len(version) && version[pos] >= 'a' && version[pos] <= 'z' {
		pos++
	}
	if start == pos {
		return false
	}
	end = goVersionDigitsEnd(version, pos)
	return end > pos && end == len(version)
}

func goVersionDigitsEnd(version string, start int) int {
	for start < len(version) && version[start] >= '0' && version[start] <= '9' {
		start++
	}
	return start
}
