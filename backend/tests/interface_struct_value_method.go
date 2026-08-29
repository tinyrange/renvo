package main

type interfaceStructValueError struct {
	message string
	id      int
}

func (e interfaceStructValueError) Error() string { return e.message }

func interfaceStructValueMakeError() error {
	return interfaceStructValueError{message: "PASS", id: 1}
}

func appMain(args []string) int {
	if interfaceStructValueMakeError().Error() != "PASS" {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
