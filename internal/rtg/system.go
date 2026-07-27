package rtg

// SystemProfile is the hosted resource contract shared by the definition
// language and the frontend driver.
type SystemProfile struct {
	Name        string
	Target      string
	BinaryLimit int
	ArenaSize   int
}

func ParseSystem(source []byte, filename string) (SystemProfile, Diagnostic, bool) {
	var profile SystemProfile
	document := Parse(source, filename)
	if !document.Ok {
		if len(document.Tokens) == 0 || document.Tokens[0].Kind != TokenIdent ||
			tokenText(source, document.Tokens[0]) != DeclSystem {
			return profile, systemDiagnostic(filename, source, 0, "expected system declaration"), false
		}
		if document.Diagnostics[0].Code == "RTG-PARSE-012" {
			return profile, systemDiagnostic(filename, source, document.Tokens[0].Start, "expected non-empty quoted system name"), false
		}
		return profile, document.Diagnostics[0], false
	}
	var declaration Declaration
	found := false
	for i := 0; i < len(document.Declarations); i++ {
		if document.Declarations[i].Kind != DeclSystem {
			return profile, systemDiagnostic(filename, source, document.Declarations[i].Start, "system profile may only contain a system declaration"), false
		}
		if found {
			return profile, systemDiagnostic(filename, source, document.Declarations[i].Start, "duplicate system declaration"), false
		}
		declaration = document.Declarations[i]
		found = true
	}
	if !found {
		return profile, systemDiagnostic(filename, source, 0, "expected system declaration"), false
	}
	if declaration.Name == "" {
		return profile, systemDiagnostic(filename, source, declaration.Start, "expected non-empty quoted system name"), false
	}
	profile.Name = declaration.Name
	haveTarget, haveBinary, haveArena := false, false, false
	for i := 0; i < len(declaration.Fields); i++ {
		field := declaration.Fields[i]
		value := string(source[field.ValueStart:field.ValueEnd])
		if field.Name == "target" {
			if haveTarget {
				return profile, systemDiagnostic(filename, source, field.Span.Start.Offset, "duplicate target field"), false
			}
			profile.Target = unquoteSimple(value)
			if profile.Target == "" {
				return profile, systemDiagnostic(filename, source, field.ValueStart, "target must be a non-empty quoted string"), false
			}
			haveTarget = true
		} else if field.Name == "binary" {
			if haveBinary {
				return profile, systemDiagnostic(filename, source, field.Span.Start.Offset, "duplicate binary field"), false
			}
			var ok bool
			profile.BinaryLimit, ok = parseSize(value)
			if !ok || profile.BinaryLimit <= 0 {
				return profile, systemDiagnostic(filename, source, field.ValueStart, "binary must be a positive byte size"), false
			}
			haveBinary = true
		} else if field.Name == "arena" {
			if haveArena {
				return profile, systemDiagnostic(filename, source, field.Span.Start.Offset, "duplicate arena field"), false
			}
			var ok bool
			profile.ArenaSize, ok = parseSize(value)
			if !ok || profile.ArenaSize < 256 || profile.ArenaSize > 1073741824 {
				return profile, systemDiagnostic(filename, source, field.ValueStart, "arena must be between 256B and 1GiB"), false
			}
			haveArena = true
		} else {
			return profile, systemDiagnostic(filename, source, field.Span.Start.Offset, "unknown system field "+field.Name), false
		}
	}
	if !haveTarget {
		return profile, systemDiagnostic(filename, source, declaration.BodyStart, "missing target field"), false
	}
	if !haveBinary {
		return profile, systemDiagnostic(filename, source, declaration.BodyStart, "missing binary field"), false
	}
	if !haveArena {
		return profile, systemDiagnostic(filename, source, declaration.BodyStart, "missing arena field"), false
	}
	return profile, Diagnostic{}, true
}

func parseSize(text string) (int, bool) {
	multiplier := 1
	digits := len(text)
	if hasSuffix(text, "KiB") {
		multiplier, digits = 1024, digits-3
	} else if hasSuffix(text, "MiB") {
		multiplier, digits = 1024*1024, digits-3
	} else if hasSuffix(text, "GiB") {
		multiplier, digits = 1024*1024*1024, digits-3
	} else if hasSuffix(text, "B") {
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

func hasSuffix(text string, suffix string) bool {
	return len(text) >= len(suffix) && text[len(text)-len(suffix):] == suffix
}

func systemDiagnostic(filename string, source []byte, offset int, message string) Diagnostic {
	return Diagnostic{
		Filename: filename,
		Span:     sourceSpan(source, offset, offset),
		Code:     "RTG-SYSTEM-001",
		Message:  message,
	}
}
