package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 3
		}
	}()
	if v == 3 {
		panic("expected")
	}
	return false
}

func main() {
	for corpusAttempt := 0; corpusAttempt < 1; corpusAttempt++ {
		if guarded(3) {
			print("PASS\n")
			return
		}
	}

	print("FAIL\n")
}
