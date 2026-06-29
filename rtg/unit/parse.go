package unit

import (
	"fmt"
	"strconv"
	"strings"
)

type parseError string

func (err parseError) Error() string {
	return string(err)
}

func ParseSources(sources []SourceFile) ([]Unit, error) {
	units := make([]Unit, 0, len(sources))
	for i := 0; i < len(sources); i++ {
		source := sources[i]
		u, err := ParseSource(source.Path, source.Source)
		if err != nil {
			return nil, err
		}
		units = append(units, u)
	}
	return units, nil
}

func ParseSource(path string, src []byte) (Unit, error) {
	text := bytesToString(src)
	lines := strings.Split(text, "\n")
	if !hasRTGBuildConstraint(lines) {
		return Unit{}, fmt.Errorf("%s: missing rtg build constraint", path)
	}
	var u Unit
	currentDecl := -1
	seenUnit := false
	seenEntry := false
	var seenImports []string
	var seenExports []string
	var seenRefs []string
	currentBodyStart := -1
	currentBodyEnd := -1
	offset := 0
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		lineStart := offset
		lineEnd := lineStart + len(line)
		nextOffset := lineEnd + 1
		if nextOffset > len(text) {
			nextOffset = len(text)
		}
		if strings.HasPrefix(line, "package ") && u.Package == "" {
			u.Package = strings.TrimSpace(strings.TrimPrefix(line, "package "))
			offset = nextOffset
			continue
		}
		if !strings.HasPrefix(line, "// rtg:") {
			body := bodyRange(text, currentBodyStart, currentBodyEnd)
			if currentDecl >= 0 && strings.TrimSpace(body) == "" && strings.TrimSpace(line) == "" {
				offset = nextOffset
				continue
			}
			if currentDecl >= 0 {
				if currentBodyStart < 0 {
					currentBodyStart = lineStart
				}
				currentBodyEnd = nextOffset
			}
			offset = nextOffset
			continue
		}
		currentBody := bodyRange(text, currentBodyStart, currentBodyEnd)
		if currentDecl >= 0 && strings.TrimSpace(currentBody) != "" {
			decl := declAt(u, currentDecl)
			decl.Body = currentBody
			if !declBodyComplete(decl) {
				if currentBodyStart < 0 {
					currentBodyStart = lineStart
				}
				currentBodyEnd = nextOffset
				offset = nextOffset
				continue
			}
		}
		body := strings.TrimPrefix(line, "// rtg:")
		if !seenUnit && !strings.HasPrefix(body, "unit ") {
			return Unit{}, fmt.Errorf("%s: rtg metadata before unit declaration", path)
		}
		if currentDecl >= 0 && strings.TrimSpace(currentBody) == "" {
			return Unit{}, fmt.Errorf("%s: declaration metadata for %s has no body before next rtg metadata", path, declNameAt(u, currentDecl))
		}
		if strings.HasPrefix(body, "decl ") {
			if currentDecl >= 0 {
				setDeclBody(&u, currentDecl, currentBody)
			}
			decl, err := parseDecl(strings.TrimSpace(strings.TrimPrefix(body, "decl ")))
			if err != nil {
				return Unit{}, parseError(path + ": " + err.Error())
			}
			u.Decls = append(u.Decls, decl)
			currentDecl = len(u.Decls) - 1
			currentBodyStart = -1
			currentBodyEnd = -1
			offset = nextOffset
			continue
		}
		if currentDecl >= 0 {
			setDeclBody(&u, currentDecl, currentBody)
		}
		currentDecl = -1
		currentBodyStart = -1
		currentBodyEnd = -1
		if strings.HasPrefix(body, "unit ") {
			if seenUnit {
				return Unit{}, fmt.Errorf("%s: duplicate rtg unit metadata", path)
			}
			seenUnit = true
			importPath, err := unquoteMetadataField(strings.TrimSpace(strings.TrimPrefix(body, "unit ")))
			if err != nil {
				return Unit{}, fmt.Errorf("%s: invalid rtg unit metadata", path)
			}
			u.ImportPath = importPath
			if u.ImportPath == "" {
				return Unit{}, fmt.Errorf("%s: empty rtg unit metadata", path)
			}
			offset = nextOffset
			continue
		}
		if strings.TrimSpace(body) == "entrypoint" {
			if seenEntry {
				return Unit{}, fmt.Errorf("%s: duplicate entrypoint metadata", path)
			}
			seenEntry = true
			u.Entry = true
			offset = nextOffset
			continue
		}
		if strings.HasPrefix(body, "import ") {
			imp, err := parseQuoted(strings.TrimSpace(strings.TrimPrefix(body, "import ")))
			if err != nil {
				return Unit{}, parseError(path + ": " + err.Error())
			}
			if imp == "" {
				return Unit{}, fmt.Errorf("%s: empty import metadata", path)
			}
			if containsString(seenImports, imp) {
				return Unit{}, fmt.Errorf("%s: duplicate import metadata %q", path, imp)
			}
			seenImports = append(seenImports, imp)
			u.Imports = append(u.Imports, imp)
			offset = nextOffset
			continue
		}
		if strings.HasPrefix(body, "export ") {
			sym, err := parseNameArrow(strings.TrimSpace(strings.TrimPrefix(body, "export ")))
			if err != nil {
				return Unit{}, parseError(path + ": " + err.Error())
			}
			sym.ImportPath = u.ImportPath
			key := sym.Name
			if containsString(seenExports, key) {
				return Unit{}, fmt.Errorf("%s: duplicate export metadata %s", path, sym.Name)
			}
			seenExports = append(seenExports, key)
			u.Exports = append(u.Exports, sym)
			offset = nextOffset
			continue
		}
		if strings.HasPrefix(body, "ref ") {
			sym, err := parseReference(strings.TrimSpace(strings.TrimPrefix(body, "ref ")))
			if err != nil {
				return Unit{}, parseError(path + ": " + err.Error())
			}
			key := sym.ImportPath + "\x00" + sym.Name
			if containsString(seenRefs, key) {
				return Unit{}, fmt.Errorf("%s: duplicate reference metadata %s.%s", path, sym.ImportPath, sym.Name)
			}
			seenRefs = append(seenRefs, key)
			u.References = append(u.References, sym)
			offset = nextOffset
			continue
		}
		return Unit{}, fmt.Errorf("%s: unknown rtg metadata %q", path, strings.TrimSpace(body))
	}
	if currentDecl >= 0 {
		setDeclBody(&u, currentDecl, bodyRange(text, currentBodyStart, currentBodyEnd))
	}
	if u.ImportPath == "" {
		return Unit{}, fmt.Errorf("%s: missing rtg unit metadata", path)
	}
	if u.Package == "" {
		return Unit{}, fmt.Errorf("%s: missing package declaration", path)
	}
	decls := u.Decls
	for i := 0; i < len(decls); i++ {
		decl := decls[i]
		if strings.TrimSpace(decl.Body) == "" {
			return Unit{}, fmt.Errorf("%s: declaration metadata for %s has no body", path, decl.Name)
		}
	}
	return u, nil
}

func bytesToString(data []byte) string {
	var out []byte
	for i := 0; i < len(data); i++ {
		out = append(out, data[i])
	}
	return string(out)
}

func declAt(u Unit, index int) Decl {
	decls := u.Decls
	return decls[index]
}

func declBodyAt(u Unit, index int) string {
	decl := declAt(u, index)
	return decl.Body
}

func declNameAt(u Unit, index int) string {
	decl := declAt(u, index)
	return decl.Name
}

func bodyRange(text string, start int, end int) string {
	if start < 0 || end < start {
		return ""
	}
	if end > len(text) {
		end = len(text)
	}
	var out []byte
	for i := start; i < end; i++ {
		out = append(out, text[i])
	}
	return string(out)
}

func setDeclBody(u *Unit, index int, body string) {
	decls := u.Decls
	decl := decls[index]
	decl.Body = body
	decls[index] = decl
	u.Decls = decls
}

func appendDeclBodyLine(u *Unit, index int, line string) {
	decls := u.Decls
	decl := decls[index]
	body := decl.Body
	body = body + line
	body = body + "\n"
	decl.Body = body
	decls[index] = decl
	u.Decls = decls
}

func containsString(values []string, value string) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return true
		}
	}
	return false
}

func declBodyComplete(decl Decl) bool {
	return declBodyDelimitersComplete(decl.Kind, decl.Body)
}

func declBodyDelimitersComplete(kind string, body string) bool {
	depth := 0
	paren := 0
	brack := 0
	seenFuncBody := false
	i := 0
	for i < len(body) {
		c := body[i]
		if c == '/' && i+1 < len(body) && body[i+1] == '/' {
			i += 2
			for i < len(body) && body[i] != '\n' {
				i++
			}
			continue
		}
		if c == '"' || c == '`' || c == '\'' {
			quote := c
			i++
			for i < len(body) {
				if quote != '`' && body[i] == '\\' {
					i += 2
					continue
				}
				if body[i] == quote {
					i++
					break
				}
				i++
			}
			continue
		}
		if c == '(' {
			paren++
		} else if c == ')' {
			paren--
		} else if c == '[' {
			brack++
		} else if c == ']' {
			brack--
		} else if c == '{' {
			depth++
			seenFuncBody = true
		} else if c == '}' {
			depth--
		}
		if paren < 0 || brack < 0 || depth < 0 {
			return false
		}
		i++
	}
	if kind == "func" {
		return seenFuncBody && depth == 0
	}
	return paren == 0 && brack == 0 && depth == 0
}

func hasRTGBuildConstraint(lines []string) bool {
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		return trimmed == "//go:build rtg"
	}
	return false
}

func parseQuoted(s string) (string, error) {
	value, err := strconv.Unquote(s)
	if err != nil {
		return "", fmt.Errorf("invalid quoted import %q", s)
	}
	return value, nil
}

func parseNameArrow(s string) (Symbol, error) {
	parts := strings.Split(s, " => ")
	if len(parts) != 2 {
		return Symbol{}, fmt.Errorf("invalid symbol metadata %q", s)
	}
	name := strings.TrimSpace(parts[0])
	unitName := strings.TrimSpace(parts[1])
	if name == "" || unitName == "" {
		return Symbol{}, fmt.Errorf("invalid symbol metadata %q", s)
	}
	return Symbol{Name: name, UnitName: unitName}, nil
}

func parseReference(s string) (Symbol, error) {
	field, rest, err := splitFirstMetadataField(s)
	if err != nil {
		return Symbol{}, fmt.Errorf("invalid reference metadata %q", s)
	}
	importPath, err := unquoteMetadataField(field)
	if err != nil {
		return Symbol{}, fmt.Errorf("invalid reference metadata %q", s)
	}
	if importPath == "" {
		return Symbol{}, fmt.Errorf("invalid reference metadata %q", s)
	}
	sym, err := parseNameArrow(strings.TrimSpace(rest))
	if err != nil {
		return Symbol{}, err
	}
	sym.ImportPath = importPath
	return sym, nil
}

func parseDecl(s string) (Decl, error) {
	arrow := strings.Index(s, " => ")
	if arrow < 0 {
		parts, err := metadataFields(s)
		if err != nil {
			return Decl{}, fmt.Errorf("invalid declaration metadata %q", s)
		}
		if len(parts) < 2 {
			return Decl{}, fmt.Errorf("invalid declaration metadata %q", s)
		}
		path, err := unquoteMetadataField(parts[len(parts)-1])
		if err != nil {
			return Decl{}, fmt.Errorf("invalid declaration metadata %q", s)
		}
		kind := parts[0]
		name := strings.Join(parts[1:len(parts)-1], " ")
		var decl Decl
		decl.Kind = kind
		decl.Path = path
		decl.Name = name
		return decl, nil
	}
	left := strings.TrimSpace(s[:arrow])
	right := strings.TrimSpace(s[arrow+4:])
	rightParts, err := metadataFields(right)
	if err != nil {
		return Decl{}, fmt.Errorf("invalid declaration metadata %q", s)
	}
	if len(rightParts) < 2 {
		return Decl{}, fmt.Errorf("invalid declaration metadata %q", s)
	}
	leftParts := strings.Fields(left)
	if len(leftParts) < 2 {
		return Decl{}, fmt.Errorf("invalid declaration metadata %q", s)
	}
	path, err := unquoteMetadataField(rightParts[len(rightParts)-1])
	if err != nil {
		return Decl{}, fmt.Errorf("invalid declaration metadata %q", s)
	}
	if len(rightParts) > 2 {
		path = strings.Join(rightParts[1:], " ")
	}
	kind := leftParts[0]
	name := strings.Join(leftParts[1:], " ")
	unitName := rightParts[0]
	var decl Decl
	decl.Kind = kind
	decl.Name = name
	decl.UnitName = unitName
	decl.Path = path
	return decl, nil
}

func metadataFields(s string) ([]string, error) {
	var fields []string
	for i := 0; i < len(s); {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r') {
			i++
		}
		if i >= len(s) {
			break
		}
		start := i
		if s[i] == '"' {
			i++
			for i < len(s) {
				if s[i] == '\\' {
					i += 2
					continue
				}
				if s[i] == '"' {
					i++
					fields = append(fields, s[start:i])
					break
				}
				i++
			}
			if i > len(s) || len(fields) == 0 || fields[len(fields)-1] != s[start:i] {
				return nil, fmt.Errorf("invalid metadata field")
			}
			continue
		}
		for i < len(s) && s[i] != ' ' && s[i] != '\t' && s[i] != '\r' {
			i++
		}
		fields = append(fields, s[start:i])
	}
	return fields, nil
}

func splitFirstMetadataField(s string) (string, string, error) {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r') {
		i++
	}
	if i >= len(s) {
		return "", "", fmt.Errorf("missing metadata field")
	}
	start := i
	if s[i] == '"' {
		i++
		for i < len(s) {
			if s[i] == '\\' {
				i += 2
				continue
			}
			if s[i] == '"' {
				i++
				return s[start:i], s[i:], nil
			}
			i++
		}
		return "", "", fmt.Errorf("invalid metadata field")
	}
	for i < len(s) && s[i] != ' ' && s[i] != '\t' && s[i] != '\r' {
		i++
	}
	return s[start:i], s[i:], nil
}

func unquoteMetadataField(field string) (string, error) {
	if len(field) == 0 || field[0] != '"' {
		return field, nil
	}
	return strconv.Unquote(field)
}
