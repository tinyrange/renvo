package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 2
		}
	}()
	if v == 2 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(2) {
		print("PASS\n")
		return
	} else {

		print("FAIL\n")
	}
}
