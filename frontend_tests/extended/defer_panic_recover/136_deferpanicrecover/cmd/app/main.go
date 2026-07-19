package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 21
		}
	}()
	if v == 21 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(21) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
