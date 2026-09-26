package main

//renvo:reflect
type Record struct {
	A, B   int    `json:"value,omitempty"`
	hidden string `json:"secret"`
	Child  Plain
	Éclair string `json:"éclair"`
}

type Plain struct{ Public int }

type (
	//renvo:reflect
	Grouped struct {
		Value string `json:"group"`
	}
	Sibling struct{ Value string }
)

type field struct{ Name, Tag string }

func main() {
	checkConstruction()
	checkCollections()
	check(Record{})
	record := Record{A: 7, hidden: "private"}
	if !write(&record, 0, 42) || record.A != 42 {
		panic("set field")
	}
	value, found := read(record, 0)
	if !found {
		panic("read field")
	}
	n, okInt := value.(int)
	if !okInt || n != 42 {
		panic("field value")
	}
	if write(&record, 0, "wrong") || record.A != 42 || write(record, 0, 13) || write(&record, 4, "private") {
		panic("invalid write")
	}
	if record.hidden != "private" {
		panic("private field changed")
	}
	_, found = read(record, -1)
	if found {
		panic("invalid index")
	}
	var ptr *Record
	check(ptr)
	if write(ptr, 0, 1) {
		panic("nil pointer write")
	}
	_, found = read(ptr, 0)
	if found {
		panic("nil pointer read")
	}
	name, fields, ok := describe(Grouped{})
	if !ok || name != "Grouped" || len(fields) != 1 || fields[0].Tag != "json:\"group\"" {
		panic("grouped")
	}
	if optInEnforced {
		_, _, ok = describe(Plain{})
		if ok {
			panic("nested implicitly opted in")
		}
		_, _, ok = describe(Sibling{})
		if ok {
			panic("sibling implicitly opted in")
		}
		_, _, ok = describe(nil)
		if ok {
			panic("nil metadata")
		}
		if write(&Plain{}, 0, 1) {
			panic("unannotated write")
		}
		_, found = read(Plain{}, 0)
		if found {
			panic("unannotated read")
		}
	}
	print("PASS\n")
}

func check(value any) {
	name, fields, ok := describe(value)
	if !ok || name != "Record" || len(fields) != 4 {
		panic("record descriptor")
	}
	if fields[0].Name != "A" || fields[1].Name != "B" || fields[2].Name != "Child" || fields[3].Name != "Éclair" {
		panic("exported fields")
	}
	if fields[0].Tag != "json:\"value,omitempty\"" || fields[1].Tag != fields[0].Tag || fields[3].Tag != "json:\"éclair\"" {
		panic("tags")
	}
}
