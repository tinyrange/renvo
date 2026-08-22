package main

var narrowGlobalLeft int32
var narrowGlobalRight int32

func appMain(args []string) int {
	narrowGlobalRight = 7
	narrowGlobalLeft = 3
	if narrowGlobalLeft == 3 && narrowGlobalRight == 7 {
		print("PASS\n")
		return 0
	}
	return 1
}
