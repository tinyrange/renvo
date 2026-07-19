package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 14
		}
	}()
	if v == 14 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(14) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
