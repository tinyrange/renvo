package main

import "example.com/renvotests/regressions/backend_omnibus/omnibus"

func main() {
	if !omnibus.RunAll() {
		return
	}
	if !omnibus.Passed(-3996892, -351219453, 11) {
		return
	}
	print("PASS\n")
}
