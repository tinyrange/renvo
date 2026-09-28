package main

type NamedInts []int

//renvo:reflect
type Collections struct {
	Numbers []int
	Rows    [][]string
	Lookup  map[string]Record
	Fixed   [2]int
	Pointer *Record
	Defined NamedInts
	Bytes   []byte
	Octets  []uint8
}

type collection struct {
	Kind    string
	Nil     bool
	Keys    []any
	Values  []any
	Element any
}

func checkCollections() {
	original := []int{1, 2}
	info, ok := inspect(original)
	if !ok || info.Kind != "slice" || info.Nil || len(info.Values) != 2 {
		panic("slice snapshot")
	}
	item, ok := info.Values[1].(int)
	if !ok || item != 2 {
		panic("slice item")
	}
	result, ok := rebuild(original, nil, []any{4, 5}, false)
	if !ok {
		panic("rebuild slice")
	}
	numbers, ok := result.([]int)
	if !ok || len(numbers) != 2 || numbers[0] != 4 || original[0] != 1 {
		panic("slice reconstruction")
	}
	_, ok = rebuild(original, nil, []any{"wrong"}, false)
	if ok {
		panic("slice wrong type")
	}
	var empty []int
	info, ok = inspect(empty)
	if !ok || !info.Nil {
		panic("nil slice")
	}
	result, ok = rebuild(empty, nil, nil, false)
	if !ok {
		panic("empty rebuild")
	}
	numbers, ok = result.([]int)
	if !ok || numbers == nil || len(numbers) != 0 {
		panic("empty not nil")
	}
	result, ok = rebuild(original, nil, nil, true)
	if !ok {
		panic("nil rebuild")
	}
	numbers, ok = result.([]int)
	if !ok || numbers != nil {
		panic("nil rebuilt slice")
	}
	result, ok = rebuild(NamedInts{}, nil, []any{9}, false)
	defined, namedOK := result.(NamedInts)
	if !ok || !namedOK || len(defined) != 1 || defined[0] != 9 {
		panic("defined slice")
	}
	result, ok = rebuild([][]string{}, nil, []any{[]string{"nested"}}, false)
	rows, rowOK := result.([][]string)
	if !ok || !rowOK || len(rows) != 1 || len(rows[0]) != 1 || rows[0][0] != "nested" {
		panic("nested slices")
	}
	info, ok = inspect([]byte{1, 2})
	if !ok || len(info.Values) != 2 {
		panic("byte alias collection")
	}
	result, ok = rebuild([2]int{}, nil, []any{6, 7}, false)
	fixed, arrayOK := result.([2]int)
	if !ok || !arrayOK || fixed[1] != 7 {
		panic("array reconstruction")
	}
	_, ok = rebuild([2]int{}, nil, []any{6}, false)
	if ok {
		panic("array wrong length")
	}
	mapping := map[string]Record{"key": {A: 3}}
	info, ok = inspect(mapping)
	if !ok || info.Kind != "map" || info.Nil || len(info.Keys) != 1 || len(info.Values) != 1 {
		panic("map snapshot")
	}
	result, ok = rebuild(mapping, []any{"new"}, []any{Record{A: 8}}, false)
	rebuilt, mapOK := result.(map[string]Record)
	if !ok || !mapOK || len(rebuilt) != 1 || rebuilt["new"].A != 8 || mapping["key"].A != 3 {
		panic("map reconstruction")
	}
	var pointer *Record
	info, ok = inspect(pointer)
	if !ok || info.Kind != "pointer" || !info.Nil {
		panic("nil pointer snapshot")
	}
	_, elementOK := info.Element.(Record)
	if !elementOK {
		panic("pointer element prototype")
	}
	result, ok = rebuild(pointer, nil, []any{Record{A: 11}}, false)
	built, pointerOK := result.(*Record)
	if !ok || !pointerOK || built == nil || built.A != 11 {
		panic("pointer reconstruction")
	}
}
