package main

var calls int

func values() ([]byte, int)       { calls++; return []byte{7, 9}, 42 }
func forwarded() (any, any)       { return values() }
func floats() (float64, bool)     { return 1.5, true }
func forwardedFloat() (any, bool) { return floats() }
func appMain(args []string) int {
	a, b := forwarded()
	bytes, ok := a.([]byte)
	if !ok || len(bytes) != 2 || bytes[1] != 9 {
		panic("slice result")
	}
	n, ok := b.(int)
	if !ok || n != 42 || calls != 1 {
		panic("integer result or repeated call")
	}
	f, ok := forwardedFloat()
	number, typed := f.(float64)
	if !ok || !typed || number != 1.5 {
		panic("float result")
	}
	print("PASS\n")
	return 0
}
