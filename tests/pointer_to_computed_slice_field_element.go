package main

type rtg0717Field struct {
	nameStart int
	nameEnd   int
	typ       int
	offset    int
}

type rtg0717Type struct {
	first int
	count int
}

type rtg0717Meta struct {
	fields []rtg0717Field
}

func rtg0717Find(meta *rtg0717Meta, typ rtg0717Type, wantStart int, wantEnd int) int {
	for i := 0; i < typ.count; i++ {
		field := &meta.fields[typ.first+i]
		if field.nameStart == wantStart && field.nameEnd == wantEnd {
			return i
		}
	}
	return -1
}

func appMain(args []string, env []string) int {
	var meta rtg0717Meta
	meta.fields = append(meta.fields, rtg0717Field{nameStart: 10, nameEnd: 14, typ: 1, offset: 2})
	meta.fields = append(meta.fields, rtg0717Field{nameStart: 20, nameEnd: 24, typ: 3, offset: 4})
	typ := rtg0717Type{first: 0, count: 2}
	if rtg0717Find(&meta, typ, 20, 24) != 1 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
