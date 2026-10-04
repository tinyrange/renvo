package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

// A package declaration or import can hide a universe type even though its
// checked identity is still required by inference. An alias in that same
// package would capture the hidden name too. Put only the needed aliases in a
// clean compiler-owned package, using ordinary concrete Go declarations.
func (s *genericSpecializer) builtinTypeText(name string, into int) string {
	e := s.environment
	shadowed := e.lookup(into, name) >= 0 || e.packageValueName(into, name)
	for file := range e.graph.Packages[into].Files {
		if e.imported(genericTypeScope{pkg: into, file: file}, name) >= 0 {
			shadowed = true
		}
	}
	if !shadowed {
		return name
	}
	found := false
	for _, builtin := range s.builtins {
		if builtin == name {
			found = true
		}
	}
	if !found {
		s.builtins = append(s.builtins, name)
	}
	return s.qualify(into, len(e.graph.Packages), "RenvoBuiltin_"+name)
}

func (s *genericSpecializer) builtinPackage() load.Package {
	path := "renvo.generated/generic/builtins"
	for {
		used := false
		for _, pkg := range s.environment.graph.Packages {
			if pkg.Ref.ImportPath == path {
				used = true
			}
		}
		if !used {
			break
		}
		path += "_"
	}
	source := "package renvo_generic_builtins\n"
	for _, name := range s.builtins {
		source += "type RenvoBuiltin_" + name + " = " + name + "\n"
	}
	src := []byte(source)
	return load.Package{
		Ref:  load.PackageRef{Kind: load.PackageInModule, ImportPath: path, Ok: true},
		Name: "renvo_generic_builtins", GoVersion: load.CompilerGoVersion, Ok: true,
		Files: []load.ParsedFile{{Path: path + "/types.go", GoVersion: load.CompilerGoVersion, Src: src, File: syntax.ParseFile(src)}},
	}
}
