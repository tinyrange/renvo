package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 11
		}
	}()
	if v == 11 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(11) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
