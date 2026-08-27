package main

var pointerCastStringArray = [1]*byte{(*byte)("A\x00")}

func appMain() int {
	if pointerCastStringArray[0] == nil || *pointerCastStringArray[0] != 'A' {
		return 1
	}
	print("PASS\n")
	return 0
}
