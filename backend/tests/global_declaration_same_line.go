package main

type record struct{ value int32 }; var first = record{value: 3}; var next = func() int32 { value := first.value; return value + 4 };

func appMain() int {
	if next() != 7 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
