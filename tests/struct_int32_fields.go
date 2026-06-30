package main

type int32Record struct {
	a int32
	b int32
	c int32
	d int32
}

func appMain(args []string) int {
	r := int32Record{a: 11, b: 22, c: 33, d: 44}
	if r.a != 11 {
		print("FAIL\n")
		return 1
	}
	if r.b != 22 {
		print("FAIL\n")
		return 1
	}
	if r.c != 33 {
		print("FAIL\n")
		return 1
	}
	if r.d != 44 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
