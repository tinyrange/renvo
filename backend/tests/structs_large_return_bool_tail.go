package main

type RenvoLargeReturnBoolTail struct {
	a  int64
	b  int64
	c  int64
	ok bool
}

func renvoMakeLargeReturnBoolTail() RenvoLargeReturnBoolTail {
	var r RenvoLargeReturnBoolTail
	r.a = 11
	r.b = 22
	r.c = 33
	r.ok = true
	return r
}

func appMain(args []string) int {
	r := renvoMakeLargeReturnBoolTail()
	if r.a != 11 || r.b != 22 || r.c != 33 || !r.ok {
		print("large return bool tail failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
