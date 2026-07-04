package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 19
		}
	}()
	if v == 19 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(19) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
