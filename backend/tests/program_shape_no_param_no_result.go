package main

var shape21Seen bool

func shape21Mark() { shape21Seen = true }
func appMain(args []string) int {
	shape21Mark()
	if !shape21Seen {
		print("program_shape_21 void\n")
		return 1
	}
	print("PASS\n")
	return 0
}
