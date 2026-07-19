package main

type ImportInfo struct {
	Path   string
	Alias  string
	Line   int
	Column int
}

func makeImports() []ImportInfo {
	names := []string{"fmt", "os", "renvo.dev/build", "renvo.dev/check"}
	var imports []ImportInfo
	for i := 0; i < len(names); i++ {
		name := names[i]
		value, _ := identity(name)
		info := ImportInfo{Path: value, Line: i + 1, Column: 1}
		imports = append(imports, info)
	}
	return imports
}

func identity(value string) (string, error) {
	return value, nil
}

func appMain(args []string, env []string) int {
	imports := makeImports()
	if len(imports) != 4 {
		print("FAIL len\n")
		return 1
	}
	first := imports[0]
	if first.Path != "fmt" {
		print("FAIL first\n")
		return 1
	}
	last := imports[3]
	if last.Path != "renvo.dev/check" {
		print("FAIL last\n")
		return 1
	}
	print("PASS\n")
	return 0
}
