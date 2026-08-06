package main

type structuredArgumentValue struct {
	code  int
	valid bool
}

var structuredArgumentOuter = structuredArgumentValue{code: 7, valid: true}

func structuredArgumentCheck(value structuredArgumentValue) bool {
	return value.code == 7 && value.valid
}

func appMain() int {
	structuredArgumentOuter := structuredArgumentOuter
	if !structuredArgumentCheck(structuredArgumentOuter) {
		print("FAIL: structured argument shadow initializer\n")
		return 1
	}
	print("PASS\n")
	return 0
}
