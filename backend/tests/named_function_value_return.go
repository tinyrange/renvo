package main

type returnedCancelFunc func()

func returnedCancel(value *int) returnedCancelFunc {
	return func() { *value = 42 }
}

func appMain() int {
	value := 0
	cancel := returnedCancel(&value)
	cancel()
	if value == 42 {
		print("PASS\n")
		return 0
	}
	return 1
}
