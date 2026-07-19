package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 13
		}
	}()
	if v == 13 {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(13) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
