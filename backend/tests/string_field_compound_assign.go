package main

type compoundStringBuffer struct {
	text string
}

func appendCompoundString(value *compoundStringBuffer, suffix string) {
	value.text += suffix
}

func appMain() int {
	value := &compoundStringBuffer{text: "PA"}
	appendCompoundString(value, "SS\n")
	if value.text == "PASS\n" {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
