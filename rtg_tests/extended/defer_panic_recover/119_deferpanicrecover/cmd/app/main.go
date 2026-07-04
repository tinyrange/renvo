package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 4
		}
	}()
	if v == 4 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(4) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
