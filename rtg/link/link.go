package link

import (
	"fmt"
	"strconv"
	"strings"

	"j5.nz/rtg/rtg/scan"
	"j5.nz/rtg/rtg/unit"
)

type Plan struct {
	Units []unit.Unit
}

type Artifact struct {
	Source             []byte
	LinkedUnits        []string
	ReachableFunctions []string
	Entrypoint         unit.Symbol
}

func Build(units []unit.Unit) (Plan, error) {
	if err := validateUnitMetadata(units); err != nil {
		return Plan{}, err
	}
	if err := validateUniqueUnits(units); err != nil {
		return Plan{}, err
	}
	if err := validateImports(units); err != nil {
		return Plan{}, err
	}
	if err := validateReferencesDeclared(units); err != nil {
		return Plan{}, err
	}
	if err := validateExportOwnership(units); err != nil {
		return Plan{}, err
	}
	if err := validateExportsDeclared(units); err != nil {
		return Plan{}, err
	}
	if err := validateDeclSymbols(units); err != nil {
		return Plan{}, err
	}
	if err := validateUniqueDeclSymbols(units); err != nil {
		return Plan{}, err
	}
	exports := map[string]unit.Symbol{}
	for _, u := range units {
		for _, sym := range u.Exports {
			key := symbolKey(sym.ImportPath, sym.Name)
			if existing, ok := exports[key]; ok && existing.UnitName != sym.UnitName {
				return Plan{}, fmt.Errorf("duplicate export %s.%s", sym.ImportPath, sym.Name)
			}
			exports[key] = sym
		}
	}
	for _, u := range units {
		for _, ref := range u.References {
			export, ok := exports[symbolKey(ref.ImportPath, ref.Name)]
			if !ok {
				return Plan{}, fmt.Errorf("%s: unresolved reference %s.%s", u.ImportPath, ref.ImportPath, ref.Name)
			}
			if export.UnitName != ref.UnitName {
				return Plan{}, fmt.Errorf("%s: reference %s.%s resolved to %s, expected %s", u.ImportPath, ref.ImportPath, ref.Name, export.UnitName, ref.UnitName)
			}
		}
	}
	if err := validateEntrypoint(units); err != nil {
		return Plan{}, err
	}
	ordered := append([]unit.Unit(nil), units...)
	sortUnitsByImportPath(ordered)
	return Plan{Units: ordered}, nil
}

func validateUnitMetadata(units []unit.Unit) error {
	for _, u := range units {
		if u.ImportPath == "" {
			return fmt.Errorf("empty unit import path")
		}
		if u.Package == "" {
			return fmt.Errorf("%s: empty unit package", u.ImportPath)
		}
		imports := map[string]bool{}
		for _, imp := range u.Imports {
			if imp == "" {
				return fmt.Errorf("%s: empty import metadata", u.ImportPath)
			}
			if imports[imp] {
				return fmt.Errorf("%s: duplicate import metadata %q", u.ImportPath, imp)
			}
			imports[imp] = true
		}
		exports := map[string]bool{}
		for _, sym := range u.Exports {
			if sym.Name == "" || sym.UnitName == "" {
				return fmt.Errorf("%s: invalid export metadata", u.ImportPath)
			}
			if exports[sym.Name] {
				return fmt.Errorf("%s: duplicate export metadata %s", u.ImportPath, sym.Name)
			}
			exports[sym.Name] = true
		}
		refs := map[string]bool{}
		for _, sym := range u.References {
			if sym.ImportPath == "" || sym.Name == "" || sym.UnitName == "" {
				return fmt.Errorf("%s: invalid reference metadata", u.ImportPath)
			}
			key := symbolKey(sym.ImportPath, sym.Name)
			if refs[key] {
				return fmt.Errorf("%s: duplicate reference metadata %s.%s", u.ImportPath, sym.ImportPath, sym.Name)
			}
			refs[key] = true
		}
	}
	return nil
}

func validateUniqueUnits(units []unit.Unit) error {
	seen := map[string]bool{}
	for _, u := range units {
		if seen[u.ImportPath] {
			return fmt.Errorf("duplicate unit: %s", u.ImportPath)
		}
		seen[u.ImportPath] = true
	}
	return nil
}

func validateImports(units []unit.Unit) error {
	present := map[string]bool{}
	for _, u := range units {
		present[u.ImportPath] = true
	}
	for _, u := range units {
		for _, imp := range u.Imports {
			if !present[imp] {
				return fmt.Errorf("%s: missing imported unit %s", u.ImportPath, imp)
			}
		}
	}
	return nil
}

func validateReferencesDeclared(units []unit.Unit) error {
	for _, u := range units {
		imports := map[string]bool{}
		for _, imp := range u.Imports {
			imports[imp] = true
		}
		for _, ref := range u.References {
			if !imports[ref.ImportPath] {
				return fmt.Errorf("%s: reference %s.%s missing import metadata", u.ImportPath, ref.ImportPath, ref.Name)
			}
		}
	}
	return nil
}

func validateExportOwnership(units []unit.Unit) error {
	for _, u := range units {
		for _, sym := range u.Exports {
			if sym.ImportPath != u.ImportPath {
				return fmt.Errorf("%s: export %s belongs to %s", u.ImportPath, sym.Name, sym.ImportPath)
			}
		}
	}
	return nil
}

func validateExportsDeclared(units []unit.Unit) error {
	for _, u := range units {
		for _, sym := range u.Exports {
			found := false
			for _, decl := range u.Decls {
				if bodyReferencesSymbol(decl.Body, sym.UnitName) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("%s: export %s has no declaration for %s", u.ImportPath, sym.Name, sym.UnitName)
			}
		}
	}
	return nil
}

func validateDeclSymbols(units []unit.Unit) error {
	for _, u := range units {
		for _, decl := range u.Decls {
			if decl.UnitName == "" {
				continue
			}
			if decl.Body == "" {
				return fmt.Errorf("%s: declaration %s has empty body", u.ImportPath, decl.Name)
			}
			if decl.Kind == "func" && !strings.HasPrefix(decl.Body, "func "+decl.UnitName) {
				return fmt.Errorf("%s: declaration %s body does not define %s", u.ImportPath, decl.Name, decl.UnitName)
			}
			if decl.Kind == "const" && !strings.Contains(decl.Body, decl.UnitName) {
				return fmt.Errorf("%s: declaration %s body does not define %s", u.ImportPath, decl.Name, decl.UnitName)
			}
			if decl.Kind == "var" && !strings.Contains(decl.Body, decl.UnitName) {
				return fmt.Errorf("%s: declaration %s body does not define %s", u.ImportPath, decl.Name, decl.UnitName)
			}
			if decl.Kind == "type" && !strings.Contains(decl.Body, decl.UnitName) {
				return fmt.Errorf("%s: declaration %s body does not define %s", u.ImportPath, decl.Name, decl.UnitName)
			}
		}
	}
	return nil
}

func validateUniqueDeclSymbols(units []unit.Unit) error {
	owners := map[string]string{}
	for _, u := range units {
		for _, decl := range u.Decls {
			if decl.UnitName == "" {
				continue
			}
			if owner, ok := owners[decl.UnitName]; ok {
				return fmt.Errorf("%s: duplicate declaration symbol %s already declared in %s", u.ImportPath, decl.UnitName, owner)
			}
			owners[decl.UnitName] = u.ImportPath
		}
	}
	return nil
}

func validateEntrypoint(units []unit.Unit) error {
	var found []string
	for _, u := range units {
		if u.Package != "main" {
			continue
		}
		for _, decl := range u.Decls {
			if decl.Name == "appMain" && decl.UnitName != "" {
				if appMainWrapper(decl) == "" {
					return fmt.Errorf("%s: appMain declaration cannot be linked", u.ImportPath)
				}
				found = append(found, u.ImportPath)
			}
		}
	}
	if len(found) == 0 {
		return fmt.Errorf("missing entrypoint: package main must declare appMain")
	}
	if len(found) > 1 {
		return fmt.Errorf("multiple entrypoints: %s", strings.Join(found, ", "))
	}
	return nil
}

func symbolKey(importPath string, name string) string {
	return importPath + "\x00" + name
}

func Source(plan Plan) []byte {
	return SourceArtifact(plan).Source
}

func SourceArtifact(plan Plan) Artifact {
	var out []byte
	reachable := reachableFunctionDecls(plan)
	artifact := Artifact{
		LinkedUnits:        linkedUnitNames(plan),
		ReachableFunctions: sortedReachableFunctions(reachable),
	}
	out = appendString(out, "//go:build rtg\n\n")
	out = appendString(out, "// Code generated by rtg linker; DO NOT EDIT.\n")
	out = appendString(out, "package main\n\n")
	var wrapper string
	for _, u := range plan.Units {
		out = appendString(out, "// rtg:linked-unit ")
		out = appendString(out, quoteIfNeeded(u.ImportPath))
		out = append(out, '\n')
		for _, decl := range u.Decls {
			if !shouldEmitDecl(decl, reachable) {
				continue
			}
			out = appendString(out, decl.Body)
			if decl.Body == "" || decl.Body[len(decl.Body)-1] != '\n' {
				out = append(out, '\n')
			}
			out = append(out, '\n')
			if wrapper == "" && u.Package == "main" && decl.Name == "appMain" && decl.UnitName != "" {
				wrapper = appMainWrapper(decl)
				artifact.Entrypoint = unit.Symbol{ImportPath: u.ImportPath, Name: decl.Name, UnitName: decl.UnitName}
			}
		}
	}
	if wrapper != "" {
		out = appendString(out, wrapper)
	}
	artifact.Source = out
	return artifact
}

func appendString(out []byte, s string) []byte {
	return append(out, s...)
}

func quoteIfNeeded(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\r' || s[i] == '\n' || s[i] == '"' || s[i] == '\\' {
			return strconv.Quote(s)
		}
	}
	return s
}

func linkedUnitNames(plan Plan) []string {
	names := make([]string, 0, len(plan.Units))
	for _, u := range plan.Units {
		names = append(names, u.ImportPath)
	}
	return names
}

func sortedReachableFunctions(reachable map[string]bool) []string {
	var names []string
	for name := range reachable {
		names = append(names, name)
	}
	sortStrings(names)
	return names
}

func sortUnitsByImportPath(units []unit.Unit) {
	for i := 1; i < len(units); i++ {
		value := units[i]
		j := i - 1
		for j >= 0 && units[j].ImportPath > value.ImportPath {
			units[j+1] = units[j]
			j = j - 1
		}
		units[j+1] = value
	}
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		value := values[i]
		j := i - 1
		for j >= 0 && values[j] > value {
			values[j+1] = values[j]
			j = j - 1
		}
		values[j+1] = value
	}
}

func shouldEmitDecl(decl unit.Decl, reachable map[string]bool) bool {
	if decl.Kind != "func" || decl.UnitName == "" {
		return true
	}
	return reachable[decl.UnitName]
}

func reachableFunctionDecls(plan Plan) map[string]bool {
	bodies := map[string]string{}
	var queue []string
	for _, u := range plan.Units {
		for _, decl := range u.Decls {
			if decl.Kind != "func" || decl.UnitName == "" {
				continue
			}
			bodies[decl.UnitName] = decl.Body
			if u.Package == "main" && decl.Name == "appMain" {
				queue = append(queue, decl.UnitName)
			}
		}
	}
	reachable := map[string]bool{}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if reachable[name] {
			continue
		}
		reachable[name] = true
		body := bodies[name]
		for candidate := range bodies {
			if !reachable[candidate] && bodyReferencesSymbol(body, candidate) {
				queue = append(queue, candidate)
			}
		}
	}
	return reachable
}

func bodyReferencesSymbol(body string, symbol string) bool {
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		return strings.Contains(body, symbol)
	}
	for _, tok := range toks {
		if tok.Text == symbol {
			return true
		}
	}
	return false
}

func appMainWrapper(decl unit.Decl) string {
	prefix := "func " + decl.UnitName
	if !strings.HasPrefix(decl.Body, prefix) {
		return ""
	}
	brace := strings.Index(decl.Body, "{")
	if brace < 0 {
		return ""
	}
	signature := strings.TrimSpace(decl.Body[len(prefix):brace])
	if !strings.HasPrefix(signature, "(") {
		return ""
	}
	close := matchingParen(signature)
	if close < 0 {
		return ""
	}
	params := signature[:close+1]
	result := strings.TrimSpace(signature[close+1:])
	args, ok := argumentNames(params)
	if !ok {
		return ""
	}
	var out []byte
	out = appendString(out, "func appMain")
	out = appendString(out, signature)
	out = appendString(out, " {\n")
	out = append(out, '\t')
	if result != "" {
		out = appendString(out, "return ")
	}
	out = appendString(out, decl.UnitName)
	out = append(out, '(')
	out = appendString(out, args)
	out = appendString(out, ")\n")
	out = appendString(out, "}\n")
	return string(out)
}

func matchingParen(s string) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '(' {
			depth++
		} else if s[i] == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func argumentNames(params string) (string, bool) {
	if len(params) < 2 {
		return "", false
	}
	inner := strings.TrimSpace(params[1 : len(params)-1])
	if inner == "" {
		return "", true
	}
	parts := strings.Split(inner, ",")
	var names []string
	var pending []string
	for _, part := range parts {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) == 0 {
			return "", false
		}
		if len(fields) == 1 {
			pending = append(pending, fields[0])
			continue
		}
		for _, name := range pending {
			if !isArgumentIdentifier(name) {
				return "", false
			}
			names = append(names, name)
		}
		pending = nil
		name := fields[0]
		if !isArgumentIdentifier(name) {
			return "", false
		}
		names = append(names, name)
	}
	if len(pending) > 0 {
		return "", false
	}
	return strings.Join(names, ", "), true
}

func isArgumentIdentifier(name string) bool {
	if name == "" || name == "_" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if i == 0 {
			if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && c != '_' {
				return false
			}
			continue
		}
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
}
