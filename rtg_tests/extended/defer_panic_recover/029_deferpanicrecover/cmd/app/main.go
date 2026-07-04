package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 6
		}
	}()
	if v == 6 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(6) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
