package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 18
		}
	}()
	if v == 18 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(18) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
