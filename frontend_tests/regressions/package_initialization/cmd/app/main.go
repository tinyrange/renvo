package main

import "example.com/renvotests/regressions/package_initialization/state"

var rootTrace int

func init() {
	if state.Trace == 912 {
		rootTrace = 3
	}
}

func init() { rootTrace = rootTrace*10 + 4 }

func main() {
	if state.Trace == 912 && rootTrace == 34 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
