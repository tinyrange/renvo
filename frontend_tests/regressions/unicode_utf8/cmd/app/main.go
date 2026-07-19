package main

import "unicode/utf8"

func main() {
	r, size := utf8.DecodeRuneInString("éx")
	if r == 'é' && size == 2 && utf8.RuneCountInString("Aé世😀") == 4 && utf8.ValidString("Aé世😀") && !utf8.ValidString("\xff") {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
