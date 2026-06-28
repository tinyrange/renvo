package link

import (
	"fmt"
	"strconv"
	"strings"

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

type symbolEntry struct {
	key    string
	symbol unit.Symbol
}

type ownerEntry struct {
	name  string
	owner string
}

type declBodyEntry struct {
	name string
	body string
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
	exports, err := collectExports(units)
	if err != nil {
		return Plan{}, err
	}
	if err := validateResolvedReferences(units, exports); err != nil {
		return Plan{}, err
	}
	if err := validateEntrypoint(units); err != nil {
		return Plan{}, err
	}
	ordered := copyUnits(units)
	sortUnitsByImportPath(ordered)
	ordered = dependencyFirstUnits(ordered)
	return Plan{Units: ordered}, nil
}

func collectExports(units []unit.Unit) ([]symbolEntry, error) {
	var exports []symbolEntry
	for uIndex := 0; uIndex < len(units); uIndex++ {
		u := units[uIndex]
		unitExports := u.Exports
		for symIndex := 0; symIndex < len(unitExports); symIndex++ {
			sym := unitExports[symIndex]
			key := symbolKey(sym.ImportPath, sym.Name)
			if existing, ok := symbolEntryValue(exports, key); ok && existing.UnitName != sym.UnitName {
				return nil, fmt.Errorf("duplicate export %s.%s", sym.ImportPath, sym.Name)
			}
			if !hasSymbolEntry(exports, key) {
				exports = append(exports, symbolEntry{key: key, symbol: sym})
			}
		}
	}
	return exports, nil
}

func validateResolvedReferences(units []unit.Unit, exports []symbolEntry) error {
	for uIndex := 0; uIndex < len(units); uIndex++ {
		u := units[uIndex]
		unitReferences := u.References
		for refIndex := 0; refIndex < len(unitReferences); refIndex++ {
			ref := unitReferences[refIndex]
			export, ok := symbolEntryValue(exports, symbolKey(ref.ImportPath, ref.Name))
			if !ok {
				return fmt.Errorf("%s: unresolved reference %s.%s", u.ImportPath, ref.ImportPath, ref.Name)
			}
			if export.UnitName != ref.UnitName {
				return fmt.Errorf("%s: reference %s.%s resolved to %s, expected %s", u.ImportPath, ref.ImportPath, ref.Name, export.UnitName, ref.UnitName)
			}
		}
	}
	return nil
}

func validateUnitMetadata(units []unit.Unit) error {
	for uIndex := 0; uIndex < len(units); uIndex++ {
		u := units[uIndex]
		if u.ImportPath == "" {
			return fmt.Errorf("empty unit import path")
		}
		if u.Package == "" {
			return fmt.Errorf("%s: empty unit package", u.ImportPath)
		}
		var imports []string
		unitImports := u.Imports
		for impIndex := 0; impIndex < len(unitImports); impIndex++ {
			imp := unitImports[impIndex]
			if imp == "" {
				return fmt.Errorf("%s: empty import metadata", u.ImportPath)
			}
			if containsString(imports, imp) {
				return fmt.Errorf("%s: duplicate import metadata %q", u.ImportPath, imp)
			}
			imports = append(imports, imp)
		}
		var exports []string
		unitExports := u.Exports
		for symIndex := 0; symIndex < len(unitExports); symIndex++ {
			sym := unitExports[symIndex]
			if sym.Name == "" || sym.UnitName == "" {
				return fmt.Errorf("%s: invalid export metadata", u.ImportPath)
			}
			if containsString(exports, sym.Name) {
				return fmt.Errorf("%s: duplicate export metadata %s", u.ImportPath, sym.Name)
			}
			exports = append(exports, sym.Name)
		}
		var refs []string
		unitReferences := u.References
		for symIndex := 0; symIndex < len(unitReferences); symIndex++ {
			sym := unitReferences[symIndex]
			if sym.ImportPath == "" || sym.Name == "" || sym.UnitName == "" {
				return fmt.Errorf("%s: invalid reference metadata", u.ImportPath)
			}
			key := symbolKey(sym.ImportPath, sym.Name)
			if containsString(refs, key) {
				return fmt.Errorf("%s: duplicate reference metadata %s.%s", u.ImportPath, sym.ImportPath, sym.Name)
			}
			refs = append(refs, key)
		}
	}
	return nil
}

func validateUniqueUnits(units []unit.Unit) error {
	var seen []string
	for i := 0; i < len(units); i++ {
		u := units[i]
		if containsString(seen, u.ImportPath) {
			return fmt.Errorf("duplicate unit: %s", u.ImportPath)
		}
		seen = append(seen, u.ImportPath)
	}
	return nil
}

func validateImports(units []unit.Unit) error {
	var present []string
	for i := 0; i < len(units); i++ {
		u := units[i]
		present = append(present, u.ImportPath)
	}
	for uIndex := 0; uIndex < len(units); uIndex++ {
		u := units[uIndex]
		unitImports := u.Imports
		for impIndex := 0; impIndex < len(unitImports); impIndex++ {
			imp := unitImports[impIndex]
			if !containsString(present, imp) {
				return fmt.Errorf("%s: missing imported unit %s", u.ImportPath, imp)
			}
		}
	}
	return nil
}

func validateReferencesDeclared(units []unit.Unit) error {
	for uIndex := 0; uIndex < len(units); uIndex++ {
		u := units[uIndex]
		var imports []string
		unitImports := u.Imports
		for impIndex := 0; impIndex < len(unitImports); impIndex++ {
			imp := unitImports[impIndex]
			imports = append(imports, imp)
		}
		unitReferences := u.References
		for refIndex := 0; refIndex < len(unitReferences); refIndex++ {
			ref := unitReferences[refIndex]
			if !containsString(imports, ref.ImportPath) {
				return fmt.Errorf("%s: reference %s.%s missing import metadata", u.ImportPath, ref.ImportPath, ref.Name)
			}
		}
	}
	return nil
}

func validateExportOwnership(units []unit.Unit) error {
	for uIndex := 0; uIndex < len(units); uIndex++ {
		u := units[uIndex]
		unitExports := u.Exports
		for symIndex := 0; symIndex < len(unitExports); symIndex++ {
			sym := unitExports[symIndex]
			if sym.ImportPath != u.ImportPath {
				return fmt.Errorf("%s: export %s belongs to %s", u.ImportPath, sym.Name, sym.ImportPath)
			}
		}
	}
	return nil
}

func validateExportsDeclared(units []unit.Unit) error {
	for uIndex := 0; uIndex < len(units); uIndex++ {
		u := units[uIndex]
		unitExports := u.Exports
		unitDecls := u.Decls
		for symIndex := 0; symIndex < len(unitExports); symIndex++ {
			sym := unitExports[symIndex]
			found := false
			for declIndex := 0; declIndex < len(unitDecls); declIndex++ {
				decl := unitDecls[declIndex]
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
	for uIndex := 0; uIndex < len(units); uIndex++ {
		u := units[uIndex]
		unitDecls := u.Decls
		for declIndex := 0; declIndex < len(unitDecls); declIndex++ {
			decl := unitDecls[declIndex]
			if decl.UnitName == "" {
				continue
			}
			if decl.Body == "" {
				return fmt.Errorf("%s: declaration %s has empty body", u.ImportPath, decl.Name)
			}
			declBody := decl.Body
			bodyStart := 0
			for bodyStart < len(declBody) {
				c := declBody[bodyStart]
				if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
					break
				}
				bodyStart++
			}
			body := declBody[bodyStart:]
			if decl.Kind == "func" && !strings.HasPrefix(body, "func "+decl.UnitName) {
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
	var owners []ownerEntry
	for uIndex := 0; uIndex < len(units); uIndex++ {
		u := units[uIndex]
		unitDecls := u.Decls
		for declIndex := 0; declIndex < len(unitDecls); declIndex++ {
			decl := unitDecls[declIndex]
			if decl.UnitName == "" {
				continue
			}
			if owner, ok := ownerEntryValue(owners, decl.UnitName); ok {
				return fmt.Errorf("%s: duplicate declaration symbol %s already declared in %s", u.ImportPath, decl.UnitName, owner)
			}
			owners = append(owners, ownerEntry{name: decl.UnitName, owner: u.ImportPath})
		}
	}
	return nil
}

func validateEntrypoint(units []unit.Unit) error {
	var found []string
	explicit := hasExplicitEntrypoint(units)
	for uIndex := 0; uIndex < len(units); uIndex++ {
		u := units[uIndex]
		if !unitCanProvideEntrypoint(u, explicit) {
			continue
		}
		unitDecls := u.Decls
		for declIndex := 0; declIndex < len(unitDecls); declIndex++ {
			decl := unitDecls[declIndex]
			if decl.Name == "appMain" && decl.UnitName != "" {
				if !appMainWrapperValid(&decl) {
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

func hasExplicitEntrypoint(units []unit.Unit) bool {
	for i := 0; i < len(units); i++ {
		if units[i].Entry {
			return true
		}
	}
	return false
}

func unitCanProvideEntrypoint(u unit.Unit, explicit bool) bool {
	if u.Package != "main" {
		return false
	}
	if explicit {
		return u.Entry
	}
	return true
}

func symbolKey(importPath string, name string) string {
	return importPath + "\x00" + name
}

func Source(plan Plan) []byte {
	out := make([]byte, 0, 4194304)
	reachable := reachableFunctionDecls(plan)
	explicit := hasExplicitEntrypoint(plan.Units)
	out = appendString(out, "//go:build rtg\n\n")
	out = appendString(out, "// Code generated by rtg linker; DO NOT EDIT.\n")
	out = appendString(out, "package main\n\n")
	wrapperDone := false
	planUnits := plan.Units
	for uIndex := 0; uIndex < len(planUnits); uIndex++ {
		u := planUnits[uIndex]
		out = appendString(out, "// rtg:linked-unit ")
		out = appendString(out, quoteIfNeeded(u.ImportPath))
		out = append(out, '\n')
		unitDecls := u.Decls
		for declIndex := 0; declIndex < len(unitDecls); declIndex++ {
			decl := unitDecls[declIndex]
			if !shouldEmitDecl(decl, reachable) {
				continue
			}
			out = appendString(out, decl.Body)
			if decl.Body == "" || decl.Body[len(decl.Body)-1] != '\n' {
				out = append(out, '\n')
			}
			out = append(out, '\n')
			if !wrapperDone && unitCanProvideEntrypoint(u, explicit) && decl.Name == "appMain" && decl.UnitName != "" {
				out = appendAppMainWrapper(&decl, out)
				wrapperDone = true
			}
		}
	}
	return out
}

func SourceArtifact(plan Plan) Artifact {
	out := make([]byte, 0, 4194304)
	reachable := reachableFunctionDecls(plan)
	linkedUnits := linkedUnitNames(plan)
	reachableFunctions := sortedReachableFunctions(plan, reachable)
	var entrypoint unit.Symbol
	explicit := hasExplicitEntrypoint(plan.Units)
	out = appendString(out, "//go:build rtg\n\n")
	out = appendString(out, "// Code generated by rtg linker; DO NOT EDIT.\n")
	out = appendString(out, "package main\n\n")
	wrapperDone := false
	planUnits := plan.Units
	for uIndex := 0; uIndex < len(planUnits); uIndex++ {
		u := planUnits[uIndex]
		out = appendString(out, "// rtg:linked-unit ")
		out = appendString(out, quoteIfNeeded(u.ImportPath))
		out = append(out, '\n')
		unitDecls := u.Decls
		for declIndex := 0; declIndex < len(unitDecls); declIndex++ {
			decl := unitDecls[declIndex]
			if !shouldEmitDecl(decl, reachable) {
				continue
			}
			out = appendString(out, decl.Body)
			if decl.Body == "" || decl.Body[len(decl.Body)-1] != '\n' {
				out = append(out, '\n')
			}
			out = append(out, '\n')
			if !wrapperDone && unitCanProvideEntrypoint(u, explicit) && decl.Name == "appMain" && decl.UnitName != "" {
				out = appendAppMainWrapper(&decl, out)
				wrapperDone = true
				entrypoint = unit.Symbol{ImportPath: u.ImportPath, Name: decl.Name, UnitName: decl.UnitName}
			}
		}
	}
	return newArtifact(out, linkedUnits, reachableFunctions, entrypoint)
}

func newArtifact(source []byte, linkedUnits []string, reachableFunctions []string, entrypoint unit.Symbol) Artifact {
	return Artifact{
		Source:             source,
		LinkedUnits:        linkedUnits,
		ReachableFunctions: reachableFunctions,
		Entrypoint:         entrypoint,
	}
}

func appendString(out []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	return out
}

func copyUnits(values []unit.Unit) []unit.Unit {
	out := make([]unit.Unit, len(values))
	for i := 0; i < len(values); i++ {
		out[i] = values[i]
	}
	return out
}

func dependencyFirstUnits(units []unit.Unit) []unit.Unit {
	var ordered []unit.Unit
	state := make([]int, len(units))
	for i := 0; i < len(units); i++ {
		ordered = appendDependencyFirstUnit(units, state, ordered, i)
	}
	return ordered
}

func appendDependencyFirstUnit(units []unit.Unit, state []int, ordered []unit.Unit, index int) []unit.Unit {
	if index < 0 || index >= len(units) {
		return ordered
	}
	if state[index] == 2 {
		return ordered
	}
	if state[index] == 1 {
		return ordered
	}
	state[index] = 1
	imports := units[index].Imports
	for i := 0; i < len(imports); i++ {
		depIndex := unitIndexByImportPath(units, imports[i])
		if depIndex >= 0 {
			ordered = appendDependencyFirstUnit(units, state, ordered, depIndex)
		}
	}
	state[index] = 2
	value := units[index]
	ordered = append(ordered, value)
	return ordered
}

func unitIndexByImportPath(units []unit.Unit, importPath string) int {
	for i := 0; i < len(units); i++ {
		if units[i].ImportPath == importPath {
			return i
		}
	}
	return -1
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
	planUnits := plan.Units
	names := make([]string, 0, len(planUnits))
	for i := 0; i < len(planUnits); i++ {
		u := planUnits[i]
		names = append(names, u.ImportPath)
	}
	return names
}

func sortedReachableFunctions(plan Plan, reachable []string) []string {
	var names []string
	planUnits := plan.Units
	for uIndex := 0; uIndex < len(planUnits); uIndex++ {
		u := planUnits[uIndex]
		unitDecls := u.Decls
		for declIndex := 0; declIndex < len(unitDecls); declIndex++ {
			decl := unitDecls[declIndex]
			if decl.Kind == "func" && decl.UnitName != "" && containsString(reachable, decl.UnitName) {
				names = append(names, decl.UnitName)
			}
		}
	}
	sortStrings(names)
	return names
}

func sortUnitsByImportPath(units []unit.Unit) {
	for i := 1; i < len(units); i++ {
		value := units[i]
		j := i - 1
		for j >= 0 && stringGreater(units[j].ImportPath, value.ImportPath) {
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
		for j >= 0 && stringGreater(values[j], value) {
			values[j+1] = values[j]
			j = j - 1
		}
		values[j+1] = value
	}
}

func stringGreater(a string, b string) bool {
	i := 0
	for i < len(a) && i < len(b) {
		if a[i] > b[i] {
			return true
		}
		if a[i] < b[i] {
			return false
		}
		i = i + 1
	}
	return len(a) > len(b)
}

func containsString(values []string, value string) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return true
		}
	}
	return false
}

func hasSymbolEntry(values []symbolEntry, key string) bool {
	for i := 0; i < len(values); i++ {
		if values[i].key == key {
			return true
		}
	}
	return false
}

func symbolEntryValue(values []symbolEntry, key string) (unit.Symbol, bool) {
	for i := 0; i < len(values); i++ {
		if values[i].key == key {
			return values[i].symbol, true
		}
	}
	return unit.Symbol{}, false
}

func ownerEntryValue(values []ownerEntry, name string) (string, bool) {
	for i := 0; i < len(values); i++ {
		if values[i].name == name {
			return values[i].owner, true
		}
	}
	return "", false
}

func declBody(values []declBodyEntry, name string) string {
	for i := 0; i < len(values); i++ {
		if values[i].name == name {
			return values[i].body
		}
	}
	return ""
}

func shouldEmitDecl(decl unit.Decl, reachable []string) bool {
	if decl.Kind != "func" || decl.UnitName == "" {
		return true
	}
	return containsString(reachable, decl.UnitName)
}

func reachableFunctionDecls(plan Plan) []string {
	var bodies []declBodyEntry
	var queue []string
	var candidates []string
	explicit := hasExplicitEntrypoint(plan.Units)
	planUnits := plan.Units
	for uIndex := 0; uIndex < len(planUnits); uIndex++ {
		u := planUnits[uIndex]
		unitDecls := u.Decls
		for declIndex := 0; declIndex < len(unitDecls); declIndex++ {
			decl := unitDecls[declIndex]
			if decl.Kind != "func" || decl.UnitName == "" {
				continue
			}
			bodies = append(bodies, declBodyEntry{name: decl.UnitName, body: decl.Body})
			candidates = append(candidates, decl.UnitName)
			if unitCanProvideEntrypoint(u, explicit) && decl.Name == "appMain" {
				queue = append(queue, decl.UnitName)
			}
		}
	}
	var reachable []string
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if containsString(reachable, name) {
			continue
		}
		reachable = append(reachable, name)
		body := declBody(bodies, name)
		for i := 0; i < len(candidates); i++ {
			candidate := candidates[i]
			if !containsString(reachable, candidate) && bodyReferencesSymbol(body, candidate) {
				queue = append(queue, candidate)
			}
		}
	}
	return reachable
}

func bodyReferencesSymbol(body string, symbol string) bool {
	start := 0
	for start < len(body) {
		at := strings.Index(body[start:], symbol)
		if at < 0 {
			return false
		}
		at = at + start
		end := at + len(symbol)
		if !isIdentifierByteBefore(body, at) && !isIdentifierByteAfter(body, end) {
			return true
		}
		start = end
	}
	return false
}

func isIdentifierByteBefore(s string, pos int) bool {
	if pos <= 0 {
		return false
	}
	return isIdentifierByte(s[pos-1])
}

func isIdentifierByteAfter(s string, pos int) bool {
	if pos >= len(s) {
		return false
	}
	return isIdentifierByte(s[pos])
}

func isIdentifierByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func appMainWrapperValid(decl *unit.Decl) bool {
	return appMainWrapperSignature(decl) != ""
}

func appendAppMainWrapper(decl *unit.Decl, out []byte) []byte {
	signature := appMainWrapperSignature(decl)
	if signature == "" {
		return out
	}
	close := matchingParen(signature)
	params := signature[:close+1]
	result := signature[close+1:]
	for len(result) > 0 {
		c := result[0]
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			break
		}
		result = result[1:]
	}
	args, ok := argumentNames(params)
	if !ok {
		return out
	}
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
	return out
}

func appMainWrapperSignature(decl *unit.Decl) string {
	prefix := "func " + decl.UnitName
	declBody := decl.Body
	bodyStart := 0
	for bodyStart < len(declBody) {
		c := declBody[bodyStart]
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			break
		}
		bodyStart++
	}
	body := declBody[bodyStart:]
	if !strings.HasPrefix(body, prefix) {
		return ""
	}
	brace := strings.Index(body, "{")
	if brace < 0 {
		return ""
	}
	signature := body[len(prefix):brace]
	if !strings.HasPrefix(signature, "(") {
		return ""
	}
	close := matchingParen(signature)
	if close < 0 {
		return ""
	}
	for len(signature) > 0 {
		c := signature[len(signature)-1]
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			break
		}
		signature = signature[:len(signature)-1]
	}
	close = matchingParen(signature)
	if close < 0 {
		return ""
	}
	_, ok := argumentNames(signature[:close+1])
	if !ok {
		return ""
	}
	return signature
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
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) == 0 {
			return "", false
		}
		if len(fields) == 1 {
			pending = append(pending, fields[0])
			continue
		}
		for j := 0; j < len(pending); j++ {
			name := pending[j]
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
