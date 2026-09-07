package main

func bytesMatch(value any) bool { _, ok := value.([]byte); return ok }
func intsMatch(value any) bool  { _, ok := value.([]int); return ok }
func valid(data []byte) bool    { return string(data) == "null" }

func appMain(args []string) int {
	if !valid([]byte("null")) {
		panic("conversion")
	}
	if !bytesMatch([]byte{}) || !bytesMatch([]byte("text")) {
		panic("byte slice identity")
	}
	var bytes [2]byte
	view := bytes[:]
	if !bytesMatch(view) || !bytesMatch([]byte{}) {
		panic("byte array slice identity")
	}
	var numbers [2]int
	items := numbers[:]
	if !intsMatch(items) || !intsMatch([]int{}) {
		panic("integer array slice identity")
	}
	print("PASS\n")
	return 0
}
