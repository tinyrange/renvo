package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 1
		}
	}()
	if v == 1 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(1) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
