package main

type rtgStringBox struct {
	value string
}

func appMain(args []string) int {
	box := rtgStringBox{value: "same"}
	if box.value == "same" && box.value != "other" {
		print("PASS\n")
		return 0
	} else {
		print("FAIL\n")
		return 1
	}
}
