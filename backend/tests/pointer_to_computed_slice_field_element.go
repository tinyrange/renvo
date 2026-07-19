package main

type renvo0717Field struct {
	nameStart int
	nameEnd   int
	typ       int
	offset    int
}

type renvo0717Type struct {
	first int
	count int
}

type renvo0717Meta struct {
	fields []renvo0717Field
}

func renvo0717Find(meta *renvo0717Meta, typ renvo0717Type, wantStart int, wantEnd int) int {
	for i := 0; i < typ.count; i++ {
		field := &meta.fields[typ.first+i]
		if field.nameStart == wantStart && field.nameEnd == wantEnd {
			return i
		}
	}
	return -1
}

func appMain(args []string, env []string) int {
	var meta renvo0717Meta
	meta.fields = append(meta.fields, renvo0717Field{nameStart: 10, nameEnd: 14, typ: 1, offset: 2})
	meta.fields = append(meta.fields, renvo0717Field{nameStart: 20, nameEnd: 24, typ: 3, offset: 4})
	typ := renvo0717Type{first: 0, count: 2}
	if renvo0717Find(&meta, typ, 20, 24) != 1 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
