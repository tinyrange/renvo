package main

type parserState struct {
	ok bool
}

func parse(ok bool) ([]byte, bool) {
	state := parserState{ok: ok}
	if !state.ok {
		return nil, false
	}
	return []byte{80, 65, 83, 83, 10}, true
}

func appMain() int {
	data, ok := parse(true)
	if !ok {
		print("if_not_struct_bool_field_tuple_return ok failed\n")
		return 1
	}
	print(string(data))
	return 0
}
