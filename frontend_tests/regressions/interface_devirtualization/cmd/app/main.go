package main

type valueReader interface {
	Value(delta int) int
}

type item struct {
	value int
}

func (value item) Value(delta int) int {
	return value.value + delta
}

// read deliberately shadows the backend's read builtin. User declarations
// must take precedence over compiler intrinsics during call resolution.
func read(value valueReader) int {
	return value.Value(1)
}

func main() {
	var value valueReader = item{value: 42}
	asserted, ok := value.(item)
	if !ok || asserted.value != 42 || read(value) != 43 {
		print("FAIL\n")
		return
	}
	switch value.(type) {
	case item:
		print("PASS\n")
	default:
		print("FAIL\n")
	}
}
