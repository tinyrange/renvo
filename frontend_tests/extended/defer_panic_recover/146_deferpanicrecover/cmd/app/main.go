package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 8
		}
	}()
	if v == 8 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(8) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
