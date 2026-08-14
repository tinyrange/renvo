package main

var mirrored bool

func renvo_runtime_PrintMirror(text string) {
	if text == "PASS\n" {
		mirrored = true
	}
}

func appMain() int {
	print("PASS\n")
	if mirrored {
		return 0
	}
	return 1
}
