package main

type Aligned struct { b byte; x int64 }

func appMain(args []string) int {
	if Alignof(byte(0)) != 1 || Alignof([5]byte{}) != 1 || Alignof(Aligned{}) != Alignof(int64(0)) { return 1 }
	print("PASS\n")
	return 0
}
