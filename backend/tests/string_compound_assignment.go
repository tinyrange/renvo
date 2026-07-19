package main

type stringCompoundBox struct {
	value string
}

var stringCompoundGlobal = "a"

func appMain() int {
	local := "a"
	local += "b"
	box := stringCompoundBox{value: "a"}
	box.value += "b"
	stringCompoundGlobal += "b"
	if local == "ab" && box.value == "ab" && stringCompoundGlobal == "ab" {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
