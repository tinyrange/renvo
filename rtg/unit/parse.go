package unit

import (
	"fmt"
	"strings"
)

func ParseSource(path string, src []byte) (Unit, error) {
	lines := strings.Split(string(src), "\n")
	var u Unit
	currentDecl := -1
	seenUnit := false
	seenImports := map[string]bool{}
	seenExports := map[string]bool{}
	seenRefs := map[string]bool{}
	for _, line := range lines {
		if strings.HasPrefix(line, "package ") && u.Package == "" {
			u.Package = strings.TrimSpace(strings.TrimPrefix(line, "package "))
			continue
		}
		if !strings.HasPrefix(line, "// rtg:") {
			if currentDecl >= 0 {
				u.Decls[currentDecl].Body += line
				u.Decls[currentDecl].Body += "\n"
			}
			continue
		}
		body := strings.TrimPrefix(line, "// rtg:")
		if strings.HasPrefix(body, "decl ") {
			decl, err := parseDecl(strings.TrimSpace(strings.TrimPrefix(body, "decl ")))
			if err != nil {
				return Unit{}, fmt.Errorf("%s: %w", path, err)
			}
			u.Decls = append(u.Decls, decl)
			currentDecl = len(u.Decls) - 1
			continue
		}
		currentDecl = -1
		if strings.HasPrefix(body, "unit ") {
			if seenUnit {
				return Unit{}, fmt.Errorf("%s: duplicate rtg unit metadata", path)
			}
			seenUnit = true
			u.ImportPath = strings.TrimSpace(strings.TrimPrefix(body, "unit "))
			if u.ImportPath == "" {
				return Unit{}, fmt.Errorf("%s: empty rtg unit metadata", path)
			}
			continue
		}
		if strings.HasPrefix(body, "import ") {
			imp, err := parseQuoted(strings.TrimSpace(strings.TrimPrefix(body, "import ")))
			if err != nil {
				return Unit{}, fmt.Errorf("%s: %w", path, err)
			}
			if imp == "" {
				return Unit{}, fmt.Errorf("%s: empty import metadata", path)
			}
			if seenImports[imp] {
				return Unit{}, fmt.Errorf("%s: duplicate import metadata %q", path, imp)
			}
			seenImports[imp] = true
			u.Imports = append(u.Imports, imp)
			continue
		}
		if strings.HasPrefix(body, "export ") {
			sym, err := parseNameArrow(strings.TrimSpace(strings.TrimPrefix(body, "export ")))
			if err != nil {
				return Unit{}, fmt.Errorf("%s: %w", path, err)
			}
			sym.ImportPath = u.ImportPath
			key := sym.Name
			if seenExports[key] {
				return Unit{}, fmt.Errorf("%s: duplicate export metadata %s", path, sym.Name)
			}
			seenExports[key] = true
			u.Exports = append(u.Exports, sym)
			continue
		}
		if strings.HasPrefix(body, "ref ") {
			sym, err := parseReference(strings.TrimSpace(strings.TrimPrefix(body, "ref ")))
			if err != nil {
				return Unit{}, fmt.Errorf("%s: %w", path, err)
			}
			key := sym.ImportPath + "\x00" + sym.Name
			if seenRefs[key] {
				return Unit{}, fmt.Errorf("%s: duplicate reference metadata %s.%s", path, sym.ImportPath, sym.Name)
			}
			seenRefs[key] = true
			u.References = append(u.References, sym)
			continue
		}
	}
	if u.ImportPath == "" {
		return Unit{}, fmt.Errorf("%s: missing rtg unit metadata", path)
	}
	if u.Package == "" {
		return Unit{}, fmt.Errorf("%s: missing package declaration", path)
	}
	return u, nil
}

func parseQuoted(s string) (string, error) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return "", fmt.Errorf("invalid quoted import %q", s)
	}
	var out []byte
	for i := 1; i < len(s)-1; i++ {
		if s[i] == '\\' {
			i++
			if i >= len(s)-1 {
				return "", fmt.Errorf("invalid quoted escape")
			}
		}
		out = append(out, s[i])
	}
	return string(out), nil
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
	firstSpace := strings.IndexByte(s, ' ')
	if firstSpace < 0 {
		return Symbol{}, fmt.Errorf("invalid reference metadata %q", s)
	}
	importPath := s[:firstSpace]
	if importPath == "" {
		return Symbol{}, fmt.Errorf("invalid reference metadata %q", s)
	}
	sym, err := parseNameArrow(strings.TrimSpace(s[firstSpace+1:]))
	if err != nil {
		return Symbol{}, err
	}
	sym.ImportPath = importPath
	return sym, nil
}

func parseDecl(s string) (Decl, error) {
	arrow := strings.Index(s, " => ")
	if arrow < 0 {
		parts := strings.Fields(s)
		if len(parts) < 2 {
			return Decl{}, fmt.Errorf("invalid declaration metadata %q", s)
		}
		return Decl{Kind: parts[0], Path: parts[len(parts)-1], Name: strings.Join(parts[1:len(parts)-1], " ")}, nil
	}
	left := strings.TrimSpace(s[:arrow])
	right := strings.TrimSpace(s[arrow+4:])
	rightParts := strings.Fields(right)
	if len(rightParts) < 2 {
		return Decl{}, fmt.Errorf("invalid declaration metadata %q", s)
	}
	leftParts := strings.Fields(left)
	if len(leftParts) < 2 {
		return Decl{}, fmt.Errorf("invalid declaration metadata %q", s)
	}
	return Decl{
		Kind:     leftParts[0],
		Name:     strings.Join(leftParts[1:], " "),
		UnitName: rightParts[0],
		Path:     strings.Join(rightParts[1:], " "),
	}, nil
}
