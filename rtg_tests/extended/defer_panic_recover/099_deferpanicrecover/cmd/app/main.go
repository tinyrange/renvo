package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 7
		}
	}()
	if v == 7 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(7) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
