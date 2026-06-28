package main

type Package struct {
	ImportPath string
	Dir        string
	Name       string
	Imports    []string
	Positions  []Position
}

type Position struct {
	ImportPath string
	Path       string
	Line       int
	Column     int
}

func add(pkg *Package, importSet *[]string, value string) {
	impPath := copyString(value)
	values := *importSet
	if !contains(values, impPath) {
		values = append(values, impPath)
		*importSet = values
		pkg.Imports = append(pkg.Imports, impPath)
	}
	if !hasPosition(pkg.Positions, impPath) {
		pkg.Positions = append(pkg.Positions, Position{ImportPath: impPath, Path: "main.go", Line: 1, Column: 8})
	}
}

func contains(values []string, value string) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return true
		}
	}
	return false
}

func hasPosition(values []Position, value string) bool {
	for i := 0; i < len(values); i++ {
		if values[i].ImportPath == value {
			return true
		}
	}
	return false
}

func copyString(value string) string {
	var out []byte
	for i := 0; i < len(value); i++ {
		out = append(out, value[i])
	}
	return string(out)
}

func appMain(args []string, env []string) int {
	var pkg Package
	var importSet []string
	add(&pkg, &importSet, "fmt")
	add(&pkg, &importSet, "j5.nz/rtg/rtg/load")
	if !contains(pkg.Imports, "fmt") {
		print("FAIL\n")
		return 1
	}
	if !contains(pkg.Imports, "j5.nz/rtg/rtg/load") {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
