package main

type Info struct {
	Name    string
	Imports []ImportInfo
}

type ImportInfo struct {
	Path  string
	Alias string
}

func makeInfo() Info {
	var imports []ImportInfo
	first := ImportInfo{Path: "j5.nz/rtg/rtg/build"}
	second := ImportInfo{Path: "j5.nz/rtg/rtg/check"}
	imports = append(imports, first)
	imports = append(imports, second)
	return Info{Name: "main", Imports: imports}
}

func appMain(args []string, env []string) int {
	info := makeInfo()
	if len(info.Imports) != 2 {
		print("FAIL len\n")
		return 1
	}
	imports := info.Imports
	first := imports[0]
	if first.Path != "j5.nz/rtg/rtg/build" {
		print("FAIL first\n")
		return 1
	}
	second := imports[1]
	if second.Path != "j5.nz/rtg/rtg/check" {
		print("FAIL second\n")
		return 1
	}
	print("PASS\n")
	return 0
}
