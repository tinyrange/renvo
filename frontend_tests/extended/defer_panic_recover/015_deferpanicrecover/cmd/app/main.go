package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 15
		}
	}()
	if v == 15 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(15) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
