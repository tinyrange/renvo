package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 1
		}
	}()
	if v == 1 {
		panic("expected")
	}
	return false
}

func main() {
	corpusOK := guarded(1)
	if !corpusOK {

		print("FAIL\n")
		return
	}
	print("PASS\n")

}
