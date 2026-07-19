package main

type group struct {
	importPath string
}

type symbol struct {
	ImportPath string
	Name       string
	UnitName   string
}

func symbolName(importPath string, name string) string {
	return importPath + "." + name
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	c := name[0]
	return c >= 'A' && c <= 'Z'
}

func makeSymbol(g group, name string) (symbol, bool) {
	if g.importPath != "" && isExported(name) {
		return symbol{ImportPath: g.importPath, Name: name, UnitName: symbolName(g.importPath, name)}, true
	}
	return symbol{}, false
}

func appMain(args []string, env []string) int {
	sym, ok := makeSymbol(group{importPath: "pkg"}, "Name")
	if !ok {
		return 1
	}
	if sym.ImportPath != "pkg" {
		return 1
	}
	if sym.Name != "Name" {
		return 1
	}
	if sym.UnitName != "pkg.Name" {
		return 1
	}
	print("PASS\n")
	return 0
}
