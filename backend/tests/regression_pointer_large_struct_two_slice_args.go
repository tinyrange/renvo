package main

type token1142 struct {
	kind   int
	text   string
	start  int32
	end    int32
	line   int32
	column int32
}

type decl1142 struct {
	kind     string
	name     string
	names    []string
	tok      token1142
	nameTok  token1142
	nameToks []token1142
	receiver bool
	start    int
	end      int
}

type parsed1142 struct {
	path          string
	source        []byte
	packageName   string
	imports       []string
	decls         []decl1142
	tokens        []token1142
	topLevelFuncs []int
}

type symbol1142 struct {
	name     string
	unitName string
}

type info1142 struct {
	name string
	elem string
}

func appendInfo1142(file *parsed1142, names []symbol1142, out []info1142) []info1142 {
	if file == nil {
		return append(out, info1142{name: "nil", elem: "file"})
	}
	if len(file.tokens) != 2 || file.tokens[1].text != "main" {
		return append(out, info1142{name: "bad", elem: "tokens"})
	}
	if len(names) != 1 || names[0].name != "T" || names[0].unitName != "pkg_T" {
		return append(out, info1142{name: "bad", elem: "names"})
	}
	return append(out, info1142{name: names[0].name, elem: file.tokens[1].text})
}

func appMain() int {
	file := parsed1142{
		path:        "x.go",
		source:      []byte("package main\n"),
		packageName: "main",
		tokens: []token1142{
			{kind: 1, text: "package", start: 0, end: 7, line: 1, column: 1},
			{kind: 1, text: "main", start: 8, end: 12, line: 1, column: 9},
		},
	}
	names := []symbol1142{{name: "T", unitName: "pkg_T"}}
	var out []info1142
	out = appendInfo1142(&file, names, out)
	if len(out) == 1 && out[0].name == "T" && out[0].elem == "main" {
		print("PASS\n")
		return 0
	}
	print("RENVO-1142 pointer large struct two slice args failed\n")
	return 1
}
