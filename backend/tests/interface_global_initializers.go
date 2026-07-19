package main

type initializerReader interface {
	Read([]byte) (int, error)
}

type initializerError struct{}

func (initializerError) Error() string { return "expected" }

var initializedError error = initializerError{}

func acceptInitializerReader(initializerReader) {}

func appMain() int {
	print("PASS\n")
	return 0
}
