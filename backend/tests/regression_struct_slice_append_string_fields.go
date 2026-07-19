package main

type stringRefItem struct {
	importPath string
	name       string
	unitName   string
}

func appendStringRefItem(refs *[]stringRefItem, item stringRefItem) {
	values := *refs
	values = append(values, item)
	*refs = values
}

func appMain(args []string) int {
	var refs []stringRefItem
	item := stringRefItem{importPath: "unit", name: "Decl", unitName: "unit_Decl"}
	i := 0
	for i < 32 {
		appendStringRefItem(&refs, item)
		i++
	}
	appendStringRefItem(&refs, stringRefItem{importPath: "fmt", name: "Println", unitName: "fmt_Println"})

	if len(refs) != 33 {
		print("RENVO-1139 string struct slice append length failed\n")
		return 1
	}
	first := refs[0]
	if first.importPath != "unit" || first.name != "Decl" || first.unitName != "unit_Decl" {
		print("RENVO-1139 string struct slice append first item failed\n")
		return 1
	}
	key := first.importPath + "\x00" + first.name
	if key != "unit\x00Decl" {
		print("RENVO-1139 string struct slice append concat failed\n")
		return 1
	}
	last := refs[32]
	if last.importPath != "fmt" || last.name != "Println" || last.unitName != "fmt_Println" {
		print("RENVO-1139 string struct slice append grown item failed\n")
		return 1
	}

	print("PASS\n")
	return 0
}
