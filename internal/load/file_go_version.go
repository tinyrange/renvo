package load

// GoVersionBefore reports whether a known language version predates minimum.
// An unset or malformed version is not evidence for rejecting a feature.
func GoVersionBefore(version, minimum string) bool {
	version = goLanguageVersion(version)
	minimum = goLanguageVersion(minimum)
	return version != "" && minimum != "" && version != minimum && laterGoLanguageVersion(version, minimum) == minimum
}

func effectiveFileGoVersion(pkg Package, src []byte) string {
	version := goLanguageVersion(pkg.GoVersion)
	if version == "" && (pkg.Ref.Kind == PackageInModule || pkg.Ref.Kind == PackageDependency) {
		version = "1.16"
	}
	if version == "" && pkg.Ref.Kind == PackageStandard {
		// Standard-library sources belong to this compiler's language baseline,
		// not to the importing application's module. Explicit file constraints
		// below still select a file-specific language version.
		version = CompilerGoVersion
	}
	// Only leading comments can supply a file language constraint.
	start := 0
	if len(src) >= 3 && src[0] == 0xef && src[1] == 0xbb && src[2] == 0xbf {
		start = 3
	}
	for pos := start; pos < len(src); {
		for pos < len(src) && (src[pos] == ' ' || src[pos] == '\t' || src[pos] == '\r' || src[pos] == '\n') {
			pos++
		}
		if pos+1 < len(src) && src[pos] == '/' && src[pos+1] == '*' {
			pos += 2
			for pos+1 < len(src) && (src[pos] != '*' || src[pos+1] != '/') {
				pos++
			}
			pos += 2
			continue
		}
		if pos+1 >= len(src) || src[pos] != '/' || src[pos+1] != '/' {
			break
		}
		end := pos
		for end < len(src) && src[end] != '\n' {
			end++
		}
		line := string(src[pos:end])
		if stringHasPrefix(line, "//go:build") && len(line) > 10 && (line[10] == ' ' || line[10] == '\t') {
			parser := fileVersionParser{src: line[10:], ok: true}
			minimum := parser.or(false, 0)
			parser.space()
			if parser.ok && parser.pos == len(parser.src) && minimum != "" {
				return laterGoLanguageVersion(minimum, "1.21")
			}
		}
		pos = end
	}
	return version
}

func goLanguageVersion(version string) string {
	if !validGoDirectiveVersion(version) {
		return ""
	}
	majorEnd := goVersionDigitsEnd(version, 0)
	minorEnd := goVersionDigitsEnd(version, majorEnd+1)
	return version[:minorEnd]
}

func laterGoLanguageVersion(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	ad, bd := goVersionDigitsEnd(a, 0), goVersionDigitsEnd(b, 0)
	if ad != bd {
		if ad > bd {
			return a
		}
		return b
	}
	if a[:ad] != b[:bd] {
		if a[:ad] > b[:bd] {
			return a
		}
		return b
	}
	if len(a)-ad != len(b)-bd {
		if len(a)-ad > len(b)-bd {
			return a
		}
		return b
	}
	if a > b {
		return a
	}
	return b
}

type fileVersionParser struct {
	src string
	pos int
	ok  bool
}

func (p *fileVersionParser) space() {
	for p.pos < len(p.src) && (p.src[p.pos] == ' ' || p.src[p.pos] == '\t' || p.src[p.pos] == '\r') {
		p.pos++
	}
}

func (p *fileVersionParser) take(op string) bool {
	p.space()
	if p.pos+len(op) > len(p.src) || p.src[p.pos:p.pos+len(op)] != op {
		return false
	}
	p.pos += len(op)
	return true
}

func combineFileGoVersion(a, b string, conjunction bool) string {
	later := laterGoLanguageVersion(a, b)
	if conjunction {
		return later
	}
	if later == a {
		return b
	}
	return a
}

func (p *fileVersionParser) or(negated bool, depth int) string {
	value := p.and(negated, depth)
	for p.take("||") {
		value = combineFileGoVersion(value, p.and(negated, depth), negated)
	}
	return value
}

func (p *fileVersionParser) and(negated bool, depth int) string {
	value := p.atom(negated, depth)
	for p.take("&&") {
		value = combineFileGoVersion(value, p.atom(negated, depth), !negated)
	}
	return value
}

func (p *fileVersionParser) atom(negated bool, depth int) string {
	if depth > 128 {
		p.ok = false
		return ""
	}
	if p.take("!") {
		return p.atom(!negated, depth+1)
	}
	if p.take("(") {
		value := p.or(negated, depth+1)
		if !p.take(")") {
			p.ok = false
		}
		return value
	}
	p.space()
	start := p.pos
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c == ' ' || c == '\t' || c == '\r' || c == '(' || c == ')' || c == '!' || c == '&' || c == '|' {
			break
		}
		p.pos++
	}
	if start == p.pos {
		p.ok = false
		return ""
	}
	tag := p.src[start:p.pos]
	if negated {
		return ""
	}
	if tag == "go1" {
		return "1.0"
	}
	if !stringHasPrefix(tag, "go1.") || len(tag) == 4 || goVersionDigitsEnd(tag, 4) != len(tag) {
		return ""
	}
	minor := 4
	for minor+1 < len(tag) && tag[minor] == '0' {
		minor++
	}
	return "1." + tag[minor:]
}
