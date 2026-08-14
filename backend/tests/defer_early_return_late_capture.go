package main

type earlyError string

func (err earlyError) Error() string { return string(err) }

func returnBeforeCapture(stop bool) error {
	if stop {
		return earlyError("expected")
	}
	value := 7
	defer func() {
		if value != 7 {
			print("FAIL capture\n")
		}
	}()
	return nil
}

func appMain() int {
	err := returnBeforeCapture(true)
	if err == nil || err.Error() != "expected" {
		print("FAIL error\n")
		return 1
	}
	print("PASS\n")
	return 0
}
