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

type reachEntry struct {
	name      string
	body      string
	next      int
	reachable bool
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
	exports := make([]symbolEntry, 0, unitSymbolCount(units, "export"))
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
		imports := make([]string, 0, len(u.Imports))
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
		exports := make([]string, 0, len(u.Exports))
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
		refs := make([]string, 0, len(u.References))
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
	seen := make([]string, 0, len(units))
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
	present := make([]string, 0, len(units))
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
		unitImports := u.Imports
		imports := make([]string, 0, len(unitImports))
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
				if decl.UnitName == sym.UnitName || (decl.UnitName == "" && bodyReferencesSymbol(decl.Body, sym.UnitName)) {
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
	owners := make([]ownerEntry, 0, unitDeclCount(units))
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
	found := make([]string, 0, 1)
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

func unitSymbolCount(units []unit.Unit, kind string) int {
	count := 0
	for i := 0; i < len(units); i++ {
		if kind == "export" {
			count += len(units[i].Exports)
		} else if kind == "ref" {
			count += len(units[i].References)
		}
	}
	return count
}

func unitDeclCount(units []unit.Unit) int {
	count := 0
	for i := 0; i < len(units); i++ {
		count += len(units[i].Decls)
	}
	return count
}

func unitFuncDeclCount(units []unit.Unit) int {
	count := 0
	for i := 0; i < len(units); i++ {
		decls := units[i].Decls
		for j := 0; j < len(decls); j++ {
			decl := decls[j]
			if decl.Kind == "func" && decl.UnitName != "" {
				count++
			}
		}
	}
	return count
}

func Source(plan Plan) []byte {
	reachable := reachableFunctionDecls(plan)
	out := make([]byte, 0, sourceCapacity(plan, reachable))
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
	reachable := reachableFunctionDecls(plan)
	out := make([]byte, 0, sourceCapacity(plan, reachable))
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

func sourceCapacity(plan Plan, reachable []string) int {
	capacity := 256
	planUnits := plan.Units
	for uIndex := 0; uIndex < len(planUnits); uIndex++ {
		u := planUnits[uIndex]
		capacity += len(u.ImportPath) + 24
		unitDecls := u.Decls
		for declIndex := 0; declIndex < len(unitDecls); declIndex++ {
			decl := unitDecls[declIndex]
			if shouldEmitDecl(decl, reachable) {
				capacity += len(decl.Body) + 2
			}
		}
	}
	if capacity < 1024 {
		return 1024
	}
	return capacity
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
	ordered := make([]unit.Unit, 0, len(units))
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
	names := make([]string, 0, len(reachable))
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
	entries := make([]reachEntry, 0, unitFuncDeclCount(plan.Units))
	buckets := make([]int, 4096)
	for i := 0; i < len(buckets); i++ {
		buckets[i] = -1
	}
	queue := make([]int, 0, 16)
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
			entryIndex := len(entries)
			bucket := hashStringModulo(decl.UnitName, len(buckets))
			entries = append(entries, reachEntry{name: decl.UnitName, body: decl.Body, next: buckets[bucket]})
			buckets[bucket] = entryIndex
			if unitCanProvideEntrypoint(u, explicit) && decl.Name == "appMain" {
				entries[entryIndex].reachable = true
				queue = append(queue, entryIndex)
			}
		}
	}
	reachable := make([]string, 0, len(entries))
	for len(queue) > 0 {
		index := queue[0]
		queue = queue[1:]
		if index < 0 || index >= len(entries) {
			continue
		}
		reachable = append(reachable, entries[index].name)
		body := entries[index].body
		for pos := 0; pos < len(body); {
			c := body[pos]
			if !isIdentifierStartByte(c) {
				pos++
				continue
			}
			start := pos
			pos++
			for pos < len(body) && isIdentifierByte(body[pos]) {
				pos++
			}
			candidate := reachEntryIndexByBodyRange(entries, buckets, body, start, pos)
			if candidate >= 0 && !entries[candidate].reachable {
				entries[candidate].reachable = true
				queue = append(queue, candidate)
			}
		}
	}
	return reachable
}

func reachEntryIndexByBodyRange(entries []reachEntry, buckets []int, body string, start int, end int) int {
	if len(buckets) == 0 {
		return -1
	}
	bucket := hashStringRangeModulo(body, start, end, len(buckets))
	for i := buckets[bucket]; i >= 0; i = entries[i].next {
		if stringRangeEquals(body, start, end, entries[i].name) {
			return i
		}
	}
	return -1
}

func hashStringModulo(value string, modulo int) int {
	if modulo <= 0 {
		return 0
	}
	hash := 0
	for i := 0; i < len(value); i++ {
		hash = (hash*33 + int(value[i])) % modulo
	}
	return hash
}

func hashStringRangeModulo(value string, start int, end int, modulo int) int {
	if modulo <= 0 {
		return 0
	}
	hash := 0
	for i := start; i < end; i++ {
		hash = (hash*33 + int(value[i])) % modulo
	}
	return hash
}

func stringRangeEquals(value string, start int, end int, name string) bool {
	if end-start != len(name) {
		return false
	}
	for i := 0; i < len(name); i++ {
		if value[start+i] != name[i] {
			return false
		}
	}
	return true
}

func isIdentifierStartByte(c byte) bool {
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
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
	names := make([]string, 0, len(parts))
	pending := make([]string, 0, len(parts))
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
