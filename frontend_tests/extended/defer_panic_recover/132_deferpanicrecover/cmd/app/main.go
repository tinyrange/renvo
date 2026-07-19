package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 17
		}
	}()
	if v == 17 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(17) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
