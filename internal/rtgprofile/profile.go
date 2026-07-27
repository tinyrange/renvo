// Package rtgprofile is the compact system-profile projection of the RTG
// language. The full definition compiler delegates system declarations here,
// allowing ordinary driver users to retain system constraints without linking
// the machine-definition compiler.
package rtgprofile

type Profile struct {
	Name        string
	Target      string
	BinaryLimit int
	ArenaSize   int
}

type Diagnostic struct {
	Offset  int
	Message string
}

type token struct {
	start int
	end   int
	kind  int
}

const (
	tokenEOF = iota
	tokenIdent
	tokenString
	tokenNumber
	tokenOperator
)

func Parse(source []byte) (Profile, Diagnostic, bool) {
	var profile Profile
	tokens, diagnostic, ok := scan(source)
	if !ok {
		return profile, diagnostic, false
	}
	at := 0
	if tokenText(source, tokens[at]) != "system" {
		return profile, fail(tokens[at].start, "expected system declaration"), false
	}
	at++
	if tokens[at].kind != tokenString {
		return profile, fail(tokens[at].start, "expected non-empty quoted system name"), false
	}
	profile.Name, ok = quoted(tokenText(source, tokens[at]))
	if !ok || profile.Name == "" {
		return Profile{}, fail(tokens[at].start, "expected non-empty quoted system name"), false
	}
	at++
	if tokenText(source, tokens[at]) != "{" {
		return Profile{}, fail(tokens[at].start, "expected { and } around system fields"), false
	}
	at++
	haveTarget, haveBinary, haveArena := false, false, false
	for tokens[at].kind != tokenEOF && tokenText(source, tokens[at]) != "}" {
		if tokens[at].kind != tokenIdent {
			return Profile{}, fail(tokens[at].start, "expected field = value entries"), false
		}
		field := tokenText(source, tokens[at])
		fieldOffset := tokens[at].start
		at++
		if tokenText(source, tokens[at]) != "=" {
			return Profile{}, fail(tokens[at].start, "expected field = value entries"), false
		}
		at++
		if tokens[at].kind != tokenString && tokens[at].kind != tokenNumber {
			return Profile{}, fail(tokens[at].start, "expected field = value entries"), false
		}
		value := tokenText(source, tokens[at])
		at++
		if field == "target" {
			if haveTarget {
				return Profile{}, fail(fieldOffset, "duplicate target field"), false
			}
			profile.Target, ok = quoted(value)
			if !ok || profile.Target == "" {
				return Profile{}, fail(fieldOffset, "target must be a non-empty quoted string"), false
			}
			haveTarget = true
		} else if field == "binary" {
			if haveBinary {
				return Profile{}, fail(fieldOffset, "duplicate binary field"), false
			}
			profile.BinaryLimit, ok = parseSize(value)
			if !ok || profile.BinaryLimit <= 0 {
				return Profile{}, fail(fieldOffset, "binary must be a positive byte size"), false
			}
			haveBinary = true
		} else if field == "arena" {
			if haveArena {
				return Profile{}, fail(fieldOffset, "duplicate arena field"), false
			}
			profile.ArenaSize, ok = parseSize(value)
			if !ok || profile.ArenaSize < 256 || profile.ArenaSize > 1073741824 {
				return Profile{}, fail(fieldOffset, "arena must be between 256B and 1GiB"), false
			}
			haveArena = true
		} else {
			return Profile{}, fail(fieldOffset, "unknown system field "+field), false
		}
	}
	if tokens[at].kind == tokenEOF {
		return Profile{}, fail(tokens[at].start, "expected { and } around system fields"), false
	}
	at++
	if tokens[at].kind != tokenEOF {
		return Profile{}, fail(tokens[at].start, "system profile may only contain a system declaration"), false
	}
	if !haveTarget {
		return Profile{}, fail(0, "missing target field"), false
	}
	if !haveBinary {
		return Profile{}, fail(0, "missing binary field"), false
	}
	if !haveArena {
		return Profile{}, fail(0, "missing arena field"), false
	}
	return profile, Diagnostic{}, true
}

func scan(source []byte) ([]token, Diagnostic, bool) {
	var tokens []token
	at := 0
	for at < len(source) {
		ch := source[at]
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			at++
			continue
		}
		if ch == '#' || ch == '/' && at+1 < len(source) && source[at+1] == '/' {
			for at < len(source) && source[at] != '\n' {
				at++
			}
			continue
		}
		if ch == '/' && at+1 < len(source) && source[at+1] == '*' {
			start := at
			at += 2
			for at+1 < len(source) && (source[at] != '*' || source[at+1] != '/') {
				at++
			}
			if at+1 >= len(source) {
				return nil, fail(start, "unterminated block comment"), false
			}
			at += 2
			continue
		}
		start := at
		if isIdentStart(ch) {
			at++
			for at < len(source) && isIdentPart(source[at]) {
				at++
			}
			tokens = append(tokens, token{start: start, end: at, kind: tokenIdent})
			continue
		}
		if ch >= '0' && ch <= '9' {
			at++
			for at < len(source) && isIdentPart(source[at]) {
				at++
			}
			tokens = append(tokens, token{start: start, end: at, kind: tokenNumber})
			continue
		}
		if ch == '"' {
			at++
			for at < len(source) && source[at] != '"' && source[at] != '\n' {
				if source[at] == '\\' {
					at++
				}
				at++
			}
			if at >= len(source) || source[at] != '"' {
				return nil, fail(start, "unterminated quoted literal"), false
			}
			at++
			tokens = append(tokens, token{start: start, end: at, kind: tokenString})
			continue
		}
		at++
		tokens = append(tokens, token{start: start, end: at, kind: tokenOperator})
	}
	tokens = append(tokens, token{start: len(source), end: len(source), kind: tokenEOF})
	return tokens, Diagnostic{}, true
}

func quoted(text string) (string, bool) {
	if len(text) < 2 || text[0] != '"' || text[len(text)-1] != '"' {
		return "", false
	}
	for i := 1; i < len(text)-1; i++ {
		if text[i] == '"' || text[i] == '\\' {
			return "", false
		}
	}
	return text[1 : len(text)-1], true
}

func parseSize(text string) (int, bool) {
	multiplier := 1
	digits := len(text)
	if suffix(text, "KiB") {
		multiplier, digits = 1024, digits-3
	} else if suffix(text, "MiB") {
		multiplier, digits = 1024*1024, digits-3
	} else if suffix(text, "GiB") {
		multiplier, digits = 1024*1024*1024, digits-3
	} else if suffix(text, "B") {
		digits--
	}
	if digits <= 0 {
		return 0, false
	}
	value := 0
	for i := 0; i < digits; i++ {
		if text[i] < '0' || text[i] > '9' {
			return 0, false
		}
		digit := int(text[i] - '0')
		if value > (1073741824-digit)/10 {
			return 0, false
		}
		value = value*10 + digit
	}
	if value > 1073741824/multiplier {
		return 0, false
	}
	return value * multiplier, true
}

func suffix(text string, ending string) bool {
	return len(text) >= len(ending) && text[len(text)-len(ending):] == ending
}

func tokenText(source []byte, item token) string {
	return string(source[item.start:item.end])
}

func isIdentStart(ch byte) bool {
	return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch == '_'
}

func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || ch >= '0' && ch <= '9'
}

func fail(offset int, message string) Diagnostic {
	return Diagnostic{Offset: offset, Message: message}
}
