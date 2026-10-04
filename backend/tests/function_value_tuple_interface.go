package main

func tupleValuePair() (int, interface{}) { return 5, nil }
func tupleValueCallback() int            { return 0 }

func appMain() int {
	f := tupleValuePair
	n, value := f()
	if n != 5 || value != nil {
		return 1
	}
	// A local function value shadows a declaration with a different result.
	tupleValueCallback := f
	n, value = tupleValueCallback()
	if n != 5 || value != nil {
		return 2
	}
	print("PASS\n")
	return 0
}
