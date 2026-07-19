package main

type ImportInfo struct {
	Path   string
	Alias  string
	Line   int
	Column int
}

type Info struct {
	Imports []ImportInfo
}

func makeInfo() Info {
	names := []string{"fmt", "os", "renvo.dev/build", "renvo.dev/check"}
	var imports []ImportInfo
	for i := 0; i < len(names); i++ {
		name := names[i]
		value := identity(name)
		info := ImportInfo{Path: value, Line: i + 1, Column: 1}
		imports = append(imports, info)
	}
	var out Info
	out.Imports = imports
	return out
}

func identity(value string) string {
	return value
}

func appMain(args []string, env []string) int {
	info := makeInfo()
	imports := info.Imports
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
