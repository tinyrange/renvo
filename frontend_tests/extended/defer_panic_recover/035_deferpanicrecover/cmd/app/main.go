package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 12
		}
	}()
	if v == 12 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(12) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
