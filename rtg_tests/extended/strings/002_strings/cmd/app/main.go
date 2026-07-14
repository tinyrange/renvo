package main

func main() {
	text := "aaPASS\nbb"
	start := len("aa")
	end := len(text) - len("bb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	} else {

		print("FAIL\n")
	}
}
