package main

const fixedPrunedOptionalTarget = 1

var renvoFixedTarget = fixedPrunedOptionalTarget

func fixedPrunedOptionalBackendCall() bool {
	if renvoFixedTarget != 0 {
		return true
	}
	return missingOptionalBackendCall()
}

func appMain(args []string) int {
	if fixedPrunedOptionalBackendCall() {
		print("PASS\n")
		return 0
	}
	return 1
}
