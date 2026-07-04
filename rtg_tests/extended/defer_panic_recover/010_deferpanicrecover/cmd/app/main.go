package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 10
		}
	}()
	if v == 10 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(10) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
