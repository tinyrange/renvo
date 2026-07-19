package main

type token1141 struct {
	kind   int
	text   string
	start  int32
	end    int32
	line   int32
	column int32
}

type parsedFile1141 struct {
	path          string
	source        []byte
	packageName   string
	imports       []string
	decls         []string
	tokens        []token1141
	topLevelFuncs []int
}

type file1141 struct {
	path     string
	unitPath string
	source   []byte
	parsed   parsedFile1141
}

type pkg1141 struct {
	importPath      string
	dir             string
	name            string
	entry           bool
	files           []file1141
	imports         []string
	importPositions []int
}

type graph struct {
	packages []pkg1141
}

func dependencyPackages(g *graph) []pkg1141 {
	var packages []pkg1141
	for i := 0; i < len(g.packages); i++ {
		dep := g.packages[i]
		packages = append(packages, dep)
	}
	return packages
}

func copyFiles(values []file1141) []file1141 {
	out := make([]file1141, len(values))
	for i := 0; i < len(values); i++ {
		out[i] = values[i]
	}
	return out
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
		i++
	}
	return len(a) > len(b)
}

func sortFilesByPath(files []file1141) {
	for i := 1; i < len(files); i++ {
		value := files[i]
		j := i - 1
		for j >= 0 && stringGreater(files[j].path, value.path) {
			files[j+1] = files[j]
			j--
		}
		files[j+1] = value
	}
}

func appMain() int {
	toks := []token1141{
		{kind: 1, text: "package", start: 0, end: 7, line: 1, column: 1},
		{kind: 1, text: "main", start: 8, end: 12, line: 1, column: 9},
	}
	source := []byte("package main\n")
	files := []file1141{
		{path: "z.go", unitPath: "z.go", source: source, parsed: parsedFile1141{path: "z.go", source: source, packageName: "main", tokens: toks}},
		{path: "a.go", unitPath: "a.go", source: source, parsed: parsedFile1141{path: "a.go", source: source, packageName: "main", tokens: toks}},
	}
	g := graph{packages: []pkg1141{
		{importPath: "one", dir: "/tmp/one", name: "main", files: files, imports: []string{"fmt"}, importPositions: []int{1}},
		{importPath: "two", dir: "/tmp/two", name: "main", files: files, imports: []string{"os"}, importPositions: []int{2}},
	}}

	packages := dependencyPackages(&g)
	if len(packages) != 2 {
		print("RENVO-1141 package append length failed\n")
		return 1
	}
	copied := copyFiles(packages[1].files)
	sortFilesByPath(copied)
	if len(copied) != 2 {
		print("RENVO-1141 file copy length failed\n")
		return 1
	}
	if copied[0].path != "a.go" || copied[1].path != "z.go" {
		print("RENVO-1141 file sort path corrupted\n")
		return 1
	}
	if len(copied[0].parsed.tokens) != 2 || copied[0].parsed.tokens[1].text != "main" {
		print("RENVO-1141 nested token slice corrupted\n")
		return 1
	}
	print("PASS\n")
	return 0
}
