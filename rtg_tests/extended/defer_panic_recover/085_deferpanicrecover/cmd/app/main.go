package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 16
		}
	}()
	if v == 16 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(16) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
