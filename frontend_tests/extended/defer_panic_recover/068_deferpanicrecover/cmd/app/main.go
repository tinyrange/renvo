package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 22
		}
	}()
	if v == 22 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(22) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
