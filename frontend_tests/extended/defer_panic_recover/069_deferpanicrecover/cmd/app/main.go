package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 0
		}
	}()
	if v == 0 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(0) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
