package main

type textError string

func (err textError) Error() string {
	return string(err)
}

func makeError() error {
	return textError("PASS")
}

func message(err error) string {
	msg := err.Error()
	return msg
}

func appMain(args []string) int {
	if makeError() == nil {
		print("FAIL nil\n")
		return 1
	}
	if message(makeError()) != "PASS" {
		print("FAIL message\n")
		return 1
	}
	print("PASS\n")
	return 0
}
