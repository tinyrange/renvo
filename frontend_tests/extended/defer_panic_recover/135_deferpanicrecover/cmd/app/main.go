package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 20
		}
	}()
	if v == 20 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(20) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
