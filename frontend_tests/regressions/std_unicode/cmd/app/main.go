package main

import "unicode"

func main() {
	if !unicode.IsLetter('世') || !unicode.IsDigit('٧') || !unicode.IsSpace(0x2003) || unicode.ToLower('Ω') != 'ω' || unicode.ToUpper('ж') != 'Ж' {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
