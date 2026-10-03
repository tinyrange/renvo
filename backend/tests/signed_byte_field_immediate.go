package main

type signedByteField struct {
	value  int8
	values []int
}

func signedByteFieldValue(values []int) signedByteField {
	return signedByteField{value: -19, values: values}
}
func appMain() int {
	values := make([]int, 1, 2)
	field := signedByteFieldValue(values)
	if field.value != -19 {
		panic("signed byte comparison")
	}
	print("PASS\n")
	return 0
}
