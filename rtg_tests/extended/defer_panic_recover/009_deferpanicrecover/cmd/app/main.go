package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 9
		}
	}()
	if v == 9 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(9) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
