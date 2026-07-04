package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 5
		}
	}()
	if v == 5 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(5) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
