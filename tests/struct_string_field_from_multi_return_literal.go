package main

type result struct {
	value int
	kind  string
}

func makeResult() (result, bool) {
	return result{value: 7, kind: "return"}, true
}

func appMain() int {
	r, ok := makeResult()
	if !ok {
		print("struct_string_field_from_multi_return_literal ok failed\n")
		return 1
	}
	if r.value != 7 {
		print("struct_string_field_from_multi_return_literal value failed\n")
		return 1
	}
	if r.kind != "return" {
		print("struct_string_field_from_multi_return_literal kind failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
