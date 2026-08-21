package main

type CancelFunc func()

func withCancel(value *int) (int, CancelFunc) {
	return 1, func() { *value = 42 }
}

func main() {
	value := 0
	ctx, cancel := withCancel(&value)
	cancel()
	if ctx == 1 && value == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
